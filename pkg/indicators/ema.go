package indicators

import (
	"fmt"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
)

// EMA implements Exponential Moving Average indicator.
// EMA gives more weight to recent prices, making it more responsive to new information
// than a simple moving average (SMA).
//
// The EMA is calculated using the formula:
// EMA = (Current Price - Previous EMA) * Multiplier + Previous EMA
//
// The Multiplier is calculated as:
// Multiplier = 2 / (Period + 1)
//
// Example usage:
//
//	// Create a 20-period EMA
//	ema := indicators.NewEMA(20)
//
//	// Calculate EMA value
//	value := ema.Calculate(candles)
//	emaValue := value.Value()
//
//	// Use as dependency in other indicators
//	macd := indicators.NewMACD(12, 26, 9) // uses EMA internally
type EMA struct {
	period int
}

// NewEMA creates a new EMA indicator with the specified period.
// The period determines how many candles are used in the initial SMA calculation
// and affects the weighting multiplier.
func NewEMA(period int) *EMA {
	return &EMA{period: period}
}

// Calculate computes the EMA value for the given candles.
// Returns a slice of values, one per candle. If there are fewer candles than the period at a given index, returns nil for that index.
func (e *EMA) Calculate(candles []core.Candle) []any {
	result := make([]any, len(candles))
	if len(candles) == 0 {
		return result
	}
	multiplier := 2.0 / float64(e.period+1)
	var ema float64
	for i := range candles {
		if i+1 < e.period {
			result[i] = nil
			continue
		}
		if i+1 == e.period {
			// Start with SMA for the first EMA value
			sum := 0.0
			for j := 0; j < e.period; j++ {
				sum += candles[j].Close
			}
			ema = sum / float64(e.period)
			result[i] = ema
			continue
		}
		ema = (candles[i].Close-ema)*multiplier + ema
		result[i] = ema
	}
	return result
}

// Name returns the identifier for this EMA instance
func (e *EMA) Name() string {
	return fmt.Sprintf("EMA_%d", e.period)
}

// Dependencies returns empty slice as EMA has no dependencies
func (e *EMA) Dependencies() []interfaces.Indicator {
	return nil
}
