package strategy

import (
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

// BaseStrategy provides default implementations for strategy methods
type BaseStrategy struct {
	Portfolio interfaces.PortfolioManager
	Settings  interface{}
}

// OnTick is called for each new data point
func (s *BaseStrategy) OnTick(data map[string]core.Candle) []portfolio.Order {
	return nil
}

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
