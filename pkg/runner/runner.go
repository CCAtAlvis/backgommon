package runner

import (
	"fmt"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

// Runner handles the core backtesting logic
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

	// Indicator configuration
	IndicatorConfig *IndicatorConfig
}

// Option defines a function that modifies Runner
type Option func(*Runner)

// New creates a new runner with options
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

// Start begins the backtest
func (r *Runner) Start() error {
	// Initialize components
	if err := r.validateComponents(); err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	for _, row := range r.Data.Rows() {
		data, ok := row.Get()
		if !ok {
			return fmt.Errorf("data error at %v", row.Timestamp)
		}

		r.CurrentTime = row.Timestamp
		if err := r.processTick(data); err != nil {
			return fmt.Errorf("processing error at %v: %w", row.Timestamp, err)
		}

		r.updateEquityCurve()
	}

	return nil
}

// processTick handles a single tick of data
func (r *Runner) processTick(data map[string]core.Candle) error {
	// Update portfolio with current prices
	prices := getCurrentPrices(data)
	r.Portfolio.UpdatePositions(prices)

	// Check for position exits
	exitOrders := r.RiskManager.CheckPositionExits(r.Portfolio, prices)
	if len(exitOrders) > 0 {
		if err := r.processOrders(exitOrders); err != nil {
			return err
		}
	}

	// Get new orders from strategy
	orders := r.Strategy.OnTick(data)
	if len(orders) > 0 {
		if err := r.processOrders(orders); err != nil {
			return err
		}
	}

	return nil
}

// processOrders handles a batch of orders
func (r *Runner) processOrders(orders []portfolio.Order) error {
	for _, ord := range orders {
		if err := r.processOrder(ord); err != nil {
			return err
		}
	}
	return nil
}

// processOrder handles a single order
func (r *Runner) processOrder(ord portfolio.Order) error {
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
	prices := make(map[string]float64)
	for symbol, candle := range data {
		prices[symbol] = candle.Close
	}
	return prices
}

func (r *Runner) updateEquityCurve() {
	r.EquityCurve = append(r.EquityCurve, types.AccountValue{
		Time:          r.CurrentTime,
		Value:         r.Portfolio.Value(),
		Cash:          r.Portfolio.Cash(),
		OpenPositions: len(r.Portfolio.Positions()),
		UnrealizedPnL: calculateUnrealizedPnL(r.Portfolio),
	})
}

func calculateUnrealizedPnL(p interfaces.PortfolioManager) float64 {
	var total float64
	for _, pos := range p.Positions() {
		total += pos.UnrealizedPnL
	}
	return total
}

// IndicatorConfig holds configuration for indicator calculation
type IndicatorConfig struct {
	Indicators   []interfaces.Indicator
	LookbackSize int
}

// preCalculateIndicators calculates indicator values for the current window
func (r *Runner) preCalculateIndicators(data map[string]*core.Candle, historicalData map[string][]core.Candle) error {
	if r.IndicatorConfig == nil {
		return nil
	}

	cfg := r.IndicatorConfig

	// Calculate indicators for each symbol
	for symbol, candle := range data {
		// Get historical data with lookback
		history := historicalData[symbol]
		if len(history) < cfg.LookbackSize {
			return fmt.Errorf("insufficient historical data for symbol %s", symbol)
		}

		// Calculate each indicator
		for _, indicator := range cfg.Indicators {
			// Get calculation window
			window := append(history[len(history)-cfg.LookbackSize:], core.Candle(*candle))

			// Calculate indicator value
			value := indicator.Calculate(window)

			// Store result
			candle.SetIndicator(indicator.Name(), value)
		}
	}

	return nil
}

// Run executes the backtest with indicator support
func (r *Runner) Run(data *types.TimeseriesTable[core.Candle]) error {
	if r.IndicatorConfig != nil {
		// Pre-calculate indicators for the entire dataset
		err := data.ApplyIndicators(r.IndicatorConfig.Indicators)
		if err != nil {
			return fmt.Errorf("failed to pre-calculate indicators: %v", err)
		}
	}

	// Continue with existing run logic...
	return nil
}
