package indicator_strategy

import (
	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/indicators"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
	"github.com/CCAtAlvis/backgommon/pkg/strategy"
)

// IndicatorStrategy demonstrates how to use indicators in a strategy
type IndicatorStrategy struct {
	strategy.BaseStrategy

	// Indicators
	sma20     *indicators.SMA
	sma50     *indicators.SMA
	customInd interfaces.Indicator

	// Strategy parameters
	symbol string
}

// NewIndicatorStrategy creates a new instance of the indicator strategy
func NewIndicatorStrategy(symbol string) *IndicatorStrategy {
	s := &IndicatorStrategy{
		symbol: symbol,
	}

	// Initialize indicators
	s.sma20 = indicators.NewSMA(20)
	s.sma50 = indicators.NewSMA(50)

	// Create a custom momentum indicator
	s.customInd = indicators.NewCustomIndicator("Momentum", func(candles []core.Candle) []any {
		result := make([]any, len(candles))
		for i := range candles {
			if i < 10 {
				result[i] = nil
				continue
			}
			result[i] = candles[i].Close - candles[i-10].Close
		}
		return result
	}, nil)

	return s
}

// OnTick reads pre-computed indicator values from the candle data.
func (s *IndicatorStrategy) OnTick(data map[string]core.Candle) []portfolio.Order {
	candle, ok := data[s.symbol]
	if !ok {
		return nil
	}

	// Verify indicators are available (set via TimeseriesTable.ApplyIndicators)
	_, _ = candle.GetIndicator(s.sma20.Name())
	_, _ = candle.GetIndicator(s.sma50.Name())
	_, _ = candle.GetIndicator(s.customInd.Name())

	return nil
}
