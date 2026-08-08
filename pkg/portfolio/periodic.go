package portfolio

import "time"

// Option is a functional option applied during [New] to configure the Portfolio's
// cost, cash flow, or periodic event models.
type Option func(*Portfolio)

// WithCostCalculator replaces the default fee/tax calculator.
func WithCostCalculator(c CostCalculator) Option {
	return func(p *Portfolio) {
		p.costCalculator = c
	}
}

// WithCashFlowCalculator replaces the default cash flow model.
func WithCashFlowCalculator(c CashFlowCalculator) Option {
	return func(p *Portfolio) {
		p.cashFlowCalculator = c
	}
}

// WithPeriodicEventsModel replaces the default SIP/interest/fee scheduler.
func WithPeriodicEventsModel(m PeriodicEventsModel) Option {
	return func(p *Portfolio) {
		p.periodicEvents = m
	}
}

func (p *Portfolio) costCalc() CostCalculator {
	if p.costCalculator != nil {
		return p.costCalculator
	}
	return StandardCostCalculator{}
}

func (p *Portfolio) cashFlowCalc() CashFlowCalculator {
	if p.cashFlowCalculator != nil {
		return p.cashFlowCalculator
	}
	return &StandardCashFlowCalculator{Costs: p.costCalc()}
}

func (p *Portfolio) periodicModel() PeriodicEventsModel {
	if p.periodicEvents != nil {
		return p.periodicEvents
	}
	return StandardPeriodicEventsModel{}
}

// ProcessPeriodicEvents applies scheduled cash flows (SIP, idle interest,
// leverage cost, management fee) for the given bar time. The runner calls this
// once per bar. No-op if settings is nil.
func (p *Portfolio) ProcessPeriodicEvents(t time.Time) {
	if p.settings == nil {
		return
	}

	state := PeriodicState{
		LastSIPTime:           p.lastSIPTime,
		LastIdleInterestTime:  p.lastIdleInterestTime,
		LastLeverageCostTime:  p.lastLeverageCostTime,
		LastManagementFeeTime: p.lastManagementFeeTime,
	}

	result := p.periodicModel().Apply(PeriodicContext{
		Time:             t,
		Cash:             p.cash,
		Equity:           p.markToMarketEquity(),
		BorrowedNotional: p.totalBorrowedNotional(),
		Settings:         p.settingsSnapshot(),
		State:            state,
	})

	p.cash += result.CashDelta
	p.lastSIPTime = result.State.LastSIPTime
	p.lastIdleInterestTime = result.State.LastIdleInterestTime
	p.lastLeverageCostTime = result.State.LastLeverageCostTime
	p.lastManagementFeeTime = result.State.LastManagementFeeTime
}

// markToMarketEquity returns cash + open position contributions without periodic side effects.
func (p *Portfolio) markToMarketEquity() float64 {
	value := p.cash
	calc := p.cashFlowCalc()
	settings := p.settingsSnapshot()
	for _, pos := range p.openPositions {
		value += calc.PositionEquityContribution(PositionEquityContext{
			Position: pos,
			Settings: settings,
		})
	}
	return value
}

// totalBorrowedNotional sums borrowed capital across open long positions.
func (p *Portfolio) totalBorrowedNotional() float64 {
	var borrowed float64
	for _, pos := range p.openPositions {
		if pos.Side != Long || pos.Leverage <= 1 {
			continue
		}
		notional := tradeNotional(pos.Quantity, pos.OpenPrice)
		borrowed += notional * (1 - 1/pos.Leverage)
	}
	return borrowed
}
