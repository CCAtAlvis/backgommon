package portfolio

// StandardCostCalculator is the default [CostCalculator] implementation. It
// derives all fees and taxes from [Settings] fields (fixed + percent brokerage,
// buy/sell transaction tax, and short/long-term capital gains tax). Use it
// directly or as the fallback inside [FuncCostCalculator] for partial overrides.
type StandardCostCalculator struct{}

// Brokerage returns the sum of FixedBrokerageFee and PercentBrokerageRate * notional.
func (StandardCostCalculator) Brokerage(ctx TradeCostContext) float64 {
	s := ctx.Settings
	fee := s.FixedBrokerageFee
	if s.PercentBrokerageRate > 0 {
		fee += ctx.Notional * s.PercentBrokerageRate
	}
	return fee
}

// TransactionTax returns the applicable buy or sell tax on notional value.
// Returns 0 when Settings.EnableTaxes is false. For entry orders the tax rate
// depends on side (BuyTaxRate for Long, SellTaxRate for Short); exits always
// use SellTaxRate.
func (StandardCostCalculator) TransactionTax(ctx TradeCostContext) float64 {
	s := ctx.Settings
	if !s.EnableTaxes {
		return 0
	}
	switch ctx.Order.Type {
	case Entry:
		if ctx.Order.Side == Long {
			return ctx.Notional * s.BuyTaxRate
		}
		return ctx.Notional * s.SellTaxRate
	case Exit:
		return ctx.Notional * s.SellTaxRate
	default:
		return 0
	}
}

// CapitalGainsTax applies the appropriate short-term or long-term capital gains
// rate to realized profit. Returns 0 when taxes are disabled or profit is <= 0.
// The holding period determines which rate applies.
func (StandardCostCalculator) CapitalGainsTax(ctx TradeCostContext) float64 {
	s := ctx.Settings
	if !s.EnableTaxes || ctx.RealizedProfit <= 0 {
		return 0
	}
	rate := s.LTCapitalGainsTaxRate
	if s.ShortTermHoldingPeriod > 0 && ctx.HoldingPeriod < s.ShortTermHoldingPeriod {
		rate = s.STCapitalGainsTaxRate
	}
	return ctx.RealizedProfit * rate
}

// FuncCostCalculator allows selective override of individual cost methods via
// function fields. Any nil function delegates to Fallback (or the default
// [StandardCostCalculator] if Fallback is nil). This enables partial
// customization without reimplementing the entire CostCalculator interface:
//
//	calc := &portfolio.FuncCostCalculator{
//	    BrokerageFn: func(ctx portfolio.TradeCostContext) float64 { return 10.0 },
//	}
type FuncCostCalculator struct {
	Fallback *StandardCostCalculator

	BrokerageFn       func(TradeCostContext) float64
	TransactionTaxFn  func(TradeCostContext) float64
	CapitalGainsTaxFn func(TradeCostContext) float64
}

func (f *FuncCostCalculator) fallback() CostCalculator {
	if f.Fallback != nil {
		return f.Fallback
	}
	return StandardCostCalculator{}
}

// Brokerage delegates to BrokerageFn if set, otherwise falls back to the
// standard implementation.
func (f *FuncCostCalculator) Brokerage(ctx TradeCostContext) float64 {
	if f.BrokerageFn != nil {
		return f.BrokerageFn(ctx)
	}
	return f.fallback().Brokerage(ctx)
}

// TransactionTax delegates to TransactionTaxFn if set, otherwise falls back to
// the standard implementation.
func (f *FuncCostCalculator) TransactionTax(ctx TradeCostContext) float64 {
	if f.TransactionTaxFn != nil {
		return f.TransactionTaxFn(ctx)
	}
	return f.fallback().TransactionTax(ctx)
}

// CapitalGainsTax delegates to CapitalGainsTaxFn if set, otherwise falls back
// to the standard implementation.
func (f *FuncCostCalculator) CapitalGainsTax(ctx TradeCostContext) float64 {
	if f.CapitalGainsTaxFn != nil {
		return f.CapitalGainsTaxFn(ctx)
	}
	return f.fallback().CapitalGainsTax(ctx)
}

