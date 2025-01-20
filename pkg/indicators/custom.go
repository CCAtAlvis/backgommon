package indicators

import (
	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
)

// CustomIndicator allows users to create their own indicators
type CustomIndicator struct {
	name     string
	calcFunc func([]core.Candle) []any
	deps     []interfaces.Indicator
}

// NewCustomIndicator creates a new custom indicator
func NewCustomIndicator(name string, calcFunc func([]core.Candle) []any, deps []interfaces.Indicator) interfaces.Indicator {
	return &CustomIndicator{
		name:     name,
		calcFunc: calcFunc,
		deps:     deps,
	}
}

// Calculate calls the user-provided calculation function
func (c *CustomIndicator) Calculate(candles []core.Candle) []any {
	return c.calcFunc(candles)
}

// Name returns the custom indicator's name
func (c *CustomIndicator) Name() string {
	return c.name
}

func (c *CustomIndicator) Dependencies() []interfaces.Indicator {
	return c.deps
}
