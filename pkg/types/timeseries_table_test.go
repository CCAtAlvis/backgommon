package types_test

import (
	"testing"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/indicators"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

func TestCloneCandleTableIsolatesIndicators(t *testing.T) {
	table := types.NewTimeseriesTable[core.Candle]([]string{"A"})
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 5; i++ {
		ts := baseTime.AddDate(0, 0, i)
		c := core.Candle{Time: ts, Close: 100 + float64(i)}
		c.SetIndicator("seed", float64(i))
		if err := table.AddRow(ts, map[string]core.Candle{"A": c}); err != nil {
			t.Fatalf("add row: %v", err)
		}
	}

	clone := types.CloneCandleTable(table)
	sma := indicators.NewSMA(3)
	if err := clone.ApplyIndicators([]interfaces.Indicator{sma}); err != nil {
		t.Fatalf("apply on clone: %v", err)
	}

	last := baseTime.AddDate(0, 0, 4)
	orig, ok := table.GetValue(last, "A")
	if !ok {
		t.Fatal("missing original candle")
	}
	if orig.HasIndicator(sma.Name()) {
		t.Fatal("applying indicators on clone mutated the original table")
	}

	cloned, ok := clone.GetValue(last, "A")
	if !ok {
		t.Fatal("missing cloned candle")
	}
	if !cloned.HasIndicator(sma.Name()) {
		t.Fatal("expected SMA on cloned candle")
	}
	seed, err := cloned.GetIndicator("seed")
	if err != nil || seed.(float64) != 4 {
		t.Fatalf("seed indicator not preserved on clone: %v %v", seed, err)
	}
}

func TestFillRowReusesDestination(t *testing.T) {
	table := types.NewTimeseriesTable[core.Candle]([]string{"A", "B"})
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	_ = table.AddRow(ts, map[string]core.Candle{
		"A": {Time: ts, Close: 10},
		"B": {Time: ts, Close: 20},
	})

	dst := make(map[string]core.Candle, 2)
	if !table.FillRow(ts, dst) {
		t.Fatal("FillRow failed")
	}
	if dst["A"].Close != 10 || dst["B"].Close != 20 {
		t.Fatalf("unexpected values: %+v", dst)
	}
	// Second fill into same map must not allocate a new map header (same pointer).
	ptr := &dst
	ts2 := ts.AddDate(0, 0, 1)
	_ = table.AddRow(ts2, map[string]core.Candle{
		"A": {Time: ts2, Close: 11},
		"B": {Time: ts2, Close: 21},
	})
	if !table.FillRow(ts2, dst) {
		t.Fatal("FillRow 2 failed")
	}
	if &dst != ptr {
		t.Fatal("destination map replaced")
	}
	if dst["A"].Close != 11 || dst["B"].Close != 21 {
		t.Fatalf("unexpected second values: %+v", dst)
	}
}


func TestApplyIndicatorsAllColumns(t *testing.T) {
	table := types.NewTimeseriesTable[core.Candle]([]string{"A", "B"})
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 10; i++ {
		ts := baseTime.AddDate(0, 0, i)
		price := 100.0 + float64(i)
		row := map[string]core.Candle{
			"A": {Time: ts, Close: price},
			"B": {Time: ts, Close: price * 2},
		}
		if err := table.AddRow(ts, row); err != nil {
			t.Fatalf("add row: %v", err)
		}
	}

	sma := indicators.NewSMA(3)
	if err := table.ApplyIndicators([]interfaces.Indicator{sma}); err != nil {
		t.Fatalf("apply indicators: %v", err)
	}

	lastTime := baseTime.AddDate(0, 0, 9)
	for _, col := range []string{"A", "B"} {
		row, ok := table.GetRow(lastTime)
		if !ok {
			t.Fatalf("missing row for column %s", col)
		}
		candle := row[col]
		if _, err := candle.GetIndicator(sma.Name()); err != nil {
			t.Fatalf("indicator not applied to column %s: %v", col, err)
		}
	}
}
