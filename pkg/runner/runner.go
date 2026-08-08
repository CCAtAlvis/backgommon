// Package runner orchestrates the backtest event loop. It feeds market data
// bar-by-bar to a user-defined Strategy while coordinating portfolio accounting,
// risk management, fill pricing, and report generation. Callers construct a
// Runner via [New] with functional [Option] values, then call [Runner.Start].
package runner

import (
	"fmt"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/execution"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
	"github.com/CCAtAlvis/backgommon/pkg/outmgr"
	"github.com/CCAtAlvis/backgommon/pkg/output"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

// dayLifecycle is an optional interface detected via type assertion on the
// strategy at runtime. If the strategy satisfies it (e.g. by embedding
// strategy.BaseStrategy), the runner calls OnDayStart/OnDayEnd at calendar-day
// boundaries so strategies can perform daily housekeeping like rebalancing or
// logging.
type dayLifecycle interface {
	OnDayStart(time.Time)
	OnDayEnd(time.Time)
}

// Runner is the top-level backtest orchestrator. It iterates over a
// TimeseriesTable of Candle data, invoking the Strategy on each bar and routing
// the resulting orders through risk validation, fill pricing, and portfolio
// processing.
//
// Configure a Runner entirely through functional options passed to [New] — see
// the With* helpers in options.go for the full set of knobs.
type Runner struct {
	// Core components
	Strategy    interfaces.Strategy
	Portfolio   interfaces.PortfolioManager
	RiskManager interfaces.RiskManager

	// Data handling
	Data        *types.TimeseriesTable[core.Candle]
	CurrentTime time.Time

	// Results and analytics
	Results     *types.Results
	EquityCurve []types.AccountValue

	// preRunIndicators are applied to Data at the start of Start() when non-empty.
	preRunIndicators []interfaces.Indicator

	fillPricer interfaces.FillPricer
	slippage   execution.SlippageModel

	outputDir      string
	fixedOutputDir bool        // when true, export directly into outputDir (no run_<ts> subfolder)
	reportSettings interface{} // JSON-serializable settings for config.json
	exportOpts     output.ExportOptions
	startTime      time.Time // Skip processing rows before this time (data still available for lookbacks)
	endTime        time.Time // Stop processing rows after this time
	lastDay        time.Time

	// strictOrderProcessing when true aborts the backtest on the first risk or
	// ProcessOrder failure. Default false: skip the failed order, log a warning,
	// and continue (skippedOrders counts how many were dropped).
	strictOrderProcessing bool
	skippedOrders         int
}

// Option is a functional option that configures a [Runner] before the backtest
// starts. See the With* constructors in options.go for the full catalogue.
type Option func(*Runner)

// New creates a Runner for the given Strategy and applies every supplied Option.
// At minimum callers should provide WithData, WithPortfolio, and WithRiskManager;
// Start will return an error if any required component is missing.
func New(strategy interfaces.Strategy, opts ...Option) *Runner {
	r := &Runner{
		Strategy:    strategy,
		EquityCurve: make([]types.AccountValue, 0),
	}

	// Apply options
	for _, opt := range opts {
		opt(r)
	}

	r.Strategy.SetPortfolio(r.Portfolio)
	return r
}

// Start runs the backtest event loop from the first to the last data bar
// (respecting WithStartTime/WithEndTime bounds). For each bar it applies
// periodic events, day-lifecycle hooks, risk checks, and strategy signals in
// that order. After the loop it finalizes metrics and exports reports (if
// WithOutputDir was set). Returns nil on success.
func (r *Runner) Start() error {
	if err := r.validateComponents(); err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	if len(r.preRunIndicators) > 0 {
		if err := r.Data.ApplyIndicators(r.preRunIndicators); err != nil {
			return fmt.Errorf("apply indicators: %w", err)
		}
	}

	rows := r.Data.Rows()
	nCols := len(r.Data.Cols())
	// Reuse per-bar maps across the loop. GetRow previously allocated two large
	// maps every bar (2030+ symbols), which dominated CPU via GC under parallel sweeps.
	cur := make(map[string]core.Candle, nCols)
	nxt := make(map[string]core.Candle, nCols)
	prices := make(map[string]float64, nCols)

	for i, row := range rows {
		if !r.startTime.IsZero() && row.Timestamp.Before(r.startTime) {
			continue
		}
		if !r.endTime.IsZero() && row.Timestamp.After(r.endTime) {
			break
		}

		if !r.Data.FillRow(row.Timestamp, cur) {
			return fmt.Errorf("data error at %v", row.Timestamp)
		}

		var next map[string]core.Candle
		if i+1 < len(rows) {
			if !r.Data.FillRow(rows[i+1].Timestamp, nxt) {
				return fmt.Errorf("data error at next bar %v", rows[i+1].Timestamp)
			}
			next = nxt
		}

		r.CurrentTime = row.Timestamp
		if err := r.processTick(cur, next, prices); err != nil {
			return fmt.Errorf("processing error at %v: %w", row.Timestamp, err)
		}

		r.updateEquityCurve()
	}

	r.finalizeResults()
	r.exportReports()
	return nil
}

// processTick handles a single bar: updates positions at current prices,
// evaluates drawdown/position exits via the risk manager, then calls the
// strategy's OnTick for new signals.
func (r *Runner) processTick(data, next map[string]core.Candle, prices map[string]float64) error {
	r.runPeriodicEvents()
	r.invokeDayLifecycle()

	fillPrices(data, prices)
	r.Portfolio.UpdatePositions(prices)

	dd := r.RiskManager.CheckDrawdown(r.Portfolio, r.CurrentTime)
	if dd.LiquidateAll {
		liquidation := r.liquidationOrders()
		if err := r.applyFillPrices(liquidation, data, next); err != nil {
			return err
		}
		if err := r.processOrders(liquidation); err != nil {
			return err
		}
		return nil
	}

	exitOrders := r.RiskManager.CheckPositionExits(r.Portfolio, prices)
	if err := r.applyFillPrices(exitOrders, data, next); err != nil {
		return err
	}
	if len(exitOrders) > 0 {
		if err := r.processOrders(exitOrders); err != nil {
			return err
		}
	}

	orders := r.Strategy.OnTick(data)
	if dd.BlockNewEntries {
		orders = filterEntryOrders(orders)
	}
	if err := r.applyFillPrices(orders, data, next); err != nil {
		return err
	}
	if len(orders) > 0 {
		if err := r.processOrders(orders); err != nil {
			return err
		}
	}

	return nil
}

func (r *Runner) runPeriodicEvents() {
	if p, ok := r.Portfolio.(*portfolio.Portfolio); ok {
		p.ProcessPeriodicEvents(r.CurrentTime)
	}
}

// invokeDayLifecycle detects calendar-day boundaries by comparing r.CurrentTime
// to r.lastDay. On a new day it calls OnDayEnd for the previous day and
// OnDayStart for the new one. The strategy must satisfy the dayLifecycle
// interface (e.g. by embedding strategy.BaseStrategy) — otherwise this is a no-op.
func (r *Runner) invokeDayLifecycle() {
	dl, ok := r.Strategy.(dayLifecycle)
	if !ok {
		return
	}

	day := r.CurrentTime.Truncate(24 * time.Hour)
	if r.lastDay.IsZero() {
		r.lastDay = day
		dl.OnDayStart(r.CurrentTime)
		return
	}
	if day.After(r.lastDay) {
		dl.OnDayEnd(r.lastDay.Add(23*time.Hour + 59*time.Minute))
		dl.OnDayStart(r.CurrentTime)
		r.lastDay = day
	}
}

func (r *Runner) liquidationOrders() []portfolio.Order {
	var orders []portfolio.Order
	for _, pos := range r.Portfolio.Positions() {
		orders = append(orders, portfolio.ExitOrderForPosition(pos))
	}
	return orders
}

func filterEntryOrders(orders []portfolio.Order) []portfolio.Order {
	filtered := make([]portfolio.Order, 0, len(orders))
	for _, ord := range orders {
		if ord.Type != portfolio.Entry {
			filtered = append(filtered, ord)
		}
	}
	return filtered
}

// processOrders handles a batch of orders. In lenient mode (default), a single
// order failure is logged and skipped; in strict mode the first error aborts.
func (r *Runner) processOrders(orders []portfolio.Order) error {
	for _, ord := range orders {
		if err := r.processOrder(ord); err != nil {
			if r.strictOrderProcessing {
				return err
			}
			r.skippedOrders++
			outmgr.Printf("WARN: skipping order type=%v side=%s %s qty=%d: %v\n",
				ord.Type, ord.Side.String(), ord.Instrument, ord.Quantity, err)
		}
	}
	return nil
}

// processOrder handles a single order
func (r *Runner) processOrder(ord portfolio.Order) error {
	if ord.FilledAt.IsZero() {
		ord.FilledAt = r.CurrentTime
	}

	// Validate against risk settings
	if err := r.RiskManager.ValidateOrder(r.Portfolio, ord); err != nil {
		return fmt.Errorf("risk validation failed: %w", err)
	}

	// Process the order
	if err := r.Portfolio.ProcessOrder(ord); err != nil {
		return fmt.Errorf("order processing failed: %w", err)
	}

	// Notify strategy
	r.Strategy.OnOrderFilled(ord)

	return nil
}

// SkippedOrders returns how many orders were dropped in lenient mode.
func (r *Runner) SkippedOrders() int {
	return r.skippedOrders
}

// Helper functions

func (r *Runner) validateComponents() error {
	if r.Strategy == nil {
		return fmt.Errorf("strategy not set")
	}
	if r.Portfolio == nil {
		return fmt.Errorf("portfolio not set")
	}
	if r.RiskManager == nil {
		return fmt.Errorf("risk manager not set")
	}
	if r.Data == nil {
		return fmt.Errorf("data not set")
	}
	return nil
}

func getCurrentPrices(data map[string]core.Candle) map[string]float64 {
	prices := make(map[string]float64, len(data))
	fillPrices(data, prices)
	return prices
}

func fillPrices(data map[string]core.Candle, prices map[string]float64) {
	for symbol, candle := range data {
		prices[symbol] = candle.Close
	}
}

// updateEquityCurve snapshots the portfolio state at the current bar:
// total value, cash, open position count, unrealized PnL, and per-position
// holdings (instrument, quantity, last price, avg entry). These snapshots
// drive the equity chart and drawdown analysis in the HTML report.
func (r *Runner) updateEquityCurve() {
	positions := r.Portfolio.Positions()
	holdings := make([]types.HoldingSnapshot, 0, len(positions))
	for _, pos := range positions {
		lastPrice := pos.OpenPrice
		if pos.Quantity > 0 && pos.UnrealizedPnL != 0 {
			lastPrice = pos.OpenPrice + (pos.UnrealizedPnL / float64(pos.Quantity))
		}
		holdings = append(holdings, types.HoldingSnapshot{
			Instrument: pos.Instrument,
			Quantity:   pos.Quantity,
			Price:      lastPrice,
			AvgEntry:   pos.OpenPrice,
		})
	}

	r.EquityCurve = append(r.EquityCurve, types.AccountValue{
		Time:          r.CurrentTime,
		Value:         r.Portfolio.Value(),
		Cash:          r.Portfolio.Cash(),
		OpenPositions: len(positions),
		UnrealizedPnL: calculateUnrealizedPnL(r.Portfolio),
		Holdings:      holdings,
	})
}

func calculateUnrealizedPnL(p interfaces.PortfolioManager) float64 {
	var total float64
	for _, pos := range p.Positions() {
		total += pos.UnrealizedPnL
	}
	return total
}

func (r *Runner) exportReports() {
	if r.outputDir == "" {
		return
	}

	opts := r.exportOpts
	if opts == (output.ExportOptions{}) {
		if r.fixedOutputDir {
			// Sweep trials: zstd only (shared viewer written by sweep package).
			opts = output.ExportOptions{Compress: output.CompressZstd}
		} else {
			// Single run: zstd + offline data.js + local viewer.
			opts = output.ExportOptions{
				Compress:      output.CompressZstd,
				OfflineDataJS: true,
				WriteViewer:   true,
			}
		}
	}

	data := output.ReportData{
		Results:         r.Results,
		EquityCurve:     r.EquityCurve,
		ClosedPositions: r.Portfolio.ClosedPositions(),
		OpenPositions:   r.Portfolio.Positions(),
		Settings:        r.reportSettings,
		Options:         opts,
	}

	if r.fixedOutputDir {
		if err := output.ExportRunTo(r.outputDir, data); err != nil {
			outmgr.Printf("Report export error: %v\n", err)
			return
		}
		outmgr.Printf("Reports written to %s\n", r.outputDir)
		return
	}

	runDir, err := output.ExportRun(r.outputDir, data)
	if err != nil {
		outmgr.Printf("Report export error: %v\n", err)
		return
	}
	outmgr.Printf("Reports written to %s\n", runDir)
}

// Run executes the backtest. Prefer Start() — Run exists for backward compatibility.
func (r *Runner) Run(data *types.TimeseriesTable[core.Candle]) error {
	if data != nil {
		r.Data = data
	}
	return r.Start()
}
