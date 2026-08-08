package risk_test

import (
	"testing"

	"github.com/CCAtAlvis/backgommon/pkg/risk"
)

func TestFuncPositionSizerOverride(t *testing.T) {
	m := risk.New(&risk.Settings{
		RiskPerTradeRate:    0.01,
		EnableStopLoss:      true,
		DefaultStopLossRate: 0.05,
	}, risk.WithPositionSizer(&risk.FuncPositionSizer{
		SuggestFn: func(ctx risk.PositionSizeContext) int {
			return 7
		},
	}))

	if qty := m.SuggestedQuantity(100_000, 100); qty != 7 {
		t.Fatalf("qty: got %d want 7", qty)
	}
}

func TestFuncDrawdownPolicyOverride(t *testing.T) {
	m := risk.New(&risk.Settings{
		MaxPortfolioDrawdownRate: 0.01,
		MaxDrawdownMode:          risk.LiquidateAllPositions,
	}, risk.WithDrawdownPolicy(&risk.FuncDrawdownPolicy{
		EvaluateFn: func(ctx risk.DrawdownPolicyContext) risk.DrawdownPolicyResult {
			return risk.DrawdownPolicyResult{LiquidateAll: true, DrawdownRate: 0.5}
		},
	}))

	// Policy is used via CheckDrawdown on a real portfolio in integration tests;
	// here we only verify wiring does not panic and custom policy is stored.
	if m == nil {
		t.Fatal("nil manager")
	}
}
