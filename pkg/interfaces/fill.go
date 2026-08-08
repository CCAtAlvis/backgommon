// Package interfaces defines core contracts for strategies, portfolio, risk, indicators, and execution.
//
// Pluggable simulation hooks (costs, slippage, cash flow) live in pkg/portfolio, pkg/execution, and pkg/risk
// to avoid import cycles — see docs/extending-the-framework.md.
//
// Order fill simulation: see FillPricer and FillContext below, and docs/order-fill.md.
package interfaces

import (
	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

// FillContext supplies bar data used to simulate an order's execution price.
// The runner constructs this when an order has Price == 0; strategy authors normally
// do not build FillContext themselves.
//
// CurrentBar is the instrument's candle at the signal bar (time T).
// NextBar is that instrument's candle at T+1, or nil on the last backtest row.
// AllBars and NextBars are the full multi-symbol rows at T and T+1 for custom pricers.
//
// See docs/order-fill.md for field-by-field explanation and examples.
type FillContext struct {
	Order      portfolio.Order
	CurrentBar core.Candle
	NextBar    *core.Candle
	AllBars    map[string]core.Candle
	NextBars   map[string]core.Candle
}

// FillPricer resolves the execution price for an order during a backtest.
// When Order.Price is already set (> 0), the runner skips FillPricer and uses that price.
//
// Configure via runner.WithFillMode or runner.WithFillPricer — see docs/order-fill.md.
type FillPricer interface {
	FillPrice(ctx FillContext) (float64, error)
}
