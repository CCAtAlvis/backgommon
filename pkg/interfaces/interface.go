package interfaces

import (
	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

// Strategy defines the interface for trading strategies
type Strategy interface {
	OnTick(data map[string]core.Candle) []portfolio.Order
	SetPortfolio(portfolio PortfolioManager)
	OnOrderFilled(order portfolio.Order)
	OnPositionOpened(position portfolio.Position)
	OnPositionClosed(position portfolio.Position)
}

// PortfolioManager defines portfolio management operations
type PortfolioManager interface {
	ProcessOrder(portfolio.Order) error
	UpdatePositions(map[string]float64)
	Value() float64
	Cash() float64
	Positions() map[string]*portfolio.Position
}

// RiskManager defines risk management operations
type RiskManager interface {
	ValidateOrder(PortfolioManager, portfolio.Order) error
	CheckPositionExits(PortfolioManager, map[string]float64) []portfolio.Order
}

// Indicator defines the interface for all technical indicators
type Indicator interface {
	// Calculate computes the indicator value for a series of candles
	Calculate(candles []core.Candle) []any
	// Name returns the unique identifier for this indicator
	Name() string
	// Dependencies returns the list of indicators this indicator depends on
	Dependencies() []Indicator
}
