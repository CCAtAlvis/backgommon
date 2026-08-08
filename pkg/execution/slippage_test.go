package execution

import (
	"testing"

	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

func TestApplySlippagePercentLongEntry(t *testing.T) {
	exec := portfolio.ExecutionSettings{
		SlippageMode:        "PercentOfPrice",
		PercentSlippageRate: 0.01,
	}
	ord := portfolio.NewOrder("X", portfolio.Long, portfolio.Entry, 1, 1)
	got := ApplySlippage(100, ord, exec)
	if got != 101 {
		t.Fatalf("got %.2f want 101", got)
	}
}

func TestApplySlippageLongExit(t *testing.T) {
	exec := portfolio.ExecutionSettings{
		SlippageMode:        "FixedPoints",
		FixedSlippageAmount: 0.5,
	}
	ord := portfolio.NewOrder("X", portfolio.Long, portfolio.Exit, 1, 1)
	got := ApplySlippage(100, ord, exec)
	if got != 99.5 {
		t.Fatalf("got %.2f want 99.5", got)
	}
}
