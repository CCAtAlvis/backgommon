// Package output generates post-backtest reports. It writes a timestamped run
// folder containing results.json(.zst) as the source of truth, an optional
// shared viewer.html / thin report.html, and config.json.
package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

// ReportData holds all data needed to generate a full report.
type ReportData struct {
	Results         *types.Results
	EquityCurve     []types.AccountValue
	ClosedPositions []*portfolio.Position
	OpenPositions   map[string]*portfolio.Position
	Settings        interface{} // Any JSON-serializable settings struct
	Options         ExportOptions
}

// ExportRun creates a timestamped run folder under baseDir and writes report
// artifacts. When Options is zero-valued, single-run defaults are applied:
// zstd + offline data.js + local viewer.html. Returns the run folder path.
func ExportRun(baseDir string, data ReportData) (string, error) {
	ts := time.Now().Format("2006-01-02T15-04-05.000")
	runDir := filepath.Join(baseDir, "run_"+ts)
	if data.Options == (ExportOptions{}) {
		data.Options = ExportOptions{
			Compress:      CompressZstd,
			OfflineDataJS: true,
			WriteViewer:   true,
		}
	} else {
		data.Options = data.Options.withDefaults()
	}
	if err := ExportRunTo(runDir, data); err != nil {
		return runDir, err
	}
	return runDir, nil
}

// ExportRunTo writes report artifacts directly into dir (no timestamp subdirectory).
// Suitable for parallel sweeps where each trial already has a unique folder.
// Creates dir if it does not exist.
func ExportRunTo(dir string, data ReportData) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create run dir: %w", err)
	}

	opts := data.Options.withDefaults()
	_, drawdownPeriods := CalculateDrawdownPeriods(data.EquityCurve)
	report := buildFullReport(data, drawdownPeriods)

	compact, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal results: %w", err)
	}

	switch opts.Compress {
	case CompressNone:
		if err := os.WriteFile(filepath.Join(dir, "results.json"), compact, 0644); err != nil {
			return fmt.Errorf("export results.json: %w", err)
		}
	default: // CompressZstd
		if err := WriteZSTD(filepath.Join(dir, "results.json.zst"), compact); err != nil {
			return fmt.Errorf("export results.json.zst: %w", err)
		}
	}

	if data.Settings != nil {
		if err := exportConfigJSON(data.Settings, filepath.Join(dir, "config.json")); err != nil {
			return fmt.Errorf("export config.json: %w", err)
		}
	}

	if opts.OfflineDataJS {
		if err := writeDataJS(filepath.Join(dir, "data.js"), compact, data.Settings); err != nil {
			return fmt.Errorf("export data.js: %w", err)
		}
		if err := WriteThinReportHTML(filepath.Join(dir, "report.html")); err != nil {
			return fmt.Errorf("export report.html: %w", err)
		}
	}

	if opts.WriteViewer {
		if err := WriteViewerHTML(filepath.Join(dir, "viewer.html")); err != nil {
			return fmt.Errorf("export viewer.html: %w", err)
		}
	}

	return nil
}

func writeDataJS(path string, reportJSON []byte, settings interface{}) error {
	settingsJSON := []byte("{}")
	if settings != nil {
		var err error
		settingsJSON, err = json.Marshal(settings)
		if err != nil {
			return err
		}
	}
	// Escape </ to avoid breaking out of a surrounding <script> if inlined later.
	safeReport := bytesReplaceLT(reportJSON)
	safeSettings := bytesReplaceLT(settingsJSON)
	content := append([]byte("window.__REPORT__="), safeReport...)
	content = append(content, ';', '\n')
	content = append(content, []byte("window.__SETTINGS__=")...)
	content = append(content, safeSettings...)
	content = append(content, ';', '\n')
	return os.WriteFile(path, content, 0644)
}

func bytesReplaceLT(b []byte) []byte {
	// Replace '<' with unicode escape so "</script>" cannot appear in JSON text.
	out := make([]byte, 0, len(b))
	for _, c := range b {
		if c == '<' {
			out = append(out, '\\', 'u', '0', '0', '3', 'c')
		} else {
			out = append(out, c)
		}
	}
	return out
}

func exportConfigJSON(settings interface{}, path string) error {
	out, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}

// FullReport is the top-level JSON structure written to results.json. It
// contains everything needed to reconstruct the backtest outcome: metadata,
// aggregate metrics, drawdown periods, trade analysis, the full equity curve,
// individual trade records, order records, and open positions.
type FullReport struct {
	Meta             ReportMeta        `json:"meta"`
	Metrics          FullMetrics       `json:"metrics"`
	DrawdownPeriods  []DrawdownPeriod  `json:"drawdown_periods"`
	TradeAnalysis    TradeAnalysis     `json:"trade_analysis"`
	EquityCurve      []EquityCurvePoint `json:"equity_curve"`
	AllTrades        []TradeRecord     `json:"all_trades"`
	AllOrders        []OrderRecord     `json:"all_orders"`
	OpenPositions    []HoldingRecord   `json:"open_positions"`
}

// FullMetrics holds aggregate performance metrics for reporting: returns, risk
// ratios, trade counts, and win/loss averages. All percentage fields are stored
// as decimals (e.g. 0.12 for 12%).
type FullMetrics struct {
	Returns            float64 `json:"returns"`
	CAGR               float64 `json:"cagr"`
	MaxDrawdown        float64 `json:"max_drawdown"`
	SharpeRatio        float64 `json:"sharpe_ratio"`
	SortinoRatio       float64 `json:"sortino_ratio"`
	ProfitFactor       float64 `json:"profit_factor"`
	AnnualizedStdDev   float64 `json:"annualized_std_dev"`
	TotalTrades        int     `json:"total_trades"`
	WinningTrades      int     `json:"winning_trades"`
	LosingTrades       int     `json:"losing_trades"`
	WinRate            float64 `json:"win_rate"`
	AvgWinPercent      float64 `json:"avg_win_percent"`
	AvgLossPercent     float64 `json:"avg_loss_percent"`
	AvgWinHoldingDays  float64 `json:"avg_win_holding_days"`
	AvgLossHoldingDays float64 `json:"avg_loss_holding_days"`
	RiskFreeRate         float64 `json:"risk_free_rate"`
	RiskFreeRateSource   string  `json:"risk_free_rate_source"`
	TotalBrokerage       float64 `json:"total_brokerage,omitempty"`
	TotalTransactionTax  float64 `json:"total_transaction_tax,omitempty"`
	TotalCapitalGainsTax float64 `json:"total_capital_gains_tax,omitempty"`
	TotalCosts           float64 `json:"total_costs,omitempty"`
}

// TradeAnalysis summarises trade-level statistics, splitting closed positions
// into realized profits vs losses and aggregating unrealized PnL from open
// positions.
type TradeAnalysis struct {
	RealizedProfits  TradeSummary `json:"realized_profits"`
	RealizedLosses   TradeSummary `json:"realized_losses"`
	UnrealizedProfit float64      `json:"unrealized_profit"`
	UnrealizedLoss   float64      `json:"unrealized_loss"`
	ProfitPositions  int          `json:"profit_positions"`
	LossPositions    int          `json:"loss_positions"`
}

// TradeSummary contains per-side (winning or losing) trade statistics: count,
// total PnL, average PnL percentage, and average holding period in days.
type TradeSummary struct {
	Count             int     `json:"count"`
	TotalPnL          float64 `json:"total_pnl"`
	AvgPnLPercent     float64 `json:"avg_pnl_percent"`
	AvgHoldingDays    float64 `json:"avg_holding_days"`
}

// TradeRecord is a detailed record of a single completed trade, including
// entry/exit prices, PnL, and holding period. Order timelines are not nested
// here — join all_orders by trade_id in the viewer (and recompute running state).
type TradeRecord struct {
	TradeID     string             `json:"trade_id"`
	Instrument  string             `json:"instrument"`
	Quantity    int                `json:"quantity"` // Peak qty held
	OpenDate    string             `json:"open_date"`
	CloseDate   string             `json:"close_date"`
	AvgEntry    float64            `json:"avg_entry"`
	ExitPrice   float64            `json:"exit_price"`
	PnL         float64            `json:"pnl"`
	PnLPerShare float64            `json:"pnl_per_share"`
	PnLPercent  float64            `json:"pnl_percent"`
	HoldingDays float64            `json:"holding_days"`
	Orders      []TradeOrderRecord `json:"orders,omitempty"` // legacy; not written by buildTradeRecords
}

// TradeOrderRecord is a single order within a trade, with running state for the "story" view.
type TradeOrderRecord struct {
	Date          string  `json:"date"`
	Action        string  `json:"action"` // BUY or SELL
	Qty           int     `json:"qty"`
	Price         float64 `json:"price"`
	RunningQty    int     `json:"running_qty"`
	AvgPrice      float64 `json:"avg_price"`      // Running weighted avg buy price
	RunningPnL    float64 `json:"running_pnl"`    // Cumulative realized PnL after this order
	UnrealizedPnL float64 `json:"unrealized_pnl"` // Mark-to-market unrealized at this order's price
}

// OrderRecord represents a single filled order for the flat orders table in
// reports. It links back to a TradeID so viewers can cross-reference orders
// with their parent trade.
type OrderRecord struct {
	TradeID    string  `json:"trade_id"`
	Instrument string  `json:"instrument"`
	Side       string  `json:"side"`
	Type       string  `json:"type"`
	Quantity   int     `json:"quantity"`
	Price      float64 `json:"price"`
	Date       string  `json:"date"`
}

// HoldingRecord is a snapshot of an open position at report time, showing
// current mark-to-market value and unrealized PnL.
type HoldingRecord struct {
	Instrument    string  `json:"instrument"`
	Quantity      int     `json:"quantity"`
	OpenDate      string  `json:"open_date"`
	OpenPrice     float64 `json:"open_price"`
	CurrentPrice  float64 `json:"current_price"`
	Value         float64 `json:"value"`
	PnL           float64 `json:"pnl"`
	PnLPercent    float64 `json:"pnl_percent"`
}

func buildFullReport(data ReportData, periods []DrawdownPeriod) FullReport {
	r := data.Results
	winRate := 0.0
	if r.TotalTrades > 0 {
		winRate = float64(r.WinningTrades) / float64(r.TotalTrades)
	}

	report := FullReport{
		Meta: ReportMeta{
			InitialCapital: r.InitialCapital,
			FinalCapital:   r.FinalCapital,
		},
		Metrics: FullMetrics{
			Returns:            r.Returns,
			CAGR:               r.CAGR,
			MaxDrawdown:        r.MaxDrawdown,
			SharpeRatio:        r.SharpeRatio,
			SortinoRatio:       r.SortinoRatio,
			ProfitFactor:       r.ProfitFactor,
			AnnualizedStdDev:   r.AnnualizedStdDev,
			TotalTrades:        r.TotalTrades,
			WinningTrades:      r.WinningTrades,
			LosingTrades:       r.LosingTrades,
			WinRate:            winRate,
			AvgWinPercent:      r.AvgWinPercent,
			AvgLossPercent:     r.AvgLossPercent,
			AvgWinHoldingDays:  r.AvgWinHoldingDays,
			AvgLossHoldingDays: r.AvgLossHoldingDays,
			RiskFreeRate:         r.RiskFreeRate,
			RiskFreeRateSource:   r.RiskFreeRateSource,
			TotalBrokerage:       r.TotalBrokerage,
			TotalTransactionTax:  r.TotalTransactionTax,
			TotalCapitalGainsTax: r.TotalCapitalGainsTax,
			TotalCosts:           r.TotalCosts,
		},
		DrawdownPeriods: periods,
	}

	if !r.StartTime.IsZero() {
		report.Meta.StartTime = r.StartTime.Format("2006-01-02")
	}
	if !r.EndTime.IsZero() {
		report.Meta.EndTime = r.EndTime.Format("2006-01-02")
	}

	// Build trade analysis
	report.TradeAnalysis = buildTradeAnalysis(data.ClosedPositions, data.OpenPositions)

	// Build equity curve points
	report.EquityCurve = make([]EquityCurvePoint, len(data.EquityCurve))
	for i, pt := range data.EquityCurve {
		report.EquityCurve[i] = EquityCurvePoint{
			Time:          pt.Time.Format("2006-01-02"),
			Value:         pt.Value,
			Cash:          pt.Cash,
			OpenPositions: pt.OpenPositions,
			UnrealizedPnL: pt.UnrealizedPnL,
			Holdings:      pt.Holdings,
		}
	}

	// Build all trades (orders live in all_orders only)
	report.AllTrades = buildTradeRecords(data.ClosedPositions, false)

	// Build all orders (from both closed and open positions)
	report.AllOrders = buildOrderRecords(data.ClosedPositions, data.OpenPositions)

	// Build open positions
	report.OpenPositions = buildHoldingRecords(data.OpenPositions)

	return report
}

func buildTradeAnalysis(closed []*portfolio.Position, open map[string]*portfolio.Position) TradeAnalysis {
	var ta TradeAnalysis
	var profitPnLSum, lossPnLSum float64
	var profitPctSum, lossPctSum float64
	var profitDaysSum, lossDaysSum float64

	for _, pos := range closed {
		holdDays := pos.CloseTime.Sub(pos.OpenTime).Hours() / 24
		if pos.RealizedPnL > 0 {
			ta.RealizedProfits.Count++
			profitPnLSum += pos.RealizedPnL
			if pos.OpenPrice > 0 {
				profitPctSum += (pos.ClosePrice - pos.OpenPrice) / pos.OpenPrice * 100
			}
			profitDaysSum += holdDays
		} else if pos.RealizedPnL < 0 {
			ta.RealizedLosses.Count++
			lossPnLSum += pos.RealizedPnL
			if pos.OpenPrice > 0 {
				lossPctSum += (pos.ClosePrice - pos.OpenPrice) / pos.OpenPrice * 100
			}
			lossDaysSum += holdDays
		}
	}

	ta.RealizedProfits.TotalPnL = profitPnLSum
	ta.RealizedLosses.TotalPnL = lossPnLSum
	if ta.RealizedProfits.Count > 0 {
		ta.RealizedProfits.AvgPnLPercent = profitPctSum / float64(ta.RealizedProfits.Count)
		ta.RealizedProfits.AvgHoldingDays = profitDaysSum / float64(ta.RealizedProfits.Count)
	}
	if ta.RealizedLosses.Count > 0 {
		ta.RealizedLosses.AvgPnLPercent = lossPctSum / float64(ta.RealizedLosses.Count)
		ta.RealizedLosses.AvgHoldingDays = lossDaysSum / float64(ta.RealizedLosses.Count)
	}

	for _, pos := range open {
		if pos.UnrealizedPnL > 0 {
			ta.ProfitPositions++
			ta.UnrealizedProfit += pos.UnrealizedPnL
		} else {
			ta.LossPositions++
			ta.UnrealizedLoss += pos.UnrealizedPnL
		}
	}

	return ta
}

func buildTradeRecords(closed []*portfolio.Position, includeOrders bool) []TradeRecord {
	records := make([]TradeRecord, 0, len(closed))
	for _, pos := range closed {
		qty := pos.PeakQty
		if qty == 0 {
			qty = pos.Quantity
		}
		pnlPerShare := 0.0
		if qty > 0 {
			pnlPerShare = pos.RealizedPnL / float64(qty)
		}

		orders, peakInvestment := buildTradeOrderTimeline(pos)

		// PnL% = realized_pnl / peak_invested_capital * 100
		// This measures actual return on max capital deployed
		pnlPct := 0.0
		if peakInvestment > 0 {
			pnlPct = pos.RealizedPnL / peakInvestment * 100
		}

		rec := TradeRecord{
			TradeID:     pos.ID,
			Instrument:  pos.Instrument,
			Quantity:    qty,
			OpenDate:    pos.OpenTime.Format("2006-01-02"),
			CloseDate:   pos.CloseTime.Format("2006-01-02"),
			AvgEntry:    pos.OpenPrice,
			ExitPrice:   pos.ClosePrice,
			PnL:         pos.RealizedPnL,
			PnLPerShare: pnlPerShare,
			PnLPercent:  pnlPct,
			HoldingDays: pos.CloseTime.Sub(pos.OpenTime).Hours() / 24,
		}
		if includeOrders {
			rec.Orders = orders
		}
		records = append(records, rec)
	}
	return records
}

// buildTradeOrderTimeline constructs the order-by-order "story" of a trade with running state.
// Returns the timeline and peak investment (max capital deployed).
func buildTradeOrderTimeline(pos *portfolio.Position) ([]TradeOrderRecord, float64) {
	timeline := make([]TradeOrderRecord, 0, len(pos.Orders))
	runningQty := 0
	avgPrice := 0.0
	runningPnL := 0.0
	peakInvestment := 0.0

	for _, ord := range pos.Orders {
		action := "BUY"
		if ord.Type == portfolio.Exit {
			action = "SELL"
		}

		if ord.Type == portfolio.Entry {
			totalCost := avgPrice*float64(runningQty) + ord.Price*float64(ord.Quantity)
			runningQty += ord.Quantity
			if runningQty > 0 {
				avgPrice = totalCost / float64(runningQty)
			}
		} else {
			pnl := float64(ord.Quantity) * (ord.Price - avgPrice)
			runningPnL += pnl
			runningQty -= ord.Quantity
		}

		// Track peak investment (max capital at risk)
		invested := float64(runningQty) * avgPrice
		if invested > peakInvestment {
			peakInvestment = invested
		}

		// Unrealized PnL: mark position to this order's price
		unrealizedPnL := float64(runningQty) * (ord.Price - avgPrice)

		timeline = append(timeline, TradeOrderRecord{
			Date:          ord.FilledAt.Format("2006-01-02"),
			Action:        action,
			Qty:           ord.Quantity,
			Price:         ord.Price,
			RunningQty:    runningQty,
			AvgPrice:      avgPrice,
			RunningPnL:    runningPnL,
			UnrealizedPnL: unrealizedPnL,
		})
	}
	return timeline, peakInvestment
}

func buildOrderRecords(closed []*portfolio.Position, open map[string]*portfolio.Position) []OrderRecord {
	var records []OrderRecord

	addOrders := func(pos *portfolio.Position) {
		for _, ord := range pos.Orders {
			typeStr := "ENTRY"
			if ord.Type == portfolio.Exit {
				typeStr = "EXIT"
			}
			sideStr := "LONG"
			if ord.Side == portfolio.Short {
				sideStr = "SHORT"
			}
			records = append(records, OrderRecord{
				TradeID:    pos.ID,
				Instrument: ord.Instrument,
				Side:       sideStr,
				Type:       typeStr,
				Quantity:   ord.Quantity,
				Price:      ord.Price,
				Date:       ord.FilledAt.Format("2006-01-02"),
			})
		}
	}

	for _, pos := range closed {
		addOrders(pos)
	}
	for _, pos := range open {
		addOrders(pos)
	}
	return records
}

func buildHoldingRecords(open map[string]*portfolio.Position) []HoldingRecord {
	records := make([]HoldingRecord, 0, len(open))
	for _, pos := range open {
		lastPrice := pos.HighestPrice
		if pos.LowestPrice > 0 && pos.LowestPrice < lastPrice {
			// Use the price that was last set (from UnrealizedPnL calculation)
		}
		// Current value is based on unrealized PnL
		currentPrice := pos.OpenPrice
		if pos.UnrealizedPnL != 0 && pos.Quantity > 0 {
			currentPrice = pos.OpenPrice + (pos.UnrealizedPnL / float64(pos.Quantity))
		}
		value := currentPrice * float64(pos.Quantity)
		pnlPct := 0.0
		if pos.OpenPrice > 0 {
			pnlPct = (currentPrice - pos.OpenPrice) / pos.OpenPrice * 100
		}
		records = append(records, HoldingRecord{
			Instrument:   pos.Instrument,
			Quantity:     pos.Quantity,
			OpenDate:     pos.OpenTime.Format("2006-01-02"),
			OpenPrice:    pos.OpenPrice,
			CurrentPrice: currentPrice,
			Value:        value,
			PnL:          pos.UnrealizedPnL,
			PnLPercent:   pnlPct,
		})
	}
	return records
}
