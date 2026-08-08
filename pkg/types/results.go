package types

import "time"

// Results holds the computed analytics for a completed backtest run.
// The runner populates this struct after the final bar is processed; it is also
// serialized to results.json when an output directory is configured.
type Results struct {
	StartTime      time.Time // First bar timestamp processed
	EndTime        time.Time // Last bar timestamp processed
	InitialCapital float64   // Starting cash, as configured in portfolio settings
	FinalCapital   float64   // Portfolio.Value() at end of backtest (cash + positions)
	TotalTrades    int       // Number of round-trip trades (entry → exit)
	WinningTrades  int       // Trades with positive realized P&L
	LosingTrades   int       // Trades with negative realized P&L

	// MaxDrawdown is the peak-to-trough decline as a fraction in [0, 1].
	// 0.25 means the portfolio fell 25% from its peak equity before recovering.
	MaxDrawdown float64

	// CAGR is the Compound Annual Growth Rate as a decimal (0.12 = 12% per year).
	CAGR float64

	SharpeRatio  float64 // Annualized Sharpe ratio (excess return / annualized std dev)
	SortinoRatio float64 // Like Sharpe but penalizes only downside volatility

	// Returns is total return over the backtest period as a decimal fraction.
	// 0.5 means the portfolio grew 50% from InitialCapital to FinalCapital.
	Returns float64

	// AvgWinPercent is the average per-trade return of winning trades as a percentage (0–100).
	// For example, 3.5 means winning trades averaged +3.5% return.
	AvgWinPercent float64

	// AvgLossPercent is the average per-trade return of losing trades as a percentage (0–100).
	// Stored as a positive number; 2.1 means losing trades averaged -2.1% return.
	AvgLossPercent float64

	// ProfitFactor is gross profits / gross losses. Values > 1 indicate net profitability.
	ProfitFactor float64

	// AnnualizedStdDev is the annualized standard deviation of periodic returns,
	// used as the denominator in Sharpe/Sortino calculations.
	AnnualizedStdDev float64

	AvgWinHoldingDays  float64 // Mean holding period of winning trades, in calendar days
	AvgLossHoldingDays float64 // Mean holding period of losing trades, in calendar days

	RiskFreeRate       float64 // Annual risk-free rate used for Sharpe/Sortino (decimal, e.g. 0.05 = 5%)
	RiskFreeRateSource string  // "provided" if user-supplied, "assumed" if framework default

	// Cost accumulators (populated from portfolio stats at end of backtest)
	TotalBrokerage       float64
	TotalTransactionTax  float64
	TotalCapitalGainsTax float64
	TotalCosts           float64 // Sum of all above

	// SkippedOrders counts orders rejected by risk or ProcessOrder when the
	// runner is in lenient mode (StrictOrderProcessing=false). Zero in strict mode
	// because the first failure aborts the run instead.
	SkippedOrders int

	// Metrics is an extensibility point for strategy-specific or plugin-specific KPIs.
	// Strategies or post-processors can write arbitrary named metrics here (e.g.,
	// "MaxConsecutiveLosses", "CalmarRatio") and they will be included in JSON export.
	Metrics map[string]float64
}

// HoldingSnapshot captures the state of a single open position at one equity-curve tick.
// JSON tags use abbreviated single-character names ("i", "q", "p", "e") to minimize
// payload size — the equity curve can contain thousands of snapshots with multiple
// holdings each, so compact serialization significantly reduces report file sizes.
type HoldingSnapshot struct {
	Instrument string  `json:"i"` // Instrument/symbol identifier
	Quantity   int     `json:"q"` // Current position quantity (shares/lots)
	Price      float64 `json:"p"` // Last close or current mark-to-market price
	AvgEntry   float64 `json:"e"` // Volume-weighted average entry price
}

// AccountValue is a point-in-time snapshot of the portfolio, recorded once per bar
// to build the equity curve. The runner appends one AccountValue per processed tick.
type AccountValue struct {
	Time          time.Time         // Bar timestamp this snapshot corresponds to
	Value         float64           // Total portfolio equity (cash + position values)
	Cash          float64           // Available cash balance
	OpenPositions int               // Count of currently open positions
	UnrealizedPnL float64           // Sum of unrealized P&L across all open positions
	Holdings      []HoldingSnapshot // Per-position detail for drill-down in reports
}
