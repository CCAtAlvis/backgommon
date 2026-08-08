package data

import (
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
)

// SyntheticTrendConfig controls generated OHLCV series for examples and tests.
type SyntheticTrendConfig struct {
	BarCount  int
	StartTime time.Time
	StartPrice float64
	// DailyDrift is added to close each bar (can be negative).
	DailyDrift float64
	// CycleAmplitude adds a sine-like wiggle via i%7 for demo crossovers.
	CycleAmplitude float64
}

// DefaultSyntheticTrendConfig returns settings suitable for SMA(50/200) warm-up.
func DefaultSyntheticTrendConfig() SyntheticTrendConfig {
	return SyntheticTrendConfig{
		BarCount:       300,
		StartTime:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		StartPrice:     100,
		DailyDrift:     0.05,
		CycleAmplitude: 0.7,
	}
}

// GenerateSyntheticTrend builds OHLCV candles with a mild trend and cycle.
func GenerateSyntheticTrend(cfg SyntheticTrendConfig) []core.Candle {
	if cfg.BarCount <= 0 {
		cfg.BarCount = 300
	}
	if cfg.StartTime.IsZero() {
		cfg.StartTime = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if cfg.StartPrice == 0 {
		cfg.StartPrice = 100
	}

	candles := make([]core.Candle, cfg.BarCount)
	price := cfg.StartPrice
	for i := range candles {
		change := (float64(i%7) - 3.0) * cfg.CycleAmplitude
		price += cfg.DailyDrift + change
		candles[i] = core.Candle{
			Time:   cfg.StartTime.AddDate(0, 0, i),
			Open:   price - 0.5,
			High:   price + 1.0,
			Low:    price - 1.0,
			Close:  price,
			Volume: 1000,
		}
	}
	return candles
}
