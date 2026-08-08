// This file defines the extensibility contracts (interfaces and context structs)
// for customizing portfolio accounting. Each interface has a Standard
// implementation and a Func wrapper for partial overrides. Inject custom
// implementations at portfolio construction time via the Option functions in
// periodic.go.

package portfolio

import "time"

// TradeCostContext supplies all inputs needed to compute fees and taxes for a
// single fill. Passed to every [CostCalculator] method.
type TradeCostContext struct {
	Order           Order
	Notional        float64
	RealizedProfit  float64
	HoldingPeriod   time.Duration
	Settings        Settings
}

// EntryCashContext supplies inputs for computing cash deducted when opening a position.
type EntryCashContext struct {
	Order    Order
	Settings Settings
}

// ExitCashContext supplies inputs for computing cash credited when closing
// (or partially closing) a position.
type ExitCashContext struct {
	Order            Order
	OpenPrice        float64
	Leverage         float64
	RealizedPnLSlice float64
	HoldingPeriod    time.Duration
	Settings         Settings
}

// PositionEquityContext supplies inputs for computing a single open position's
// contribution to total portfolio equity.
type PositionEquityContext struct {
	Position *Position
	Settings Settings
}

// PeriodicState tracks the last-run timestamps for each category of scheduled
// cash event, enabling frequency-based gating.
type PeriodicState struct {
	LastSIPTime           time.Time
	LastIdleInterestTime  time.Time
	LastLeverageCostTime  time.Time
	LastManagementFeeTime time.Time
}

// PeriodicContext provides a read-only portfolio snapshot to the
// [PeriodicEventsModel] so it can decide which scheduled cash flows to apply.
type PeriodicContext struct {
	Time             time.Time
	Cash             float64
	Equity           float64
	BorrowedNotional float64
	Settings         Settings
	State            PeriodicState
}

// PeriodicResult is the outcome of one periodic-events pass. CashDelta is added
// to the portfolio's cash balance, and State carries forward the updated timestamps.
type PeriodicResult struct {
	CashDelta float64
	State     PeriodicState
}

// CostCalculator computes brokerage and taxes for a trade.
// Inject via portfolio.WithCostCalculator; default is StandardCostCalculator.
type CostCalculator interface {
	Brokerage(ctx TradeCostContext) float64
	TransactionTax(ctx TradeCostContext) float64
	CapitalGainsTax(ctx TradeCostContext) float64
}

// CashFlowCalculator computes cash movement on entry, exit, and equity marks.
// Inject via portfolio.WithCashFlowCalculator; default uses StandardCostCalculator.
type CashFlowCalculator interface {
	EntryCashOutflow(ctx EntryCashContext) float64
	ExitCashInflow(ctx ExitCashContext) float64
	PositionEquityContribution(ctx PositionEquityContext) float64
}

// PeriodicEventsModel applies scheduled cash flows (SIP, interest, fees).
// Inject via portfolio.WithPeriodicEventsModel; default is StandardPeriodicEventsModel.
type PeriodicEventsModel interface {
	Apply(ctx PeriodicContext) PeriodicResult
}
