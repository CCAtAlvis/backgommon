package risk

import (
	"fmt"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

// MaxDrawdownMode defines the action to take when MaxPortfolioDrawdownRate is breached.
type MaxDrawdownMode string

const (
	// NoAction means no specific action is taken by the framework, relies on strategy.
	NoAction MaxDrawdownMode = "NoAction"
	// AlertOnly means an alert/log is generated, but trading continues.
	AlertOnly MaxDrawdownMode = "AlertOnly"
	// StopNewTrades means no new positions can be opened, existing ones can be managed/closed.
	StopNewTrades MaxDrawdownMode = "StopNewTrades"
	// LiquidateAllPositions means all open positions are immediately liquidated, and no new trades are allowed.
	LiquidateAllPositions MaxDrawdownMode = "LiquidateAllPositions"
)

// Manager handles risk management
type Manager struct {
	settings *Settings
}

// Settings contains risk management settings
type Settings struct {
	// --- Portfolio-Level Risk ---

	// MaxPortfolioDrawdownRate is the maximum allowed percentage drop in total portfolio value
	// from its peak before triggering a specified action. Expressed as a positive decimal.
	// e.g., 0.20 (for a 20% maximum drawdown from peak equity)
	MaxPortfolioDrawdownRate float64

	// MaxDrawdownMode specifies what action the framework should take if MaxPortfolioDrawdownRate is breached.
	// See MaxDrawdownMode type for options.
	// e.g., StopNewTrades
	MaxDrawdownMode MaxDrawdownMode

	// DrawdownLockDuration is how long the account remains locked (no new trades allowed) if triggered.
	// e.g., 24 * time.Hour (for a 1-day trading halt)
	DrawdownLockDuration time.Duration

	// --- Position-Level Risk ---

	// MaxLeverage is the absolute maximum leverage allowed for any single order across the portfolio.
	// This caps the leverage specified in portfolio.Settings.DefaultLeverage or an order's specific leverage.
	// e.g., 5.0 (for 5x maximum leverage)
	MaxLeverage float64

	// MaxPositionAllocationRate is the maximum size a single position can represent as a percentage
	// of the total portfolio value at the time of entry. Expressed as a decimal.
	// e.g., 0.10 (for a single position not exceeding 10% of portfolio value)
	MaxPositionAllocationRate float64

	// RiskPerTradeRate is the maximum percentage of total portfolio capital that the strategy
	// intends to risk on a single trade. Used for position sizing with a stop-loss.
	// Expressed as a decimal. e.g., 0.01 (for risking 1% of portfolio capital per trade)
	RiskPerTradeRate float64

	// --- Order-Level Risk Controls (Defaults & Enables) ---

	// EnableStopLoss flags whether stop-loss mechanisms are active.
	// e.g., true
	EnableStopLoss bool
	// DefaultStopLossRate is the default stop-loss percentage from the entry price.
	// Expressed as a positive decimal. e.g., 0.05 (for a 5% stop-loss)
	DefaultStopLossRate float64

	// EnableTakeProfit flags whether take-profit mechanisms are active.
	// e.g., true
	EnableTakeProfit bool
	// DefaultTakeProfitRate is the default take-profit percentage from the entry price.
	// Expressed as a positive decimal. e.g., 0.10 (for a 10% take-profit)
	DefaultTakeProfitRate float64

	// EnableTrailingStop flags whether trailing stop-loss mechanisms are active.
	// e.g., true
	EnableTrailingStop bool
	// DefaultTrailingStopRate is the default percentage for a trailing stop-loss.
	// Expressed as a positive decimal. e.g., 0.03 (for a 3% trailing stop)
	DefaultTrailingStopRate float64

	// --- Metrics Related ---

	// MetricsRiskFreeAnnualRate is the annualized risk-free rate for calculating performance metrics (e.g., Sharpe Ratio).
	// Expressed as a decimal. e.g., 0.02 (for 2% annualized RFR)
	MetricsRiskFreeAnnualRate float64
}

// New creates a new risk manager
func New(settings *Settings) *Manager {
	return &Manager{settings: settings}
}

// ValidateOrder checks if an order meets risk requirements
func (m *Manager) ValidateOrder(portfolioManager interfaces.PortfolioManager, ord portfolio.Order) error {
	// Position size checks
	positionValue := float64(ord.Quantity) * ord.Price

	// MinPositionSize check removed as the setting is no longer part of risk.Settings.
	// Strategies should enforce their own minimums if required.

	portfolioValue := portfolioManager.Value()    // TODO: Ensure this reflects available capital for sizing
	if m.settings.MaxPositionAllocationRate > 0 { // Check if the setting is configured
		maxSizeAllowedByPercent := portfolioValue * m.settings.MaxPositionAllocationRate
		if positionValue > maxSizeAllowedByPercent {
			return fmt.Errorf("position size %.2f (%.2f%% of portfolio) exceeds maximum allowed %.2f (%.2f%% of portfolio)",
				positionValue, (positionValue/portfolioValue)*100, maxSizeAllowedByPercent, m.settings.MaxPositionAllocationRate*100)
		}
	}

	// Leverage check
	if ord.Leverage > m.settings.MaxLeverage && m.settings.MaxLeverage > 0 { // m.settings.MaxLeverage > 0 means it's enforced
		return fmt.Errorf("order leverage %.2fx exceeds portfolio maximum %.2fx",
			ord.Leverage, m.settings.MaxLeverage)
	}

	return nil
}

// CheckPositionExits checks for exit conditions
func (m *Manager) CheckPositionExits(portfolioManager interfaces.PortfolioManager, prices map[string]float64) []portfolio.Order {
	var exitOrders []portfolio.Order

	for _, pos := range portfolioManager.Positions() {
		currentPrice, exists := prices[pos.Instrument]
		if !exists {
			continue
		}

		if shouldExit, reason := m.checkExitConditions(pos, currentPrice); shouldExit {
			// TODO: Create a proper exit order. This function signature might need to change
			// or we need a helper to create an order of appropriate type (market/limit) and quantity.
			// For now, assuming a full exit market order.
			exitOrders = append(exitOrders, portfolio.Order{
				Instrument: pos.Instrument,
				Side:       pos.Side.Opposite(), // This needs to be robust
				Type:       portfolio.Exit,
				Quantity:   pos.Quantity,
				Price:      currentPrice, // For market order, this would be fill price
				// Leverage for exit orders is typically not applicable or 1x.
			})
			fmt.Printf("INFO: Exit condition met for %s: %s. Current Price: %.2f\n", pos.Instrument, reason, currentPrice)
		}
	}

	return exitOrders
}

// Additional risk management functions...
