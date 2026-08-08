package execution

import (
	"strings"

	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

// SlippageMode names a built-in slippage simulation strategy.
type SlippageMode string

const (
	// SlippageNone applies no slippage adjustment.
	SlippageNone SlippageMode = "None"

	// SlippageFixedPoints adds/subtracts a fixed point amount from the fill price.
	SlippageFixedPoints SlippageMode = "FixedPoints"

	// SlippagePercentOfPrice adjusts the fill price by a percentage.
	SlippagePercentOfPrice SlippageMode = "PercentOfPrice"
)

// SlippageContext supplies inputs for adverse price adjustment after fill pricing.
// BasePrice is the raw fill price produced by the [interfaces.FillPricer]; the
// slippage model shifts it adversely (higher for buys, lower for sells) to
// simulate real-world market impact.
type SlippageContext struct {
	BasePrice float64
	Order     portfolio.Order
	Execution portfolio.ExecutionSettings
}

// SlippageModel adjusts a base fill price for slippage.
// Inject via runner.WithSlippageModel; default is StandardSlippageModel.
type SlippageModel interface {
	AdjustPrice(ctx SlippageContext) float64
}

// StandardSlippageModel implements SlippageModel from ExecutionSettings.
type StandardSlippageModel struct{}

// AdjustPrice applies adverse slippage to the base fill price according to
// the SlippageMode in [portfolio.ExecutionSettings]. Supported mode strings
// (matched case-insensitively):
//
//   - "None" (or empty) — no adjustment; returns the base price as-is.
//   - "FixedPoints"     — adds/subtracts Execution.FixedSlippageAmount in
//     absolute price units. Positive amounts widen the spread adversely.
//   - "PercentOfPrice"  — adjusts by Execution.PercentSlippageRate as a
//     fraction of the base price (e.g. 0.001 = 0.1%).
//
// Any unrecognized mode string silently returns the base price unchanged.
// Direction is determined automatically: buys slip upward, sells slip
// downward.
func (StandardSlippageModel) AdjustPrice(ctx SlippageContext) float64 {
	price := ctx.BasePrice
	exec := ctx.Execution
	mode := strings.TrimSpace(exec.SlippageMode)

	if mode == "" || strings.EqualFold(mode, string(SlippageNone)) {
		return price
	}

	adverse := isAdverseBuy(ctx.Order)

	if strings.EqualFold(mode, string(SlippageFixedPoints)) {
		if exec.FixedSlippageAmount <= 0 {
			return price
		}
		if adverse {
			return price + exec.FixedSlippageAmount
		}
		return price - exec.FixedSlippageAmount
	}

	if strings.EqualFold(mode, string(SlippagePercentOfPrice)) {
		if exec.PercentSlippageRate <= 0 {
			return price
		}
		if adverse {
			return price * (1 + exec.PercentSlippageRate)
		}
		return price * (1 - exec.PercentSlippageRate)
	}

	return price
}

// FuncSlippageModel wraps an optional override; unset delegates to fallback.
type FuncSlippageModel struct {
	Fallback *StandardSlippageModel
	AdjustFn func(SlippageContext) float64
}

func (f *FuncSlippageModel) fallback() SlippageModel {
	if f.Fallback != nil {
		return f.Fallback
	}
	return StandardSlippageModel{}
}

// AdjustPrice delegates to AdjustFn if set; otherwise falls back to the
// Fallback model (or a zero-value [StandardSlippageModel] if Fallback is nil).
// This allows partial overrides: set AdjustFn for custom logic, or leave it
// nil to use the standard settings-driven behavior.
func (f *FuncSlippageModel) AdjustPrice(ctx SlippageContext) float64 {
	if f.AdjustFn != nil {
		return f.AdjustFn(ctx)
	}
	return f.fallback().AdjustPrice(ctx)
}

// ApplySlippage is deprecated; use StandardSlippageModel or inject SlippageModel via runner.WithSlippageModel.
func ApplySlippage(price float64, ord portfolio.Order, exec portfolio.ExecutionSettings) float64 {
	return StandardSlippageModel{}.AdjustPrice(SlippageContext{
		BasePrice: price,
		Order:     ord,
		Execution: exec,
	})
}

func isAdverseBuy(ord portfolio.Order) bool {
	switch ord.Type {
	case portfolio.Entry:
		return ord.Side == portfolio.Long
	case portfolio.Exit:
		return ord.Side == portfolio.Short
	default:
		return true
	}
}
