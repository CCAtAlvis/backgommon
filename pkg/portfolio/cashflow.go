package portfolio

import "time"

// hoursPerYear is used to annualize periodic rates. 365.25 accounts for leap years.
const hoursPerYear = 24 * 365.25

// StandardCashFlowCalculator is the default [CashFlowCalculator] implementation.
// It computes cash movement on entry (margin + fees), exit (margin release + PnL - fees),
// and equity contribution (margin + unrealized PnL). It delegates fee calculations to
// an embedded [CostCalculator] (defaults to [StandardCostCalculator] if nil).
type StandardCashFlowCalculator struct {
	Costs CostCalculator
}

func (c *StandardCashFlowCalculator) costCalc() CostCalculator {
	if c.Costs != nil {
		return c.Costs
	}
	return StandardCostCalculator{}
}

// EntryCashOutflow returns the total cash deducted on opening a position:
// margin requirement (notional / leverage) plus brokerage and transaction tax.
func (c *StandardCashFlowCalculator) EntryCashOutflow(ctx EntryCashContext) float64 {
	ord := ctx.Order
	s := ctx.Settings
	lev := effectiveLeverage(ord, &s)
	notional := tradeNotional(ord.Quantity, ord.Price)
	margin := marginRequired(ord.Quantity, ord.Price, lev)
	costCtx := TradeCostContext{Order: ord, Notional: notional, Settings: s}
	brokerage := c.costCalc().Brokerage(costCtx)
	tax := c.costCalc().TransactionTax(costCtx)
	return margin + brokerage + tax
}

// ExitCashInflow returns net cash credited on closing (or partially closing) a
// position: margin released + realized PnL - brokerage - transaction tax - capital gains tax.
func (c *StandardCashFlowCalculator) ExitCashInflow(ctx ExitCashContext) float64 {
	notional := tradeNotional(ctx.Order.Quantity, ctx.Order.Price)
	marginReleased := marginRequired(ctx.Order.Quantity, ctx.OpenPrice, ctx.Leverage)
	costCtx := TradeCostContext{
		Order:          ctx.Order,
		Notional:       notional,
		RealizedProfit: ctx.RealizedPnLSlice,
		HoldingPeriod:  ctx.HoldingPeriod,
		Settings:       ctx.Settings,
	}
	brokerage := c.costCalc().Brokerage(costCtx)
	transactionTax := c.costCalc().TransactionTax(costCtx)
	cgTax := c.costCalc().CapitalGainsTax(costCtx)
	return marginReleased + ctx.RealizedPnLSlice - brokerage - transactionTax - cgTax
}

// PositionEquityContribution returns the equity attributable to a single open
// position: deployed margin (notional / leverage) plus unrealized PnL.
func (StandardCashFlowCalculator) PositionEquityContribution(ctx PositionEquityContext) float64 {
	pos := ctx.Position
	if pos == nil {
		return 0
	}
	return marginRequired(pos.Quantity, pos.OpenPrice, pos.Leverage) + pos.UnrealizedPnL
}

// FuncCashFlowCalculator allows selective override of individual cash flow
// methods via function fields. Any nil function delegates to Fallback (or the
// default [StandardCashFlowCalculator] if Fallback is nil). Example:
//
//	calc := &portfolio.FuncCashFlowCalculator{
//	    EntryCashOutflowFn: func(ctx portfolio.EntryCashContext) float64 { ... },
//	}
type FuncCashFlowCalculator struct {
	Fallback *StandardCashFlowCalculator

	EntryCashOutflowFn            func(EntryCashContext) float64
	ExitCashInflowFn              func(ExitCashContext) float64
	PositionEquityContributionFn func(PositionEquityContext) float64
}

func (f *FuncCashFlowCalculator) fallback() CashFlowCalculator {
	if f.Fallback != nil {
		return f.Fallback
	}
	return &StandardCashFlowCalculator{}
}

// EntryCashOutflow delegates to EntryCashOutflowFn if set, otherwise falls back
// to the standard implementation.
func (f *FuncCashFlowCalculator) EntryCashOutflow(ctx EntryCashContext) float64 {
	if f.EntryCashOutflowFn != nil {
		return f.EntryCashOutflowFn(ctx)
	}
	return f.fallback().EntryCashOutflow(ctx)
}

// ExitCashInflow delegates to ExitCashInflowFn if set, otherwise falls back to
// the standard implementation.
func (f *FuncCashFlowCalculator) ExitCashInflow(ctx ExitCashContext) float64 {
	if f.ExitCashInflowFn != nil {
		return f.ExitCashInflowFn(ctx)
	}
	return f.fallback().ExitCashInflow(ctx)
}

// PositionEquityContribution delegates to PositionEquityContributionFn if set,
// otherwise falls back to the standard implementation.
func (f *FuncCashFlowCalculator) PositionEquityContribution(ctx PositionEquityContext) float64 {
	if f.PositionEquityContributionFn != nil {
		return f.PositionEquityContributionFn(ctx)
	}
	return f.fallback().PositionEquityContribution(ctx)
}

// EstimateEntryCash returns the cash that would be deducted on processing ord as
// an Entry: margin (notional/leverage) + brokerage + transaction tax. Strategies
// should use this when sizing multi-order batches so local cash budgets match
// ProcessOrder. Price and leverage on ord should match the fill assumptions used
// by the runner (or be a conservative estimate thereof).
func (p *Portfolio) EstimateEntryCash(ord Order) float64 {
	ord = p.normalizeOrder(ord)
	return p.entryCashDelta(ord)
}

// entryCashDelta delegates to the portfolio's CashFlowCalculator.
func (p *Portfolio) entryCashDelta(ord Order) float64 {
	return p.cashFlowCalc().EntryCashOutflow(EntryCashContext{
		Order:    ord,
		Settings: p.settingsSnapshot(),
	})
}

// exitCashDelta delegates to the portfolio's CashFlowCalculator.
func (p *Portfolio) exitCashDelta(ord Order, openPrice, leverage, realizedSlice float64, holding time.Duration) float64 {
	return p.cashFlowCalc().ExitCashInflow(ExitCashContext{
		Order:            ord,
		OpenPrice:        openPrice,
		Leverage:         leverage,
		RealizedPnLSlice: realizedSlice,
		HoldingPeriod:    holding,
		Settings:         p.settingsSnapshot(),
	})
}

func (p *Portfolio) settingsSnapshot() Settings {
	if p.settings == nil {
		return Settings{}
	}
	return *p.settings
}

func tradeNotional(qty int, price float64) float64 {
	return float64(qty) * price
}

func marginRequired(qty int, price, leverage float64) float64 {
	return tradeNotional(qty, price) / leverage
}

func effectiveLeverage(ord Order, s *Settings) float64 {
	lev := ord.Leverage
	if lev <= 0 && s != nil {
		lev = s.DefaultLeverage
	}
	if lev <= 0 {
		lev = 1
	}
	return lev
}
