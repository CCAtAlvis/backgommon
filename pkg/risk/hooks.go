package risk

import (
	"math"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
)

// PositionSizeContext supplies inputs for position sizing.
type PositionSizeContext struct {
	Equity     float64
	EntryPrice float64
	Settings   *Settings
}

// PositionSizer suggests entry quantity from risk settings.
// Inject via risk.WithPositionSizer; default is StandardPositionSizer.
type PositionSizer interface {
	SuggestQuantity(ctx PositionSizeContext) int
}

// StandardPositionSizer sizes from RiskPerTradeRate and DefaultStopLossRate.
type StandardPositionSizer struct{}

// SuggestQuantity computes a position size from the risk-per-trade budget:
//
//	quantity = floor(equity × RiskPerTradeRate / (entryPrice × DefaultStopLossRate))
//
// Both EnableStopLoss and a positive DefaultStopLossRate must be configured;
// otherwise the method returns 0 (cannot size without a defined risk distance).
// A zero or negative equity, entry price, or per-share risk also yields 0.
func (StandardPositionSizer) SuggestQuantity(ctx PositionSizeContext) int {
	s := ctx.Settings
	if s == nil || s.RiskPerTradeRate <= 0 || ctx.EntryPrice <= 0 {
		return 0
	}
	if !s.EnableStopLoss || s.DefaultStopLossRate <= 0 {
		return 0
	}

	riskBudget := ctx.Equity * s.RiskPerTradeRate
	perShareRisk := ctx.EntryPrice * s.DefaultStopLossRate
	if perShareRisk <= 0 {
		return 0
	}

	qty := int(math.Floor(riskBudget / perShareRisk))
	if qty < 0 {
		return 0
	}
	return qty
}

// FuncPositionSizer wraps an optional override; unset delegates to fallback.
type FuncPositionSizer struct {
	Fallback *StandardPositionSizer
	SuggestFn func(PositionSizeContext) int
}

func (f *FuncPositionSizer) fallback() PositionSizer {
	if f.Fallback != nil {
		return f.Fallback
	}
	return StandardPositionSizer{}
}

// SuggestQuantity delegates to SuggestFn if set; otherwise falls back to the
// Fallback sizer (or a zero-value [StandardPositionSizer] if Fallback is nil).
func (f *FuncPositionSizer) SuggestQuantity(ctx PositionSizeContext) int {
	if f.SuggestFn != nil {
		return f.SuggestFn(ctx)
	}
	return f.fallback().SuggestQuantity(ctx)
}

// DrawdownPolicyContext supplies inputs for drawdown evaluation.
type DrawdownPolicyContext struct {
	Equity     float64
	PeakEquity float64
	Now        time.Time
	LockUntil  time.Time
	Settings   *Settings
	Logger     Logger
}

// DrawdownPolicyResult is the outcome of drawdown evaluation.
type DrawdownPolicyResult struct {
	DrawdownRate    float64
	PeakEquity      float64
	BlockNewEntries bool
	LiquidateAll    bool
	Breached        bool
	LockUntil       time.Time
}

// DrawdownPolicy evaluates portfolio drawdown and returns required actions.
// Inject via risk.WithDrawdownPolicy; default is StandardDrawdownPolicy.
type DrawdownPolicy interface {
	Evaluate(ctx DrawdownPolicyContext) DrawdownPolicyResult
}

// StandardDrawdownPolicy implements drawdown limits from risk.Settings.
type StandardDrawdownPolicy struct{}

// Evaluate computes the current drawdown rate from peak equity and applies the
// action specified by Settings.MaxDrawdownMode when the threshold is breached.
// Possible outcomes include logging a warning (AlertOnly), blocking new entries
// (StopNewTrades with an optional lock duration), or requesting full
// liquidation (LiquidateAllPositions).
//
// If the current time is still within a previously set lock window, new
// entries remain blocked regardless of whether drawdown has recovered.
//
// Side-effect: this method prints diagnostic messages to stdout when a
// drawdown mode triggers. This is a known design issue and may be replaced
// with a structured logging hook in a future release.
func (StandardDrawdownPolicy) Evaluate(ctx DrawdownPolicyContext) DrawdownPolicyResult {
	peak := ctx.PeakEquity
	if peak <= 0 || ctx.Equity > peak {
		peak = ctx.Equity
	}

	result := DrawdownPolicyResult{
		PeakEquity: peak,
		LockUntil:  ctx.LockUntil,
	}

	s := ctx.Settings
	if s == nil || s.MaxPortfolioDrawdownRate <= 0 || peak <= 0 {
		return result
	}

	drawdown := (peak - ctx.Equity) / peak
	result.DrawdownRate = drawdown

	if ctx.Now.Before(ctx.LockUntil) {
		result.BlockNewEntries = true
		return result
	}

	if drawdown < s.MaxPortfolioDrawdownRate {
		return result
	}

	result.Breached = true

	logf := func(format string, args ...interface{}) {
		if ctx.Logger != nil {
			ctx.Logger.Printf(format, args...)
		}
	}

	switch s.MaxDrawdownMode {
	case AlertOnly:
		logf("WARN: portfolio drawdown %.2f%% exceeds limit %.2f%%\n",
			drawdown*100, s.MaxPortfolioDrawdownRate*100)
	case StopNewTrades:
		result.BlockNewEntries = true
		if s.DrawdownLockDuration > 0 {
			result.LockUntil = ctx.Now.Add(s.DrawdownLockDuration)
		}
		logf("WARN: drawdown %.2f%% — new entries blocked\n", drawdown*100)
	case LiquidateAllPositions:
		result.BlockNewEntries = true
		result.LiquidateAll = true
		logf("WARN: drawdown %.2f%% — liquidating all positions\n", drawdown*100)
	case NoAction, "":
	default:
		logf("WARN: unknown MaxDrawdownMode %q\n", s.MaxDrawdownMode)
	}

	return result
}

// FuncDrawdownPolicy wraps an optional override; unset delegates to fallback.
type FuncDrawdownPolicy struct {
	Fallback  *StandardDrawdownPolicy
	EvaluateFn func(DrawdownPolicyContext) DrawdownPolicyResult
}

func (f *FuncDrawdownPolicy) fallback() DrawdownPolicy {
	if f.Fallback != nil {
		return f.Fallback
	}
	return StandardDrawdownPolicy{}
}

// Evaluate delegates to EvaluateFn if set; otherwise falls back to the
// Fallback policy (or a zero-value [StandardDrawdownPolicy] if Fallback is
// nil). This allows complete replacement of drawdown logic while keeping the
// same Manager integration.
func (f *FuncDrawdownPolicy) Evaluate(ctx DrawdownPolicyContext) DrawdownPolicyResult {
	if f.EvaluateFn != nil {
		return f.EvaluateFn(ctx)
	}
	return f.fallback().Evaluate(ctx)
}

// toInterfaceAction converts policy result to the runner-facing type.
func toInterfaceAction(r DrawdownPolicyResult) interfaces.DrawdownAction {
	return interfaces.DrawdownAction{
		BlockNewEntries: r.BlockNewEntries,
		LiquidateAll:    r.LiquidateAll,
		DrawdownRate:    r.DrawdownRate,
	}
}
