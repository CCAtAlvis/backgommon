package portfolio

import "time"

// StandardPeriodicEventsModel is the default [PeriodicEventsModel]. It applies
// four scheduled cash flows based on [Settings] fields:
//   - SIP contributions (adds SIPAmount at SIPFrequency intervals)
//   - Idle cash interest (accrues IdleCashInterestAnnualRate on uninvested cash)
//   - Leverage cost (charges LeverageCostAnnualRate on borrowed notional)
//   - Management fee (deducts ManagementFeeAnnualRate on total equity)
//
// Each event fires only when sufficient time has elapsed since its last execution.
type StandardPeriodicEventsModel struct{}

// Apply evaluates all periodic cash events against the current bar time and
// returns the net CashDelta plus updated last-run timestamps.
func (StandardPeriodicEventsModel) Apply(ctx PeriodicContext) PeriodicResult {
	result := PeriodicResult{State: ctx.State}
	s := ctx.Settings

	if s.SIPAmount > 0 && s.SIPFrequency > 0 {
		if ctx.State.LastSIPTime.IsZero() || ctx.Time.Sub(ctx.State.LastSIPTime) >= s.SIPFrequency {
			result.CashDelta += s.SIPAmount
			result.State.LastSIPTime = ctx.Time
		}
	}

	if s.IdleCashInterestAnnualRate > 0 && s.IdleCashInterestFrequency > 0 {
		if ctx.State.LastIdleInterestTime.IsZero() || ctx.Time.Sub(ctx.State.LastIdleInterestTime) >= s.IdleCashInterestFrequency {
			if rate := periodRate(s.IdleCashInterestAnnualRate, s.IdleCashInterestFrequency); rate > 0 {
				result.CashDelta += ctx.Cash * rate
				result.State.LastIdleInterestTime = ctx.Time
			}
		}
	}

	if s.LeverageCostAnnualRate > 0 && s.LeverageCostFrequency > 0 {
		if ctx.State.LastLeverageCostTime.IsZero() || ctx.Time.Sub(ctx.State.LastLeverageCostTime) >= s.LeverageCostFrequency {
			if ctx.BorrowedNotional > 0 {
				if rate := periodRate(s.LeverageCostAnnualRate, s.LeverageCostFrequency); rate > 0 {
					result.CashDelta -= ctx.BorrowedNotional * rate
				}
			}
			result.State.LastLeverageCostTime = ctx.Time
		}
	}

	if s.EnableManagementFee && s.ManagementFeeAnnualRate > 0 && s.ManagementFeeFrequency > 0 {
		if ctx.State.LastManagementFeeTime.IsZero() || ctx.Time.Sub(ctx.State.LastManagementFeeTime) >= s.ManagementFeeFrequency {
			if rate := periodRate(s.ManagementFeeAnnualRate, s.ManagementFeeFrequency); rate > 0 {
				result.CashDelta -= ctx.Equity * rate
				result.State.LastManagementFeeTime = ctx.Time
			}
		}
	}

	return result
}

// FuncPeriodicEventsModel allows full replacement of periodic event logic via a
// single ApplyFn. If ApplyFn is nil, delegates to Fallback (or
// [StandardPeriodicEventsModel] if Fallback is also nil).
type FuncPeriodicEventsModel struct {
	Fallback *StandardPeriodicEventsModel
	ApplyFn  func(PeriodicContext) PeriodicResult
}

func (f *FuncPeriodicEventsModel) fallback() PeriodicEventsModel {
	if f.Fallback != nil {
		return f.Fallback
	}
	return StandardPeriodicEventsModel{}
}

// Apply delegates to ApplyFn if set, otherwise falls back to the standard model.
func (f *FuncPeriodicEventsModel) Apply(ctx PeriodicContext) PeriodicResult {
	if f.ApplyFn != nil {
		return f.ApplyFn(ctx)
	}
	return f.fallback().Apply(ctx)
}

func periodRate(annualRate float64, frequency time.Duration) float64 {
	periodsPerYear := hoursPerYear / frequency.Hours()
	if periodsPerYear <= 0 {
		return 0
	}
	return annualRate / periodsPerYear
}
