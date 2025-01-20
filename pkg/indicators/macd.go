package indicators

import (
	"fmt"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
)

// MACD implements Moving Average Convergence Divergence indicator.
// MACD is a trend-following momentum indicator that shows the relationship
// between two moving averages of a price.
//
// The MACD is calculated by subtracting the longer-period EMA from the shorter-period EMA.
// The "signal line" is an EMA of the MACD line.
// The "histogram" shows the difference between MACD and signal line.
//
// Example usage:
//
//	// Create MACD with standard periods (12, 26, 9)
//	macd := indicators.NewMACD(12, 26, 9)
//
//	// Calculate MACD values
//	value := macd.Calculate(candles)
//	macdLine := value.Value()           // or use value.(MACDValue).Value()
//	signalLine := value.(MACDValue).Signal()
//	histogram := value.(MACDValue).Histogram()
//
//	// Apply to table (will automatically calculate required EMAs)
//	table.ApplyIndicator(macd)
type MACD struct {
	fastEMA      *EMA
	slowEMA      *EMA
	signalEMA    *EMA
	fastPeriod   int
	slowPeriod   int
	signalPeriod int
}

// NewMACD creates a new MACD indicator with the specified periods.
// Parameters:
//   - fastPeriod: Period for the fast EMA (typically 12)
//   - slowPeriod: Period for the slow EMA (typically 26)
//   - signalPeriod: Period for the signal line EMA (typically 9)
func NewMACD(fastPeriod, slowPeriod, signalPeriod int) *MACD {
	return &MACD{
		fastEMA:      NewEMA(fastPeriod),
		slowEMA:      NewEMA(slowPeriod),
		signalEMA:    NewEMA(signalPeriod),
		fastPeriod:   fastPeriod,
		slowPeriod:   slowPeriod,
		signalPeriod: signalPeriod,
	}
}

// Calculate computes MACD values for the given candles.
// Returns a slice of MACDValue (or nil for insufficient data), one per candle.
func (m *MACD) Calculate(candles []core.Candle) []any {
	result := make([]any, len(candles))
	if len(candles) == 0 {
		return result
	}
	// Precompute fast and slow EMA arrays
	fastArr := m.fastEMA.Calculate(candles)
	slowArr := m.slowEMA.Calculate(candles)
	macdLineArr := make([]float64, len(candles))
	for i := range candles {
		if fastArr[i] == nil || slowArr[i] == nil {
			result[i] = nil
			continue
		}
		macdLineArr[i] = fastArr[i].(float64) - slowArr[i].(float64)
	}
	// Build MACD series as candles for signal EMA
	macdCandles := make([]core.Candle, len(candles))
	for i := range candles {
		macdCandles[i] = core.Candle{
			Time:  candles[i].Time,
			Close: macdLineArr[i],
		}
	}
	signalArr := m.signalEMA.Calculate(macdCandles)
	for i := range candles {
		if fastArr[i] == nil || slowArr[i] == nil || signalArr[i] == nil {
			result[i] = nil
			continue
		}
		macd := macdLineArr[i]
		signal := signalArr[i].(float64)
		histogram := macd - signal
		result[i] = NewMACDValue(macd, signal, histogram)
	}
	return result
}

// Name returns the identifier for this MACD instance
func (m *MACD) Name() string {
	return fmt.Sprintf("MACD_%d_%d_%d", m.fastPeriod, m.slowPeriod, m.signalPeriod)
}

// Dependencies returns the list of indicators this indicator depends on.
// MACD depends on three EMAs:
//   - Fast period EMA
//   - Slow period EMA
//   - Signal line EMA
func (m *MACD) Dependencies() []interfaces.Indicator {
	return []interfaces.Indicator{
		m.fastEMA,
		m.slowEMA,
		m.signalEMA,
	}
}

// MACDValue represents MACD indicator values
type MACDValue struct {
	macd      float64
	signal    float64
	histogram float64
}

func NewMACDValue(macd, signal, histogram float64) MACDValue {
	return MACDValue{
		macd:      macd,
		signal:    signal,
		histogram: histogram,
	}
}

func (m MACDValue) Value() float64 {
	return m.macd
}

// Signal returns the signal line value
func (m MACDValue) Signal() float64 {
	return m.signal
}

// Histogram returns the histogram value
func (m MACDValue) Histogram() float64 {
	return m.histogram
}
