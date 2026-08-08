package output

import (
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/types"
)

// DrawdownPeriod represents a single peak-to-recovery drawdown event.
type DrawdownPeriod struct {
	StartDate   string  `json:"start_date"`
	EndDate     string  `json:"end_date"`
	PeakValue   float64 `json:"peak_value"`
	TroughValue float64 `json:"trough_value"`
	DrawdownPct float64 `json:"drawdown_pct"`
	DurationDays float64 `json:"duration_days"`
}

// CalculateDrawdownPeriods identifies all drawdown periods from an equity curve.
// Returns the max drawdown percentage and all drawdown periods.
func CalculateDrawdownPeriods(curve []types.AccountValue) (float64, []DrawdownPeriod) {
	if len(curve) < 2 {
		return 0, nil
	}

	var periods []DrawdownPeriod
	maxDrawdown := 0.0
	currentPeak := curve[0].Value
	peakDate := curve[0].Time
	var currentPeriod *DrawdownPeriod

	for i := 1; i < len(curve); i++ {
		current := curve[i]

		if current.Value > currentPeak {
			// New peak — close any open drawdown period
			if currentPeriod != nil {
				endDate := curve[i-1].Time
				currentPeriod.EndDate = endDate.Format("2006-01-02")
				currentPeriod.DurationDays = endDate.Sub(parseDateFallback(currentPeriod.StartDate)).Hours() / 24
				periods = append(periods, *currentPeriod)
				currentPeriod = nil
			}
			currentPeak = current.Value
			peakDate = current.Time
			continue
		}

		drawdown := (current.Value - currentPeak) / currentPeak
		if drawdown < maxDrawdown {
			maxDrawdown = drawdown
		}

		if currentPeriod == nil && drawdown < 0 {
			currentPeriod = &DrawdownPeriod{
				StartDate:   peakDate.Format("2006-01-02"),
				PeakValue:   currentPeak,
				TroughValue: current.Value,
				DrawdownPct: drawdown * 100,
			}
		}

		if currentPeriod != nil && current.Value < currentPeriod.TroughValue {
			currentPeriod.TroughValue = current.Value
			currentPeriod.DrawdownPct = ((current.Value - currentPeak) / currentPeak) * 100
		}
	}

	// Close any still-open period
	if currentPeriod != nil {
		endDate := curve[len(curve)-1].Time
		currentPeriod.EndDate = endDate.Format("2006-01-02")
		currentPeriod.DurationDays = endDate.Sub(parseDateFallback(currentPeriod.StartDate)).Hours() / 24
		periods = append(periods, *currentPeriod)
	}

	return maxDrawdown * 100, periods
}

func parseDateFallback(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}
