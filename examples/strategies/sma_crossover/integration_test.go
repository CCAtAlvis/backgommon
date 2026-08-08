package main

import (
	"testing"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/data"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
	"github.com/CCAtAlvis/backgommon/pkg/risk"
	"github.com/CCAtAlvis/backgommon/pkg/runner"
)

// TestSMACrossoverBacktest runs the example strategy through the real runner.
// Lives here (not pkg/runner) because the strategy is example code, not framework code.
func TestSMACrossoverBacktest(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := []core.Candle{
		{Time: base, Close: 10},
		{Time: base.AddDate(0, 0, 1), Close: 10},
		{Time: base.AddDate(0, 0, 2), Close: 10},
		{Time: base.AddDate(0, 0, 3), Close: 15},
		{Time: base.AddDate(0, 0, 4), Close: 20},
		{Time: base.AddDate(0, 0, 5), Close: 8},
		{Time: base.AddDate(0, 0, 6), Close: 7},
	}
	for i := range candles {
		candles[i].Open = candles[i].Close - 0.5
		candles[i].High = candles[i].Close + 1
		candles[i].Low = candles[i].Close - 1
	}

	table, err := data.LoadSingleSymbolTable("TEST", candles)
	if err != nil {
		t.Fatal(err)
	}

	strat, err := NewSMACrossoverStrategy(SMAStrategyConfig{
		Symbol:      "TEST",
		ShortWindow: 2,
		LongWindow:  3,
		Quantity:    1,
	})
	if err != nil {
		t.Fatal(err)
	}

	r := runner.New(
		strat,
		runner.WithPortfolio(portfolio.New(&portfolio.Settings{InitialCapital: 10000, EnableShorts: false})),
		runner.WithRiskManager(risk.New(&risk.Settings{MaxLeverage: 1, MaxPositionAllocationRate: 1})),
		runner.WithData(table),
		runner.WithIndicators(strat.Indicators()),
	)

	if err := r.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	if r.Results.TotalTrades == 0 {
		t.Fatalf("expected at least one closed trade")
	}
}
