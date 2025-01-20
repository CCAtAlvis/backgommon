package indicators

import (
	"fmt"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
)

// SMA implements Simple Moving Average indicator.
// SMA is calculated by taking the arithmetic mean of a given set of values over a specified period.
// For example, a 20-period SMA would average out the closing prices for the last 20 candles.
//
// Example usage:
//
//	// Create a 20-period SMA
//	sma := indicators.NewSMA(20)
//
//	// Calculate SMA value for a series of candles
//	value := sma.Calculate(candles)
//	smaValue := value.Value() // get the float64 value
//
//	// Use with other indicators
//	macd := indicators.NewMACD(12, 26, 9)
//	table.ApplyIndicators([]core.Indicator{sma, macd})
type SMA struct {
	period int
}

// NewSMA creates a new SMA indicator with the specified period.
// The period determines how many candles are used in the calculation.
// Common periods: 20 (short term), 50 (medium term), 200 (long term)
func NewSMA(period int) *SMA {
	return &SMA{period: period}
}

// Calculate computes the SMA value for the given candles.
// Returns a slice of values, one per candle. If there are fewer candles than the period at a given index, returns nil for that index.
func (s *SMA) Calculate(candles []core.Candle) []any {
	result := make([]any, len(candles))
	for i := range candles {
		if i+1 < s.period {
			result[i] = nil
			continue
		}
		sum := 0.0
		for j := i + 1 - s.period; j <= i; j++ {
			sum += candles[j].Close
		}
		result[i] = sum / float64(s.period)
	}
	return result
}

// Name returns the identifier for this SMA instance
func (s *SMA) Name() string {
	return fmt.Sprintf("SMA_%d", s.period)
}

// Dependencies returns empty slice as SMA has no dependencies
func (s *SMA) Dependencies() []interfaces.Indicator {
	return nil
}
