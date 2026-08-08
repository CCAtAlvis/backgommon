package core_test

import (
	"testing"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
)

func TestCandleCloneIsolatesIndicators(t *testing.T) {
	c := core.Candle{
		Time:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Open:   10,
		High:   12,
		Low:    9,
		Close:  11,
		Volume: 1000,
	}
	c.SetIndicator("SMA_3", 10.5)

	clone := c.Clone()
	clone.SetIndicator("SMA_3", 99.0)
	clone.SetIndicator("RSI_14", 55.0)
	clone.Close = 42

	got, err := c.GetIndicator("SMA_3")
	if err != nil {
		t.Fatalf("original missing SMA_3: %v", err)
	}
	if got.(float64) != 10.5 {
		t.Fatalf("original SMA_3 mutated: got %v want 10.5", got)
	}
	if c.HasIndicator("RSI_14") {
		t.Fatal("original unexpectedly gained RSI_14")
	}
	if c.Close != 11 {
		t.Fatalf("original Close mutated: got %v", c.Close)
	}

	if clone.Close != 42 {
		t.Fatalf("clone Close not updated: got %v", clone.Close)
	}
}

func TestCandleCloneNilIndicators(t *testing.T) {
	c := core.Candle{Close: 1}
	clone := c.Clone()
	if clone.GetAllIndicators() != nil {
		t.Fatal("expected nil indicators map on clone of empty candle")
	}
	clone.SetIndicator("x", 1.0)
	if c.HasIndicator("x") {
		t.Fatal("setting indicator on clone mutated original")
	}
}
