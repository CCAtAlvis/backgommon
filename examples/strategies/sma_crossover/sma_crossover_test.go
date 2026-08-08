package main

import (
	"testing"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/indicators"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

type stubPortfolio struct {
	positions map[string]*portfolio.Position
}

func (s *stubPortfolio) ProcessOrder(portfolio.Order) error { return nil }
func (s *stubPortfolio) UpdatePositions(map[string]float64) {}
func (s *stubPortfolio) Value() float64                     { return 0 }
func (s *stubPortfolio) Cash() float64                      { return 0 }
func (s *stubPortfolio) Positions() map[string]*portfolio.Position {
	return s.positions
}
func (s *stubPortfolio) InitialCapital() float64 { return 100000 }
func (s *stubPortfolio) Stats() portfolio.PortfolioStats {
	return portfolio.PortfolioStats{}
}
func (s *stubPortfolio) ClosedPositions() []*portfolio.Position { return nil }
func (s *stubPortfolio) EstimateEntryCash(portfolio.Order) float64 { return 0 }

func TestSMACrossoverStrategy_GoldenCross(t *testing.T) {
	cfg := SMAStrategyConfig{
		Symbol:      "TEST",
		ShortWindow: 2,
		LongWindow:  3,
		Quantity:    10,
		CustomField: "demo",
	}
	strat, err := NewSMACrossoverStrategy(cfg)
	if err != nil {
		t.Fatalf("new strategy: %v", err)
	}

	shortSMA := indicators.NewSMA(2)
	longSMA := indicators.NewSMA(3)

	// Flat then jump: golden cross on bar index 3 (closes 10,10,10,15)
	series := []core.Candle{
		{Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 10},
		{Time: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Close: 10},
		{Time: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Close: 10},
		{Time: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), Close: 15},
	}

	applyIndicators := func(candles []core.Candle) []core.Candle {
		vs := shortSMA.Calculate(candles)
		vl := longSMA.Calculate(candles)
		for i, c := range candles {
			if vs[i] != nil {
				c.SetIndicator(shortSMA.Name(), vs[i])
			}
			if vl[i] != nil {
				c.SetIndicator(longSMA.Name(), vl[i])
			}
			candles[i] = c
		}
		return candles
	}

	series = applyIndicators(series)

	stub := &stubPortfolio{positions: map[string]*portfolio.Position{}}
	strat.SetPortfolio(stub)

	for i := 0; i < 3; i++ {
		_ = strat.OnTick(map[string]core.Candle{"TEST": series[i]})
	}

	orders := strat.OnTick(map[string]core.Candle{"TEST": series[3]})
	if len(orders) != 1 {
		t.Fatalf("expected golden cross entry, got %d orders", len(orders))
	}
	if orders[0].Type != portfolio.Entry {
		t.Fatalf("expected entry order, got %v", orders[0].Type)
	}
}

func TestSMACrossoverStrategy_InvalidConfig(t *testing.T) {
	_, err := NewSMACrossoverStrategy(SMAStrategyConfig{
		Symbol:      "X",
		ShortWindow: 50,
		LongWindow:  50,
		Quantity:    1,
	})
	if err == nil {
		t.Fatal("expected error when short >= long window")
	}
}
