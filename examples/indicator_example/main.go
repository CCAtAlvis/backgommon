package main

import (
	"fmt"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/indicators"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

func main() {
	// Create sample data
	candles := createSampleData()

	// Create indicators
	sma20 := indicators.NewSMA(20)
	ema12 := indicators.NewEMA(12)
	macd := indicators.NewMACD(12, 26, 9)

	// Method 1: Direct calculation
	fmt.Println("Direct calculation:")
	smaValue, ok := lastFloat(sma20.Calculate(candles))
	if !ok {
		fmt.Println("SMA(20): insufficient data")
	} else {
		fmt.Printf("SMA(20): %.2f\n", smaValue)
	}

	emaValue, ok := lastFloat(ema12.Calculate(candles))
	if !ok {
		fmt.Println("EMA(12): insufficient data")
	} else {
		fmt.Printf("EMA(12): %.2f\n", emaValue)
	}

	macdResult, ok := lastMACD(macd.Calculate(candles))
	if !ok {
		fmt.Println("MACD: insufficient data")
	} else {
		fmt.Printf("MACD: %.2f (Signal: %.2f, Histogram: %.2f)\n",
			macdResult.Value(),
			macdResult.Signal(),
			macdResult.Histogram())
	}

	// Method 2: Using TimeseriesTable
	fmt.Println("\nUsing TimeseriesTable:")
	table := types.NewTimeseriesTable[core.Candle]([]string{"BTC-USD"})

	// Add data to table
	for _, candle := range candles {
		table.AddRow(candle.Time, map[string]core.Candle{
			"BTC-USD": candle,
		})
	}

	// Apply indicators
	// Note: MACD will automatically calculate its EMA dependencies
	table.ApplyIndicatorsToColumn([]interfaces.Indicator{sma20, macd}, "BTC-USD")

	// Get last row's values
	lastRow, _ := table.GetRow(candles[len(candles)-1].Time)
	lastCandle := lastRow["BTC-USD"]

	// Get indicator values
	sma, _ := lastCandle.GetIndicator(sma20.Name())
	if v, ok := sma.(float64); ok {
		smaValue = v
		fmt.Printf("SMA(20): %.2f\n", smaValue)
	}

	macdValue, _ := lastCandle.GetIndicator(macd.Name())
	if v, ok := macdValue.(indicators.MACDValue); ok {
		macdResult = v
		fmt.Printf("MACD: %.2f (Signal: %.2f, Histogram: %.2f)\n",
			macdResult.Value(),
			macdResult.Signal(),
			macdResult.Histogram())
	}
}

func lastFloat(values []any) (float64, bool) {
	for i := len(values) - 1; i >= 0; i-- {
		if values[i] == nil {
			continue
		}
		if v, ok := values[i].(float64); ok {
			return v, true
		}
	}
	return 0, false
}

func lastMACD(values []any) (indicators.MACDValue, bool) {
	for i := len(values) - 1; i >= 0; i-- {
		if values[i] == nil {
			continue
		}
		if v, ok := values[i].(indicators.MACDValue); ok {
			return v, true
		}
	}
	return indicators.MACDValue{}, false
}

func createSampleData() []core.Candle {
	candles := make([]core.Candle, 50)
	baseTime := time.Now().Add(-50 * 24 * time.Hour)
	price := 100.0

	for i := range candles {
		// Simple price simulation
		change := (float64(i%5) - 2.0) * 0.5
		price += change

		candles[i] = core.Candle{
			Time:   baseTime.Add(time.Duration(i) * 24 * time.Hour),
			Open:   price - 0.5,
			High:   price + 1.0,
			Low:    price - 1.0,
			Close:  price,
			Volume: 1000,
		}
	}

	return candles
}
