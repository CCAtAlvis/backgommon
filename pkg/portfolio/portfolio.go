package portfolio

import (
	"fmt"
	"time"
)

// Portfolio manages positions and cash
type Portfolio struct {
	cash            float64
	openPositions   map[string]*Position
	closedPositions []*Position
	orderHistory    []Order
	settings        *Settings
}

// Settings contains portfolio-specific settings
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

// New creates a new portfolio with given settings
func New(settings *Settings) *Portfolio {
	return &Portfolio{
		cash:            settings.InitialCapital,
		openPositions:   make(map[string]*Position),
		closedPositions: make([]*Position, 0),
		orderHistory:    make([]Order, 0),
		settings:        settings,
	}
}

// ProcessOrder handles order execution
func (p *Portfolio) ProcessOrder(ord Order) error {
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

// UpdatePositions updates all positions with current prices
func (p *Portfolio) UpdatePositions(prices map[string]float64) {
	for instrument, pos := range p.openPositions {
		if price, ok := prices[instrument]; ok {
			p.updatePosition(pos, price)
		}
	}
}

// Value returns total portfolio value
func (p *Portfolio) Value() float64 {
	value := p.cash
	for _, pos := range p.openPositions {
		value += pos.UnrealizedPnL // This needs to be more sophisticated with short positions
	}
	// TODO: Consider margin accounts, short sale proceeds/liabilities for a more accurate value.
	return value
}

// Cash returns available cash
func (p *Portfolio) Cash() float64 {
	return p.cash
}

// Positions returns all open positions
func (p *Portfolio) Positions() map[string]*Position {
	return p.openPositions
}

// Internal methods

func (p *Portfolio) validateOrder(ord Order) error {
	switch ord.Type {
	case Entry:
		if !p.settings.EnableShorts && ord.Side == Short {
			return fmt.Errorf("short positions not allowed")
		}

		requiredCash := float64(ord.Quantity) * ord.Price // This is simplified; true cost depends on leverage & margin
		if ord.Side == Long && p.settings.DefaultLeverage > 0 && ord.Leverage > 0 {
			requiredCash /= ord.Leverage // Assuming ord.Leverage is used if > 0, else DefaultLeverage
		} else if ord.Side == Long && p.settings.DefaultLeverage > 0 {
			requiredCash /= p.settings.DefaultLeverage
		}
		// TODO: Add margin calculation for short positions here for cash validation

		if requiredCash > p.cash {
			return fmt.Errorf("insufficient cash: have %.2f, need %.2f (considering leverage/margin)", p.cash, requiredCash)
		}

		// Removed MaxPositions check as per user feedback to make it strategy-dependent

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
		pos = newPos
	} else {
		if err := pos.AddOrder(ord); err != nil {
			return err
		}
	}

	// TODO: Adjust cash based on actual execution price (with slippage, brokerage)
	// TODO: For longs, cash -= (executedPrice * quantity) / leverage + brokerage
	// TODO: For shorts, cash -= (executedPrice * quantity * initialMarginRate) + brokerage
	// The current cash deduction is simplified and doesn't account for leverage or full costs.
	cost := float64(ord.Quantity) * ord.Price // Placeholder for actual cost
	if p.settings.DefaultLeverage > 1.0 && ord.Side == Long {
		// This simple division isn't quite right for margin accounting but approximates leveraged cost.
		cost /= p.settings.DefaultLeverage
	}
	p.cash -= cost
	p.orderHistory = append(p.orderHistory, ord)
	return nil
}

func (p *Portfolio) handleExitOrder(ord Order) error {
	pos, exists := p.openPositions[ord.Instrument]
	if !exists {
		return fmt.Errorf("no position found for %s", ord.Instrument)
	}

	if err := pos.AddOrder(ord); err != nil {
		return err
	}

	// TODO: Adjust cash based on actual execution price (with slippage, brokerage) and P&L
	// TODO: cash += (executedPrice * quantity) +/- PnL - brokerage (for longs)
	// TODO: cash += (shortProceeds + initialMarginBlocked) +/- PnL - brokerage (for shorts)
	// The current cash addition is simplified.
	proceeds := float64(ord.Quantity) * ord.Price // Placeholder for actual proceeds
	p.cash += proceeds

	if pos.Status == Closed {
		// TODO: Handle tax calculation on realized P&L here
		// TODO: Handle profit pocketing here
		p.closedPositions = append(p.closedPositions, pos)
		delete(p.openPositions, ord.Instrument)
	}

	p.orderHistory = append(p.orderHistory, ord)
	return nil
}

func (p *Portfolio) updatePosition(pos *Position, currentPrice float64) {
	pos.UpdatePrice(currentPrice)
}

// GetPositionMetrics returns detailed metrics for a position
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

// PositionMetrics holds detailed position metrics
type PositionMetrics struct {
	ROI           float64
	Duration      time.Duration
	MaxDrawdown   float64
	RealizedPnL   float64
	UnrealizedPnL float64
}

// GetPortfolioStats returns overall portfolio statistics
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

	return stats
}

// PortfolioStats holds overall portfolio statistics
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
}
