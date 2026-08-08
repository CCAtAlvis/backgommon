package portfolio_test

import (
	"testing"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

func TestCustomCostCalculatorOverride(t *testing.T) {
	custom := &portfolio.FuncCostCalculator{
		BrokerageFn: func(ctx portfolio.TradeCostContext) float64 {
			return 99
		},
	}

	p := portfolio.New(
		&portfolio.Settings{InitialCapital: 10_000, DefaultLeverage: 1},
		portfolio.WithCostCalculator(custom),
	)

	entry := portfolio.NewOrder("X", portfolio.Long, portfolio.Entry, 10, 1)
	entry.Price = 100
	if err := p.ProcessOrder(entry); err != nil {
		t.Fatalf("entry: %v", err)
	}

	// margin 1000 + custom brokerage 99
	wantCash := 10_000.0 - 1099.0
	if p.Cash() != wantCash {
		t.Fatalf("cash: got %.2f want %.2f", p.Cash(), wantCash)
	}
}

func TestCustomCashFlowOverride(t *testing.T) {
	p := portfolio.New(
		&portfolio.Settings{InitialCapital: 5_000, DefaultLeverage: 1},
		portfolio.WithCashFlowCalculator(&portfolio.FuncCashFlowCalculator{
			EntryCashOutflowFn: func(ctx portfolio.EntryCashContext) float64 {
				return 123
			},
		}),
	)

	entry := portfolio.NewOrder("X", portfolio.Long, portfolio.Entry, 1, 1)
	entry.Price = 1000
	if err := p.ProcessOrder(entry); err != nil {
		t.Fatalf("entry: %v", err)
	}
	if p.Cash() != 5000-123 {
		t.Fatalf("unexpected cash %.2f", p.Cash())
	}
}

func TestCustomPeriodicEventsOverride(t *testing.T) {
	p := portfolio.New(
		&portfolio.Settings{InitialCapital: 1_000},
		portfolio.WithPeriodicEventsModel(&portfolio.FuncPeriodicEventsModel{
			ApplyFn: func(ctx portfolio.PeriodicContext) portfolio.PeriodicResult {
				return portfolio.PeriodicResult{CashDelta: 250}
			},
		}),
	)

	p.ProcessPeriodicEvents(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	if p.Cash() != 1250 {
		t.Fatalf("cash: got %.2f", p.Cash())
	}
}
