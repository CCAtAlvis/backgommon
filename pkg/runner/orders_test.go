package runner

import (
	"testing"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/execution"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

func makeCandle(open, high, low, close float64) core.Candle {
	return core.Candle{Open: open, High: high, Low: low, Close: close}
}

func TestApplyFillPricesDefaultMode(t *testing.T) {
	r := &Runner{}
	orders := []portfolio.Order{
		portfolio.NewOrder("AAPL", portfolio.Long, portfolio.Entry, 10, 1.0),
	}
	current := map[string]core.Candle{
		"AAPL": makeCandle(100, 105, 95, 102),
	}

	if err := r.applyFillPrices(orders, current, nil); err != nil {
		t.Fatalf("applyFillPrices: %v", err)
	}

	if orders[0].Price != 102 {
		t.Errorf("Price = %f, want 102 (close)", orders[0].Price)
	}
}

func TestApplyFillPricesExplicitPrice(t *testing.T) {
	r := &Runner{}
	orders := []portfolio.Order{
		portfolio.NewOrder("AAPL", portfolio.Long, portfolio.Entry, 10, 1.0),
	}
	orders[0].Price = 99.0

	current := map[string]core.Candle{
		"AAPL": makeCandle(100, 105, 95, 102),
	}

	if err := r.applyFillPrices(orders, current, nil); err != nil {
		t.Fatalf("applyFillPrices: %v", err)
	}

	if orders[0].Price != 99.0 {
		t.Errorf("explicit price should be preserved, got %f", orders[0].Price)
	}
}

func TestApplyFillPricesWithSlippage(t *testing.T) {
	r := &Runner{
		Portfolio: portfolio.New(&portfolio.Settings{
			InitialCapital: 100000,
			Execution: portfolio.ExecutionSettings{
				SlippageMode:        "PercentOfPrice",
				PercentSlippageRate: 0.01,
			},
		}),
	}

	orders := []portfolio.Order{
		portfolio.NewOrder("AAPL", portfolio.Long, portfolio.Entry, 10, 1.0),
	}
	current := map[string]core.Candle{
		"AAPL": makeCandle(100, 105, 95, 100),
	}

	if err := r.applyFillPrices(orders, current, nil); err != nil {
		t.Fatalf("applyFillPrices: %v", err)
	}

	want := 100.0 * (1 + 0.01) // long entry slips up
	if orders[0].Price != want {
		t.Errorf("Price = %f, want %f (with 1%% slippage)", orders[0].Price, want)
	}
}

func TestApplyFillPricesNextBarOpen(t *testing.T) {
	r := &Runner{
		fillPricer: execution.NewStandardFillPricer(execution.FillNextBarOpen),
	}

	orders := []portfolio.Order{
		portfolio.NewOrder("AAPL", portfolio.Long, portfolio.Entry, 10, 1.0),
	}
	current := map[string]core.Candle{
		"AAPL": makeCandle(100, 105, 95, 102),
	}
	next := map[string]core.Candle{
		"AAPL": makeCandle(103, 108, 99, 106),
	}

	if err := r.applyFillPrices(orders, current, next); err != nil {
		t.Fatalf("applyFillPrices: %v", err)
	}

	if orders[0].Price != 103 {
		t.Errorf("Price = %f, want 103 (next bar open)", orders[0].Price)
	}

	orders2 := []portfolio.Order{
		portfolio.NewOrder("AAPL", portfolio.Long, portfolio.Entry, 10, 1.0),
	}
	err := r.applyFillPrices(orders2, current, nil)
	if err == nil {
		t.Error("expected error when next bar is nil for FillNextBarOpen")
	}
}

func TestApplyFillPricesCustomFillPricer(t *testing.T) {
	r := &Runner{
		fillPricer: execution.NewFuncFillPricer(
			func(ctx interfaces.FillContext) (float64, error) {
				return (ctx.CurrentBar.High + ctx.CurrentBar.Low) / 2, nil
			},
		),
	}

	orders := []portfolio.Order{
		portfolio.NewOrder("AAPL", portfolio.Long, portfolio.Entry, 10, 1.0),
	}
	current := map[string]core.Candle{
		"AAPL": makeCandle(100, 110, 90, 102),
	}

	if err := r.applyFillPrices(orders, current, nil); err != nil {
		t.Fatalf("applyFillPrices: %v", err)
	}

	want := (110.0 + 90.0) / 2
	if orders[0].Price != want {
		t.Errorf("Price = %f, want %f (mid price from custom pricer)", orders[0].Price, want)
	}
}
