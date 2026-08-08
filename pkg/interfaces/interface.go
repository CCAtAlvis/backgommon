package interfaces

import (
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

// DrawdownAction is the runner's instruction set after evaluating portfolio drawdown.
// Returned by RiskManager.CheckDrawdown on every bar; the runner uses it to decide
// whether to suppress new entries or force-liquidate all positions.
type DrawdownAction struct {
	// BlockNewEntries prevents new Entry orders from being processed this bar.
	// Exit orders (stop-loss, take-profit, risk exits) are still allowed.
	BlockNewEntries bool

	// LiquidateAll forces immediate exit of every open position. When true,
	// the runner generates market exit orders for all holdings before calling OnTick.
	LiquidateAll bool

	// DrawdownRate is the current portfolio drawdown as a fraction in [0, 1].
	// 0 means no drawdown (portfolio at peak); 0.15 means 15% below peak equity.
	DrawdownRate float64
}

// Strategy is the primary user-facing interface for defining trading logic.
// The runner calls methods in a strict per-bar lifecycle:
//
//  1. SetPortfolio — called once at construction (runner.New) to inject portfolio access.
//  2. OnTick — called on every bar within [startTime, endTime]. Return orders to trade.
//  3. OnOrderFilled — called after each order passes risk validation and is processed.
//  4. OnPositionOpened — called when an Entry order creates a new Position.
//  5. OnPositionClosed — called when an Exit order reduces a Position's quantity to zero.
//
// Embed strategy.BaseStrategy to get no-op defaults for all methods except OnTick.
type Strategy interface {
	// OnTick receives the current bar's candle data keyed by instrument symbol.
	// Return a slice of orders to submit; return nil or empty to do nothing.
	// The runner processes returned orders in slice order after applying fill prices.
	OnTick(data map[string]core.Candle) []portfolio.Order

	// SetPortfolio is called once by runner.New to give the strategy read access
	// to portfolio state (cash, positions, value). Strategies should store the
	// reference but avoid mutating portfolio state directly — use returned orders instead.
	SetPortfolio(portfolio PortfolioManager)

	// OnOrderFilled is invoked after an order has been validated by the RiskManager
	// and successfully processed by the PortfolioManager. Use it to update internal
	// strategy bookkeeping (e.g., tracking entry prices, adjusting stop levels).
	OnOrderFilled(order portfolio.Order)

	// OnPositionOpened is called when a filled Entry order results in a new Position.
	// The position argument is a value copy — mutations do not affect the portfolio.
	OnPositionOpened(position portfolio.Position)

	// OnPositionClosed is called when a filled Exit order fully closes a Position
	// (quantity reaches zero). Inspect position.RealizedPnL for trade outcome.
	// The position argument is a value copy.
	OnPositionClosed(position portfolio.Position)
}

// PortfolioManager is the contract the runner and strategy use to interact with
// the simulated trading account. The default implementation is pkg/portfolio.Portfolio.
type PortfolioManager interface {
	// ProcessOrder applies a filled order to the portfolio: opens new positions on
	// Entry orders, reduces or closes positions on Exit orders, and adjusts cash
	// according to the configured CashFlowCalculator and CostCalculator.
	ProcessOrder(portfolio.Order) error

	// UpdatePositions refreshes unrealized P&L and risk metrics for all open positions
	// using the latest prices (keyed by instrument symbol → close price).
	// Called by the runner at the start of each bar before risk checks.
	UpdatePositions(map[string]float64)

	// Value returns the total portfolio equity: cash + sum of open position values
	// (including unrealized P&L and leverage effects).
	Value() float64

	// Cash returns the current available cash balance.
	Cash() float64

	// Positions returns a shallow copy of the open positions map keyed by instrument.
	// The map itself is safe to iterate and modify without affecting portfolio state.
	// The Position pointers still reference live structs — read but do not mutate fields.
	Positions() map[string]*portfolio.Position

	// InitialCapital returns the starting capital configured at portfolio creation.
	InitialCapital() float64

	// Stats computes and returns aggregate portfolio statistics (win rate, total
	// trades, realized P&L, etc.) based on the current state of all positions.
	Stats() portfolio.PortfolioStats

	// ClosedPositions returns the historical list of all fully closed positions
	// in chronological order. Used by the runner for results computation and reporting.
	ClosedPositions() []*portfolio.Position

	// EstimateEntryCash returns the cash that would be deducted if ord were processed
	// as an Entry (margin + brokerage + transaction tax). Used by strategies for
	// fee-aware position sizing before submitting multi-order batches.
	EstimateEntryCash(portfolio.Order) float64
}

// RiskManager validates orders and enforces risk constraints on every bar.
// The runner calls its methods in this order each tick:
//
//  1. CheckDrawdown — evaluate portfolio-level drawdown; may block entries or liquidate.
//  2. CheckPositionExits — generate exit orders for positions hitting stop/target/trailing.
//  3. ValidateOrder — called per-order before ProcessOrder; reject if risk limits breached.
type RiskManager interface {
	// ValidateOrder checks whether a single order is permissible given current
	// portfolio state. Return a non-nil error to reject the order. By default the
	// runner skips rejected orders and continues; with StrictOrderProcessing it
	// propagates the error and aborts the backtest.
	ValidateOrder(PortfolioManager, portfolio.Order) error

	// CheckPositionExits evaluates all open positions against configured exit
	// conditions (stop-loss, take-profit, trailing stop) using the latest prices.
	// Returns exit orders for positions that should be closed this bar.
	CheckPositionExits(PortfolioManager, map[string]float64) []portfolio.Order

	// CheckDrawdown evaluates portfolio-level drawdown and returns an action
	// describing whether to block new entries, liquidate, or proceed normally.
	// Called once per bar before strategy OnTick.
	CheckDrawdown(PortfolioManager, time.Time) DrawdownAction
}

// Indicator is the contract for technical indicators that can be applied to
// timeseries data via TimeseriesTable.ApplyIndicator. Indicators are computed
// over a full candle series and their results are stored on each Candle's
// indicator map (see core.Candle.SetIndicator).
//
// Implement this interface for custom indicators; for simple function-based
// indicators, see indicators.CustomIndicator.
type Indicator interface {
	// Calculate computes indicator values for the entire candle series.
	// Returns a slice of length len(candles) — one value per candle. Use nil
	// for candles where the indicator cannot be computed (insufficient lookback).
	Calculate(candles []core.Candle) []any

	// Name returns a unique identifier for this indicator instance (e.g., "SMA_20").
	// This name is used as the key in Candle.SetIndicator / GetIndicator.
	Name() string

	// Dependencies returns indicators that must be computed before this one.
	// The framework applies dependencies recursively before calling Calculate.
	// Return nil if this indicator has no dependencies.
	Dependencies() []Indicator
}
