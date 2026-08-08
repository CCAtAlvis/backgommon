package indicators

import (
	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
)

// CustomIndicator allows users to create ad-hoc indicators without defining a
// new type. Supply a name, a calculation function, and optional dependencies,
// and the framework treats it like any built-in indicator.
//
// Example — a simple "spread" indicator that depends on two SMAs:
//
//	sma20 := indicators.NewSMA(20)
//	sma50 := indicators.NewSMA(50)
//
//	spread := indicators.NewCustomIndicator("SMA_Spread", func(candles []core.Candle) []any {
//	    result := make([]any, len(candles))
//	    for i, c := range candles {
//	        sma20Val := c.GetIndicator("SMA_20")
//	        sma50Val := c.GetIndicator("SMA_50")
//	        if sma20Val != nil && sma50Val != nil {
//	            result[i] = sma20Val.(float64) - sma50Val.(float64)
//	        }
//	    }
//	    return result
//	}, []interfaces.Indicator{sma20, sma50})
type CustomIndicator struct {
	name     string
	calcFunc func([]core.Candle) []any
	deps     []interfaces.Indicator
}

// NewCustomIndicator creates a custom indicator with the given name,
// calculation function, and optional dependency list. Dependencies are
// calculated first by TimeseriesTable.ApplyIndicators, so calcFunc can safely
// read their values from each Candle's indicator map.
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

// Dependencies returns the indicators that must be calculated before this one.
// TimeseriesTable.ApplyIndicators uses this to topologically sort indicator
// evaluation order. Returns nil when the custom indicator has no prerequisites.
func (c *CustomIndicator) Dependencies() []interfaces.Indicator {
	return c.deps
}
