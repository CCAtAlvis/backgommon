package runner

import (
	"testing"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/types"
)

func TestMaxDrawdownFromEquityCurve(t *testing.T) {
	curve := []types.AccountValue{
		{Time: time.Now(), Value: 100},
		{Time: time.Now(), Value: 120},
		{Time: time.Now(), Value: 90},
	}
	dd := maxDrawdownFromEquityCurve(curve)
	// peak 120 -> trough 90 => 25%
	if dd < 0.24 || dd > 0.26 {
		t.Fatalf("expected ~0.25 drawdown, got %.4f", dd)
	}
}
