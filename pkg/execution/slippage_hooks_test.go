package execution_test

import (
	"testing"

	"github.com/CCAtAlvis/backgommon/pkg/execution"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

func TestFuncSlippageModelOverride(t *testing.T) {
	model := &execution.FuncSlippageModel{
		AdjustFn: func(ctx execution.SlippageContext) float64 {
			return 42
		},
	}
	ord := portfolio.NewOrder("X", portfolio.Long, portfolio.Entry, 1, 1)
	got := model.AdjustPrice(execution.SlippageContext{BasePrice: 100, Order: ord})
	if got != 42 {
		t.Fatalf("got %.2f", got)
	}
}
