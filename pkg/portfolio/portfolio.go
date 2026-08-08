// Package portfolio manages position tracking, cash accounting, and trade cost
// simulation for backtesting strategies.
//
// The package is built around three extensible contracts:
//   - [CostCalculator] — brokerage fees, transaction taxes, and capital gains taxes
//   - [CashFlowCalculator] — cash movement on entry, exit, and equity marks
//   - [PeriodicEventsModel] — scheduled cash flows (SIP, interest, leverage costs, management fees)
//
// Each contract has a Standard implementation driven by [Settings] fields and a
// Func wrapper (e.g. [FuncCostCalculator]) for partial overrides. Inject custom
// implementations at construction time using the Option pattern:
//
//	p := portfolio.New(settings,
//	    portfolio.WithCostCalculator(myCalc),
//	    portfolio.WithPeriodicEventsModel(myModel),
//	)
package portfolio

import (
	"fmt"
	"time"
)

// Portfolio is the central accounting engine for a backtest. It tracks open and
// closed positions, maintains the cash balance, accumulates trade costs, and
// computes mark-to-market equity.
//
// Portfolio is NOT safe for concurrent use. The runner drives it sequentially,
// one bar at a time.
type Portfolio struct {
	cash            float64
	openPositions   map[string]*Position
	closedPositions []*Position
	orderHistory    []Order
	settings        *Settings

	costCalculator     CostCalculator
	cashFlowCalculator CashFlowCalculator
	periodicEvents     PeriodicEventsModel

	lastSIPTime           time.Time
	lastIdleInterestTime  time.Time
	lastLeverageCostTime  time.Time
	lastManagementFeeTime time.Time

	// Accumulated costs for reporting
	totalBrokerage         float64
	brokerageBuySide       float64
	brokerageSellSide      float64
	totalTransactionTax    float64
	transactionTaxBuySide  float64
	transactionTaxSellSide float64
	totalCapitalGainsTax   float64
	totalInvestment        float64
}

// Settings holds all portfolio configuration: capital, leverage, periodic
// contributions, brokerage fees, taxation rules, and profit management.
// Pass a *Settings to [New] to initialize the portfolio's accounting rules.
type Settings struct {
	// --- Core Portfolio Setup ---

	// InitialCapital is the starting cash balance of the portfolio.
	// e.g., 50000.0
	InitialCapital float64

	// EnableShorts determines if short selling is permitted.
	EnableShorts bool
	// DefaultLeverage is the leverage to apply to an order if not specified in the order itself.
	// This is still subject to MaxLeverage defined in risk.Settings.
	// A value of 1.0 means no leverage. Values > 1.0 imply borrowing.
	// e.g., 1.0 (no leverage), 2.0 (2x leverage)
	DefaultLeverage float64

	// CashReserveRate is the percentage of the total portfolio value (typically based on InitialCapital
	// or current equity) that should be kept as uninvested cash, effectively reducing the capital
	// available for trading. Expressed as a decimal.
	// e.g., 0.02 (for a 2% cash reserve)
	CashReserveRate float64

	// AllowNegativeCashFromFees when true allows fees (brokerage + taxes) to push
	// the cash balance below zero, but the order's margin (notional value) itself
	// must still fit within available cash. Once cash is negative, no new entries
	// are permitted. Default is false (full amount including fees must fit in cash).
	AllowNegativeCashFromFees bool

	// --- Periodic Contributions/Withdrawals ---

	// SIPAmount is the amount for Systematic Investment Plan contributions.
	// If > 0, SIPFrequency must also be set to a non-zero duration.
	// e.g., 1000.0
	SIPAmount float64
	// SIPFrequency is the interval at which SIPAmount is added to the portfolio's cash.
	// e.g., 30 * 24 * time.Hour (for monthly contributions)
	SIPFrequency time.Duration

	// --- Interest & Costs on Capital ---

	// IdleCashInterestAnnualRate is the annual interest rate earned on uninvested cash.
	// Accrued based on IdleCashInterestFrequency. Expressed as a decimal.
	// e.g., 0.03 (for 3% annual rate)
	IdleCashInterestAnnualRate float64
	// IdleCashInterestFrequency is how often the idle cash interest is calculated and added to cash.
	// The IdleCashInterestAnnualRate will be proportionally adjusted to this period.
	// e.g., 24 * time.Hour (for daily accrual)
	IdleCashInterestFrequency time.Duration

	// LeverageCostAnnualRate is the annual interest rate charged on borrowed capital.
	// Applied per LeverageCostFrequency. Expressed as a decimal.
	// e.g., 0.05 (for 5% annual rate on borrowed funds)
	LeverageCostAnnualRate float64
	// LeverageCostFrequency is the time interval for which LeverageCostAnnualRate is applied (proportionally).
	// e.g., 24 * time.Hour (for daily cost calculation)
	LeverageCostFrequency time.Duration

	// --- Brokerage / Commission Model ---

	// FixedBrokerageFee is a flat fee applied to each trade (both entry and exit).
	// If 0, this component of brokerage is ignored.
	// e.g., 5.0 (for $5 per trade)
	FixedBrokerageFee float64

	// PercentBrokerageRate is a brokerage fee calculated as a percentage of the total trade value.
	// Applied to both entry and exit. Expressed as a decimal.
	// If 0, this component of brokerage is ignored.
	// e.g., 0.001 (for 0.1% of trade value)
	PercentBrokerageRate float64

	// --- Taxation ---

	// EnableTaxes flags whether trading taxes are applied.
	// e.g., false
	EnableTaxes bool
	// BuyTaxRate is the tax rate applied to the total value of a buy order.
	// Expressed as a decimal. e.g., 0.0020 (for 0.20%)
	BuyTaxRate float64
	// SellTaxRate is the tax rate applied to the total value of a sell order.
	// Expressed as a decimal. e.g., 0.0020 (for 0.20%)
	SellTaxRate float64
	// STCapitalGainsTaxRate is the tax rate for short-term capital gains (profits from trades held less than ShortTermHoldingPeriod).
	// Expressed as a decimal. e.g., 0.15 (for 15%)
	STCapitalGainsTaxRate float64
	// LTCapitalGainsTaxRate is the tax rate for long-term capital gains (profits from trades held ShortTermHoldingPeriod or longer).
	// Expressed as a decimal. e.g., 0.10 (for 10%)
	LTCapitalGainsTaxRate float64
	// ShortTermHoldingPeriod defines the duration after which a trade is considered long-term for tax purposes.
	// e.g., 365 * 24 * time.Hour (for one year)
	ShortTermHoldingPeriod time.Duration

	// --- Profit Management ---

	// EnableProfitPocketing flags whether the profit "pocketing" mechanism is active.
	// e.g., false
	EnableProfitPocketing bool
	// MinProfitForPocketing is the minimum realized profit a position must achieve to trigger pocketing.
	// e.g., 1000.0 (currency units)
	MinProfitForPocketing float64
	// ProfitPocketingRate is the percentage of realized profit to be "pocketed".
	// Expressed as a decimal. e.g., 0.1 (for 10%)
	ProfitPocketingRate float64

	// --- Management Fees ---

	// EnableManagementFee flags whether management fees are deducted.
	// e.g., false
	EnableManagementFee bool
	// ManagementFeeAnnualRate is the annual management fee rate, typically based on total portfolio equity.
	// Expressed as a decimal. e.g., 0.01 (for 1%)
	ManagementFeeAnnualRate float64
	// ManagementFeeFrequency is how often the management fee is calculated and deducted.
	// e.g., 30 * 24 * time.Hour (for monthly deduction)
	ManagementFeeFrequency time.Duration

	// --- Embedded Execution Settings ---
	Execution ExecutionSettings
}

// New creates a Portfolio initialized with the given settings and optional
// functional options. If no options are provided, the portfolio uses
// [StandardCostCalculator], [StandardCashFlowCalculator], and
// [StandardPeriodicEventsModel] driven by the Settings fields.
func New(settings *Settings, opts ...Option) *Portfolio {
	p := &Portfolio{
		cash:            settings.InitialCapital,
		openPositions:   make(map[string]*Position),
		closedPositions: make([]*Position, 0),
		orderHistory:    make([]Order, 0),
		settings:        settings,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Settings returns a read-only view of portfolio configuration.
func (p *Portfolio) Settings() Settings {
	return p.settingsSnapshot()
}

// ProcessOrder validates and executes an order against the portfolio.
//
// For Entry orders: creates a new position or adds to an existing one for the
// same instrument, deducts margin + fees from cash.
//
// For Exit orders: reduces position quantity, credits margin release + realized
// PnL - fees to cash. A full exit (quantity == position size) closes the position
// and moves it to closed history.
//
// Returns an error if: shorts are disabled, cash is insufficient for entry,
// no matching position exists for exit, or exit quantity exceeds position size.
func (p *Portfolio) ProcessOrder(ord Order) error {
	ord = p.normalizeOrder(ord)

	if err := p.validateOrder(ord); err != nil {
		return err
	}

	switch ord.Type {
	case Entry:
		return p.handleEntryOrder(ord)
	case Exit:
		return p.handleExitOrder(ord)
	default:
		return fmt.Errorf("invalid order type")
	}
}

// normalizeOrder resolves conventions before validation (Quantity 0 exit = full close).
func (p *Portfolio) normalizeOrder(ord Order) Order {
	if ord.Type == Exit && ord.Quantity == 0 {
		if pos, ok := p.openPositions[ord.Instrument]; ok {
			ord.Quantity = pos.Quantity
		}
	}
	ord.Leverage = effectiveLeverage(ord, p.settings)
	return ord
}

// UpdatePositions refreshes mark-to-market metrics (unrealized PnL, max drawdown,
// highest/lowest price) for all open positions using the given instrument→price map.
// Instruments not present in prices are left unchanged.
func (p *Portfolio) UpdatePositions(prices map[string]float64) {
	for instrument, pos := range p.openPositions {
		if price, ok := prices[instrument]; ok {
			p.updatePosition(pos, price)
		}
	}
}

// Value returns total portfolio equity: cash balance plus the mark-to-market
// contribution of every open position (margin deployed + unrealized PnL).
func (p *Portfolio) Value() float64 {
	return p.markToMarketEquity()
}

// InitialCapital returns the configured starting cash.
func (p *Portfolio) InitialCapital() float64 {
	return p.settings.InitialCapital
}

// Stats returns aggregate portfolio statistics. This is the preferred accessor;
// it delegates to [Portfolio.GetPortfolioStats].
func (p *Portfolio) Stats() PortfolioStats {
	return p.GetPortfolioStats()
}

// ClosedPositions returns a defensive copy of the closed position slice.
// Callers may safely mutate the returned slice without affecting the portfolio.
func (p *Portfolio) ClosedPositions() []*Position {
	out := make([]*Position, len(p.closedPositions))
	copy(out, p.closedPositions)
	return out
}

// Cash returns the current uninvested cash balance.
func (p *Portfolio) Cash() float64 {
	return p.cash
}

// Positions returns a shallow copy of the open positions map keyed by instrument.
// The returned map is safe to iterate and delete from without affecting the
// portfolio. The Position pointers still refer to the same structs, so field
// reads reflect live state but callers should not mutate Position fields directly.
func (p *Portfolio) Positions() map[string]*Position {
	cp := make(map[string]*Position, len(p.openPositions))
	for k, v := range p.openPositions {
		cp[k] = v
	}
	return cp
}

// Internal methods

func (p *Portfolio) validateOrder(ord Order) error {
	switch ord.Type {
	case Entry:
		if !p.settings.EnableShorts && ord.Side == Short {
			return fmt.Errorf("short positions not allowed")
		}

		// Long and short entries both require cash for margin (+ fees unless
		// AllowNegativeCashFromFees). Previously shorts skipped this check.
		if ord.Side == Long || ord.Side == Short {
			if p.cash <= 0 {
				return fmt.Errorf("insufficient cash: have %.2f, balance is non-positive", p.cash)
			}

			if p.settings.AllowNegativeCashFromFees {
				// Only the margin (order value) must fit in cash; fees may push balance negative
				lev := effectiveLeverage(ord, p.settings)
				margin := marginRequired(ord.Quantity, ord.Price, lev)
				if margin > p.cash {
					return fmt.Errorf("insufficient cash: have %.2f, need %.2f (margin)", p.cash, margin)
				}
			} else {
				requiredCash := p.entryCashDelta(ord)
				if requiredCash > p.cash {
					return fmt.Errorf("insufficient cash: have %.2f, need %.2f (margin + fees)", p.cash, requiredCash)
				}
			}
		}

	case Exit:
		pos, exists := p.openPositions[ord.Instrument]
		if !exists {
			return fmt.Errorf("no open position for %s", ord.Instrument)
		}
		if ord.Quantity > pos.Quantity {
			return fmt.Errorf("exit quantity exceeds position size")
		}
	}

	return nil
}

func (p *Portfolio) handleEntryOrder(ord Order) error {
	pos, exists := p.openPositions[ord.Instrument]
	if !exists {
		newPos, err := NewPosition(ord)
		if err != nil {
			return err
		}
		p.openPositions[ord.Instrument] = newPos
	} else {
		if err := pos.AddOrder(ord); err != nil {
			return err
		}
	}

	p.cash -= p.entryCashDelta(ord)
	p.orderHistory = append(p.orderHistory, ord)

	// Track costs
	ctx := TradeCostContext{
		Order:    ord,
		Notional: float64(ord.Quantity) * ord.Price,
		Settings: *p.settings,
	}
	brokerage := p.costCalc().Brokerage(ctx)
	txnTax := p.costCalc().TransactionTax(ctx)
	p.totalBrokerage += brokerage
	p.brokerageBuySide += brokerage
	p.totalTransactionTax += txnTax
	p.transactionTaxBuySide += txnTax
	p.totalInvestment += float64(ord.Quantity) * ord.Price

	return nil
}

func (p *Portfolio) handleExitOrder(ord Order) error {
	pos, exists := p.openPositions[ord.Instrument]
	if !exists {
		return fmt.Errorf("no open position for %s", ord.Instrument)
	}

	prevRealized := pos.RealizedPnL
	openPrice := pos.OpenPrice
	leverage := pos.Leverage
	openTime := pos.OpenTime

	if err := pos.AddOrder(ord); err != nil {
		return err
	}

	realizedSlice := pos.RealizedPnL - prevRealized
	holding := ord.FilledAt.Sub(openTime)
	if holding < 0 {
		holding = 0
	}

	p.cash += p.exitCashDelta(ord, openPrice, leverage, realizedSlice, holding)

	// Track costs
	ctx := TradeCostContext{
		Order:          ord,
		Notional:       float64(ord.Quantity) * ord.Price,
		RealizedProfit: realizedSlice,
		HoldingPeriod:  holding,
		Settings:       *p.settings,
	}
	brokerage := p.costCalc().Brokerage(ctx)
	txnTax := p.costCalc().TransactionTax(ctx)
	cgt := p.costCalc().CapitalGainsTax(ctx)
	p.totalBrokerage += brokerage
	p.brokerageSellSide += brokerage
	p.totalTransactionTax += txnTax
	p.transactionTaxSellSide += txnTax
	p.totalCapitalGainsTax += cgt

	if pos.Status == Closed {
		p.closedPositions = append(p.closedPositions, pos)
		delete(p.openPositions, ord.Instrument)
	}

	p.orderHistory = append(p.orderHistory, ord)
	return nil
}

func (p *Portfolio) updatePosition(pos *Position, currentPrice float64) {
	pos.UpdatePrice(currentPrice)
}

// GetPositionMetrics returns a snapshot of performance metrics for the open
// position identified by instrument. Returns an error if no such position exists.
func (p *Portfolio) GetPositionMetrics(instrument string) (*PositionMetrics, error) {
	pos, exists := p.openPositions[instrument]
	if !exists {
		return nil, fmt.Errorf("no position found for %s", instrument)
	}

	return &PositionMetrics{
		ROI:           pos.ROI(),
		Duration:      pos.Duration(),
		MaxDrawdown:   pos.MaxDrawdown,
		RealizedPnL:   pos.RealizedPnL,
		UnrealizedPnL: pos.UnrealizedPnL,
	}, nil
}

// PositionMetrics is a read-only snapshot of key performance indicators for a
// single open position.
type PositionMetrics struct {
	ROI           float64
	Duration      time.Duration
	MaxDrawdown   float64
	RealizedPnL   float64
	UnrealizedPnL float64
}

// GetPortfolioStats computes and returns aggregate statistics across all open
// and closed positions. Prefer the shorter alias [Portfolio.Stats].
func (p *Portfolio) GetPortfolioStats() PortfolioStats {
	var stats PortfolioStats
	stats.TotalValue = p.Value()
	stats.Cash = p.Cash()
	stats.OpenPositions = len(p.openPositions)
	stats.ClosedPositions = len(p.closedPositions)

	for _, pos := range p.openPositions {
		if pos.UnrealizedPnL > 0 {
			stats.WinningPositions++
		} else {
			stats.LosingPositions++
		}
		stats.TotalUnrealizedPnL += pos.UnrealizedPnL
	}

	for _, pos := range p.closedPositions {
		if pos.RealizedPnL > 0 {
			stats.WinningTrades++
		} else {
			stats.LosingTrades++
		}
		stats.TotalRealizedPnL += pos.RealizedPnL
	}

	stats.TotalBrokerage = p.totalBrokerage
	stats.BrokerageBuySide = p.brokerageBuySide
	stats.BrokerageSellSide = p.brokerageSellSide
	stats.TotalTransactionTax = p.totalTransactionTax
	stats.TransactionTaxBuySide = p.transactionTaxBuySide
	stats.TransactionTaxSellSide = p.transactionTaxSellSide
	stats.TotalCapitalGainsTax = p.totalCapitalGainsTax
	stats.TotalInvestment = p.totalInvestment

	return stats
}

// PortfolioStats is a value snapshot of portfolio-wide performance and cost
// accumulators, produced by [Portfolio.Stats] or [Portfolio.GetPortfolioStats].
type PortfolioStats struct {
	TotalValue         float64
	Cash               float64
	OpenPositions      int
	ClosedPositions    int
	WinningPositions   int
	LosingPositions    int
	WinningTrades      int
	LosingTrades       int
	TotalUnrealizedPnL float64
	TotalRealizedPnL   float64

	// Cost accumulators
	TotalBrokerage        float64
	BrokerageBuySide      float64
	BrokerageSellSide     float64
	TotalTransactionTax   float64
	TransactionTaxBuySide float64
	TransactionTaxSellSide float64
	TotalCapitalGainsTax  float64
	TotalInvestment       float64
}
