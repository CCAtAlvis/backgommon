package indicators

import (
	"testing"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
)

func TestEMA_Calculate(t *testing.T) {
	candles := []core.Candle{
		{Time: time.Now(), Close: 10},
		{Time: time.Now().Add(1 * time.Minute), Close: 20},
		{Time: time.Now().Add(2 * time.Minute), Close: 30},
		{Time: time.Now().Add(3 * time.Minute), Close: 40},
		{Time: time.Now().Add(4 * time.Minute), Close: 50},
	}

	ema := NewEMA(3)
	values := ema.Calculate(candles)

	if len(values) != len(candles) {
		t.Fatalf("Expected %d values, got %d", len(candles), len(values))
	}

	// First two should be nil (insufficient data)
	for i := 0; i < 2; i++ {
		if values[i] != nil {
			t.Errorf("Expected nil for insufficient data at index %d, got %v", i, values[i])
		}
	}

	// Third value should be the initial SMA
	expected := float64((10 + 20 + 30)) / 3
	if v, ok := values[2].(float64); !ok || v != expected {
		t.Errorf("Expected EMA value %v at index 2, but got %v", expected, values[2])
	}
}

func TestEMA_InsufficientData(t *testing.T) {
	candles := []core.Candle{
		{Time: time.Now(), Close: 10},
	}

	ema := NewEMA(5)
	values := ema.Calculate(candles)

	if len(values) != len(candles) {
		t.Fatalf("Expected %d values, got %d", len(candles), len(values))
	}

	if values[0] != nil {
		t.Errorf("Expected nil for insufficient data, but got %v", values[0])
	}
}
