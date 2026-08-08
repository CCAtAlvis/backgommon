// Package strategy provides base implementations for strategy development.
// Most users will embed [BaseStrategy] in their own struct so they only need to
// override the methods they care about (typically OnTick).
package strategy

import (
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

// BaseStrategy satisfies the [interfaces.Strategy] interface with no-op
// defaults. Users embed it in their own strategy struct to avoid implementing
// every method — typically only OnTick needs a real implementation.
//
// BaseStrategy also satisfies the dayLifecycle interface (OnDayStart/OnDayEnd)
// that the runner detects via type assertion, so day-boundary hooks are
// automatically available for any strategy that embeds it.
//
// Example:
//
//	type MyStrategy struct {
//	    strategy.BaseStrategy
//	}
//
//	func (s *MyStrategy) OnTick(data map[string]core.Candle) []portfolio.Order {
//	    // your trading logic here
//	}
type BaseStrategy struct {
	Portfolio interfaces.PortfolioManager
	// Settings is a generic holder for strategy-specific configuration (e.g.
	// indicator periods, position sizing rules). The runner serializes it into
	// the report's config.json when WithReportSettings is used.
	Settings interface{}
}

// OnTick is called for each new data point
func (s *BaseStrategy) OnTick(data map[string]core.Candle) []portfolio.Order {
	return nil
}

// SetPortfolio injects the portfolio manager so the strategy can query
// positions, cash, and account value during OnTick. Called automatically by the
// runner during New().
func (s *BaseStrategy) SetPortfolio(portfolio interfaces.PortfolioManager) {
	s.Portfolio = portfolio
}

// OnOrderFilled is called when an order is filled
func (s *BaseStrategy) OnOrderFilled(ord portfolio.Order) {}

// OnPositionOpened is called when a new position is opened
func (s *BaseStrategy) OnPositionOpened(pos portfolio.Position) {}

// OnPositionClosed is called when a position is closed
func (s *BaseStrategy) OnPositionClosed(pos portfolio.Position) {}

// OnDayStart is called at the start of each trading day
func (s *BaseStrategy) OnDayStart(date time.Time) {}

// OnDayEnd is called at the end of each trading day
func (s *BaseStrategy) OnDayEnd(date time.Time) {}
