package runner_test

import (
	"testing"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/indicators"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
	"github.com/CCAtAlvis/backgommon/pkg/risk"
	"github.com/CCAtAlvis/backgommon/pkg/runner"
	"github.com/CCAtAlvis/backgommon/pkg/strategy"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

type smokeStrategy struct {
	strategy.BaseStrategy
	symbol string
	sma20  *indicators.SMA
}

func (s *smokeStrategy) OnTick(data map[string]core.Candle) []portfolio.Order {
	candle, ok := data[s.symbol]
	if !ok {
		return nil
	}

	sma, err := candle.GetIndicator(s.sma20.Name())
	if err != nil {
		return nil
	}
	smaVal, ok := sma.(float64)
	if !ok {
		return nil
	}

	_, inPosition := s.Portfolio.Positions()[s.symbol]
	if !inPosition && smaVal > candle.Close*0.9 {
		return []portfolio.Order{
			portfolio.NewOrder(s.symbol, portfolio.Long, portfolio.Entry, 1, 1.0),
		}
	}
	return nil
}

func TestBacktestSmoke(t *testing.T) {
	symbol := "TEST"
	sma20 := indicators.NewSMA(5)
	strat := &smokeStrategy{symbol: symbol, sma20: sma20}

	table := types.NewTimeseriesTable[core.Candle]([]string{symbol})
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	price := 100.0
	for i := 0; i < 20; i++ {
		price += 1.0
		candle := core.Candle{
			Time:  baseTime.AddDate(0, 0, i),
			Open:  price - 0.5,
			High:  price + 1.0,
			Low:   price - 1.0,
			Close: price,
		}
		if err := table.AddRow(candle.Time, map[string]core.Candle{symbol: candle}); err != nil {
			t.Fatalf("add row: %v", err)
		}
	}

	if err := table.ApplyIndicators([]interfaces.Indicator{sma20}); err != nil {
		t.Fatalf("apply indicators: %v", err)
	}

	portfolioSettings := &portfolio.Settings{
		InitialCapital: 100000,
		EnableShorts:   false,
	}
	riskSettings := &risk.Settings{
		MaxLeverage:               1.0,
		MaxPositionAllocationRate: 1.0,
	}

	r := runner.New(
		strat,
		runner.WithPortfolio(portfolio.New(portfolioSettings)),
		runner.WithRiskManager(risk.New(riskSettings)),
		runner.WithData(table),
	)

	if err := r.Start(); err != nil {
		t.Fatalf("backtest failed: %v", err)
	}

	if r.Portfolio.Value() <= 0 {
		t.Fatalf("expected positive portfolio value, got %.2f", r.Portfolio.Value())
	}
	if len(r.EquityCurve) != 20 {
		t.Fatalf("expected 20 equity curve points, got %d", len(r.EquityCurve))
	}
	if r.Results == nil {
		t.Fatal("expected results to be populated")
	}
	if r.Results.InitialCapital != 100000 {
		t.Fatalf("initial capital: got %.2f", r.Results.InitialCapital)
	}
}

// oversizedEntryStrategy always tries to buy more than cash allows.
type oversizedEntryStrategy struct {
	strategy.BaseStrategy
	symbol string
}

func (s *oversizedEntryStrategy) OnTick(data map[string]core.Candle) []portfolio.Order {
	c, ok := data[s.symbol]
	if !ok || c.Close <= 0 {
		return nil
	}
	return []portfolio.Order{
		portfolio.NewOrder(s.symbol, portfolio.Long, portfolio.Entry, 1000, 1.0),
	}
}

func synthTable(symbol string, n int) *types.TimeseriesTable[core.Candle] {
	table := types.NewTimeseriesTable[core.Candle]([]string{symbol})
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		price := 100.0 + float64(i)
		candle := core.Candle{
			Time:  baseTime.AddDate(0, 0, i),
			Open:  price,
			High:  price + 1,
			Low:   price - 1,
			Close: price,
		}
		_ = table.AddRow(candle.Time, map[string]core.Candle{symbol: candle})
	}
	return table
}

func TestLenientModeSkipsFailedOrders(t *testing.T) {
	symbol := "TEST"
	strat := &oversizedEntryStrategy{symbol: symbol}
	table := synthTable(symbol, 5)

	r := runner.New(
		strat,
		runner.WithPortfolio(portfolio.New(&portfolio.Settings{
			InitialCapital:            1000,
			DefaultLeverage:           1,
			AllowNegativeCashFromFees: false,
		})),
		runner.WithRiskManager(risk.New(&risk.Settings{MaxLeverage: 1})),
		runner.WithData(table),
		// default: lenient
	)
	if err := r.Start(); err != nil {
		t.Fatalf("lenient backtest should not abort: %v", err)
	}
	if r.SkippedOrders() == 0 {
		t.Fatal("expected skipped orders in lenient mode")
	}
	if r.Results == nil || r.Results.SkippedOrders == 0 {
		t.Fatal("expected Results.SkippedOrders > 0")
	}
	if len(r.Portfolio.Positions()) != 0 {
		t.Fatal("oversized entries should not have filled")
	}
}

func TestStrictModeAbortsOnFailedOrder(t *testing.T) {
	symbol := "TEST"
	strat := &oversizedEntryStrategy{symbol: symbol}
	table := synthTable(symbol, 5)

	r := runner.New(
		strat,
		runner.WithPortfolio(portfolio.New(&portfolio.Settings{
			InitialCapital:            1000,
			DefaultLeverage:           1,
			AllowNegativeCashFromFees: false,
		})),
		runner.WithRiskManager(risk.New(&risk.Settings{MaxLeverage: 1})),
		runner.WithData(table),
		runner.WithStrictOrderProcessing(true),
	)
	if err := r.Start(); err == nil {
		t.Fatal("strict backtest should abort on insufficient cash")
	}
}
