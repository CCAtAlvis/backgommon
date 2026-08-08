// Package risk enforces portfolio-level risk limits, position sizing, and
// automatic exit conditions (stop-loss, take-profit, trailing stop) for the
// backgommon backtesting framework. The central type is [Manager], which wraps
// a [Settings] struct and delegates position sizing and drawdown evaluation to
// injectable [PositionSizer] and [DrawdownPolicy] hooks. Use [New] with
// functional [Option] values to construct a configured Manager.
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

// Logger is an optional interface for risk event logging.
// When nil, risk events are silently discarded.
type Logger interface {
	Printf(format string, args ...interface{})
}

// StdLogger writes risk events to stdout via fmt.Printf.
// Use risk.WithLogger(risk.StdLogger{}) to restore the original logging behavior.
type StdLogger struct{}

func (StdLogger) Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// Manager is the primary risk gate for the runner loop. It validates every
// order against portfolio-level limits (allocation caps, leverage caps,
// drawdown locks) and evaluates per-position exit conditions on each bar.
//
// Behaviour is customised through injectable hooks:
//   - [PositionSizer] — controls how many shares/units to buy (default:
//     [StandardPositionSizer] using risk-per-trade and stop-loss rate).
//   - [DrawdownPolicy] — decides what happens when peak-to-trough equity
//     loss exceeds the configured threshold (default: [StandardDrawdownPolicy]).
//
// Use [New] with [WithPositionSizer] and/or [WithDrawdownPolicy] to override.
type Manager struct {
	settings        *Settings
	peakEquity      float64
	lockUntil       time.Time
	blockNewEntries bool

	positionSizer  PositionSizer
	drawdownPolicy DrawdownPolicy
	logger         Logger
}

func (m *Manager) logf(format string, args ...interface{}) {
	if m.logger != nil {
		m.logger.Printf(format, args...)
	}
}

// Settings holds all declarative risk parameters for a backtest. Fields are
// grouped into portfolio-level limits, position-level limits, order-level
// defaults, and metrics configuration. Zero values disable the corresponding
// check (e.g. MaxLeverage == 0 means leverage is uncapped).
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

// New creates a new risk manager with optional hooks.
func New(settings *Settings, opts ...Option) *Manager {
	m := &Manager{settings: settings}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Settings returns the risk settings for external inspection (e.g., metrics computation).
func (m *Manager) Settings() *Settings {
	return m.settings
}

// ValidateOrder checks whether ord satisfies the current risk constraints.
// For entry orders it verifies that:
//   - New entries are not blocked by an active drawdown lock.
//   - The position's notional value does not exceed MaxPositionAllocationRate
//     of the current portfolio value.
//   - The order's leverage does not exceed MaxLeverage.
//
// Exit orders bypass the drawdown-block check. Returns a descriptive error
// when a constraint is violated; nil means the order is acceptable.
func (m *Manager) ValidateOrder(portfolioManager interfaces.PortfolioManager, ord portfolio.Order) error {
	if ord.Type == portfolio.Entry && m.blockNewEntries {
		return fmt.Errorf("new entries blocked: portfolio drawdown limit active")
	}

	positionValue := float64(ord.Quantity) * ord.Price

	portfolioValue := portfolioManager.Value()
	if m.settings.MaxPositionAllocationRate > 0 {
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

// CheckPositionExits iterates all open positions and evaluates stop-loss,
// take-profit, and trailing-stop exit conditions against the supplied
// prices. For every position that triggers, an exit [portfolio.Order] is
// appended to the returned slice and an informational line is printed to
// stdout. Positions whose instrument is absent from prices are skipped.
//
// Note: this method mutates position state (e.g. TrailingStopHigh) as a
// side-effect of the trailing-stop check — see [Manager.checkExitConditions].
func (m *Manager) CheckPositionExits(portfolioManager interfaces.PortfolioManager, prices map[string]float64) []portfolio.Order {
	var exitOrders []portfolio.Order

	for _, pos := range portfolioManager.Positions() {
		currentPrice, exists := prices[pos.Instrument]
		if !exists {
			continue
		}

		if shouldExit, reason := m.checkExitConditions(pos, currentPrice); shouldExit {
			exitOrders = append(exitOrders, createExitOrder(pos, reason))
			m.logf("INFO: Exit condition met for %s: %s. Current Price: %.2f\n", pos.Instrument, reason, currentPrice)
		}
	}

	return exitOrders
}
