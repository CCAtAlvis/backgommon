// Package execution handles order fill simulation and slippage modeling for
// the backgommon backtesting framework. It provides pluggable fill pricing
// strategies (which bar price to use as the execution price) and slippage
// models (how much to adjust that price for realistic market impact).
//
// The two primary extension points are [interfaces.FillPricer] and
// [SlippageModel], both injectable via runner options.
package execution

import (
	"fmt"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

// FillMode names a built-in bar-based fill assumption. Each mode selects a
// specific price from the OHLCV candle as the simulated execution price.
// Use [FillModeFromString] to convert configuration strings, or reference
// the constants directly when calling [NewStandardFillPricer].
type FillMode string

const (
	// FillCurrentClose uses the signal bar's close price. This is the most
	// common default and assumes the order executes at the end of the bar
	// that generated the signal.
	FillCurrentClose FillMode = "CurrentBarClose"

	// FillCurrentOpen uses the signal bar's open price. Useful when modeling
	// strategies that react to pre-market information and execute at open.
	FillCurrentOpen FillMode = "CurrentBarOpen"

	// FillCurrentHigh uses the signal bar's high price. Pessimistic for buy
	// orders (pays the highest price in the bar); optimistic for sells.
	FillCurrentHigh FillMode = "CurrentBarHigh"

	// FillCurrentLow uses the signal bar's low price. Pessimistic for sell
	// orders (receives the lowest price in the bar); optimistic for buys.
	FillCurrentLow FillMode = "CurrentBarLow"

	// FillMidPrice uses the average of the bar's high and low — a simple
	// proxy for the bar's typical traded price.
	FillMidPrice FillMode = "MidPrice"

	// FillOpenCloseAvg uses the average of the bar's open and close,
	// approximating the session's average price when volume data is absent.
	FillOpenCloseAvg FillMode = "OpenCloseAvg"

	// FillNextBarOpen uses the next bar's open price. This is the most
	// realistic mode for daily-bar strategies because the signal is
	// generated after the close and the order cannot realistically fill
	// until the next session opens. Returns an error if no next bar exists.
	FillNextBarOpen FillMode = "NextBarOpen"

	// FillWorstCase selects the most adverse price within the bar:
	// high for long entries (pay the most), low for long exits (receive the
	// least), and the inverse for short positions. Useful for conservative
	// performance estimates.
	FillWorstCase FillMode = "WorstCaseWithinBar"
)

// StandardFillPricer implements FillPricer for a named FillMode.
type StandardFillPricer struct {
	Mode FillMode
}

// NewStandardFillPricer returns a fill pricer for the given mode.
func NewStandardFillPricer(mode FillMode) *StandardFillPricer {
	return &StandardFillPricer{Mode: mode}
}

// FillPrice computes the execution price for ctx using the configured mode.
func (p *StandardFillPricer) FillPrice(ctx interfaces.FillContext) (float64, error) {
	return priceForMode(p.Mode, ctx)
}

// FuncFillPricer wraps a custom fill function supplied by the developer.
type FuncFillPricer struct {
	Fn func(interfaces.FillContext) (float64, error)
}

// NewFuncFillPricer returns a FillPricer that delegates to fn.
func NewFuncFillPricer(fn func(interfaces.FillContext) (float64, error)) *FuncFillPricer {
	return &FuncFillPricer{Fn: fn}
}

// FillPrice delegates to the wrapped function. Returns an error if the
// function was not set (nil).
func (p *FuncFillPricer) FillPrice(ctx interfaces.FillContext) (float64, error) {
	if p.Fn == nil {
		return 0, fmt.Errorf("fill function is nil")
	}
	return p.Fn(ctx)
}

// FillModeFromString converts a configuration string (e.g. from JSON or YAML)
// into the corresponding FillMode constant. If s does not match any known
// mode, it silently defaults to [FillCurrentClose] rather than returning an
// error, so callers should validate input separately when strict parsing is
// needed.
func FillModeFromString(s string) FillMode {
	switch FillMode(s) {
	case FillCurrentOpen, FillCurrentHigh, FillCurrentLow,
		FillMidPrice, FillOpenCloseAvg, FillNextBarOpen, FillWorstCase:
		return FillMode(s)
	default:
		return FillCurrentClose
	}
}

func priceForMode(mode FillMode, ctx interfaces.FillContext) (float64, error) {
	c := ctx.CurrentBar
	ord := ctx.Order

	switch mode {
	case FillCurrentClose:
		return c.Close, nil
	case FillCurrentOpen:
		return c.Open, nil
	case FillCurrentHigh:
		return c.High, nil
	case FillCurrentLow:
		return c.Low, nil
	case FillMidPrice:
		return (c.High + c.Low) / 2, nil
	case FillOpenCloseAvg:
		return (c.Open + c.Close) / 2, nil
	case FillNextBarOpen:
		if ctx.NextBar == nil {
			return 0, fmt.Errorf("next bar open fill: no next bar for %s", ord.Instrument)
		}
		return ctx.NextBar.Open, nil
	case FillWorstCase:
		return worstCasePrice(ord, c), nil
	default:
		return 0, fmt.Errorf("unknown fill mode %q", mode)
	}
}

func worstCasePrice(ord portfolio.Order, c core.Candle) float64 {
	switch ord.Type {
	case portfolio.Entry:
		if ord.Side == portfolio.Long {
			return c.High
		}
		return c.Low
	case portfolio.Exit:
		if ord.Side == portfolio.Long {
			return c.Low
		}
		return c.High
	default:
		return c.Close
	}
}
