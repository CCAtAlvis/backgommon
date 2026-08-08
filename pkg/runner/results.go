package runner

import (
	"math"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/outmgr"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
	"github.com/CCAtAlvis/backgommon/pkg/risk"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

// tradingDaysPerYear is the standard US equity market assumption used to
// annualize daily returns, standard deviations, and risk-adjusted ratios.
const tradingDaysPerYear = 252

// defaultRiskFreeRate is the assumed annualized risk-free rate (5%) used when
// risk.Settings.MetricsRiskFreeAnnualRate is not configured. The runner records
// "assumed" vs "provided" in Results.RiskFreeRateSource so reports make the
// source transparent.
const defaultRiskFreeRate = 0.05

// finalizeResults computes all performance metrics after the backtest loop
// completes: total return, CAGR, max drawdown, Sharpe/Sortino ratios, profit
// factor, win/loss statistics, and annualized volatility. Results are written
// into r.Results.
func (r *Runner) finalizeResults() {
	if r.Results == nil {
		r.Results = &types.Results{Metrics: make(map[string]float64)}
	}

	stats := r.Portfolio.Stats()
	initial := r.Portfolio.InitialCapital()
	final := r.Portfolio.Value()

	r.Results.InitialCapital = initial
	r.Results.FinalCapital = final
	r.Results.TotalTrades = stats.WinningTrades + stats.LosingTrades
	r.Results.WinningTrades = stats.WinningTrades
	r.Results.LosingTrades = stats.LosingTrades

	if initial > 0 {
		r.Results.Returns = (final - initial) / initial
	}

	r.Results.MaxDrawdown = maxDrawdownFromEquityCurve(r.EquityCurve)

	if len(r.EquityCurve) > 0 {
		r.Results.StartTime = r.EquityCurve[0].Time
		r.Results.EndTime = r.EquityCurve[len(r.EquityCurve)-1].Time
	}

	if r.Results.TotalTrades > 0 {
		r.Results.Metrics["win_rate"] = float64(r.Results.WinningTrades) / float64(r.Results.TotalTrades)
	}

	// CAGR
	r.Results.CAGR = computeCAGR(initial, final, r.Results.StartTime, r.Results.EndTime)

	// Risk-free rate resolution
	rfr := r.resolveRiskFreeRate()
	r.Results.RiskFreeRate = rfr

	// Sharpe and Sortino from daily returns
	r.Results.SharpeRatio, r.Results.SortinoRatio = computeRatios(r.EquityCurve, rfr)

	// Average win/loss percentages and holding periods from closed positions
	r.Results.AvgWinPercent, r.Results.AvgLossPercent,
		r.Results.AvgWinHoldingDays, r.Results.AvgLossHoldingDays = computeAvgWinLoss(r.Portfolio)

	// Profit factor
	r.Results.ProfitFactor = computeProfitFactor(r.Portfolio)

	// Annualized standard deviation
	r.Results.AnnualizedStdDev = computeAnnualizedStdDev(r.EquityCurve)

	// Cost accumulators
	r.Results.TotalBrokerage = stats.TotalBrokerage
	r.Results.TotalTransactionTax = stats.TotalTransactionTax
	r.Results.TotalCapitalGainsTax = stats.TotalCapitalGainsTax
	r.Results.TotalCosts = stats.TotalBrokerage + stats.TotalTransactionTax + stats.TotalCapitalGainsTax
	r.Results.SkippedOrders = r.skippedOrders
	if r.skippedOrders > 0 {
		outmgr.Printf("Note: skipped %d order(s) during backtest (lenient mode)\n", r.skippedOrders)
	}
}

func (r *Runner) resolveRiskFreeRate() float64 {
	if rm, ok := r.RiskManager.(*risk.Manager); ok {
		s := rm.Settings()
		if s != nil && s.MetricsRiskFreeAnnualRate > 0 {
			r.Results.RiskFreeRateSource = "provided"
			return s.MetricsRiskFreeAnnualRate
		}
	}
	r.Results.RiskFreeRateSource = "assumed"
	return defaultRiskFreeRate
}

func computeCAGR(initial, final float64, start, end time.Time) float64 {
	if initial <= 0 || final <= 0 {
		return 0
	}
	years := end.Sub(start).Hours() / (365.25 * 24)
	if years <= 0 {
		return 0
	}
	return math.Pow(final/initial, 1.0/years) - 1
}

func computeRatios(curve []types.AccountValue, riskFreeAnnual float64) (sharpe, sortino float64) {
	if len(curve) < 2 {
		return 0, 0
	}

	dailyRFR := riskFreeAnnual / tradingDaysPerYear
	n := len(curve) - 1
	returns := make([]float64, n)

	for i := 1; i < len(curve); i++ {
		if curve[i-1].Value > 0 {
			returns[i-1] = (curve[i].Value - curve[i-1].Value) / curve[i-1].Value
		}
	}

	// Mean excess return
	sumExcess := 0.0
	for _, ret := range returns {
		sumExcess += (ret - dailyRFR)
	}
	meanExcess := sumExcess / float64(n)

	// Standard deviation of excess returns (for Sharpe)
	sumSqDev := 0.0
	for _, ret := range returns {
		dev := (ret - dailyRFR) - meanExcess
		sumSqDev += dev * dev
	}
	stdDev := math.Sqrt(sumSqDev / float64(n))

	if stdDev > 0 {
		sharpe = (meanExcess / stdDev) * math.Sqrt(tradingDaysPerYear)
	}

	// Downside deviation (for Sortino) — only negative excess returns
	sumSqDown := 0.0
	for _, ret := range returns {
		excess := ret - dailyRFR
		if excess < 0 {
			sumSqDown += excess * excess
		}
	}
	downDev := math.Sqrt(sumSqDown / float64(n))

	if downDev > 0 {
		sortino = (meanExcess / downDev) * math.Sqrt(tradingDaysPerYear)
	}

	return sharpe, sortino
}

func computeAvgWinLoss(pm interface{ ClosedPositions() []*portfolio.Position }) (avgWin, avgLoss, avgWinDays, avgLossDays float64) {
	closed := pm.ClosedPositions()
	var winSum, lossSum float64
	var winDaysSum, lossDaysSum float64
	var wins, losses int

	for _, pos := range closed {
		if pos.OpenPrice <= 0 {
			continue
		}
		pctReturn := (pos.ClosePrice - pos.OpenPrice) / pos.OpenPrice
		holdDays := pos.CloseTime.Sub(pos.OpenTime).Hours() / 24
		if pos.RealizedPnL > 0 {
			winSum += pctReturn
			winDaysSum += holdDays
			wins++
		} else if pos.RealizedPnL < 0 {
			lossSum += pctReturn
			lossDaysSum += holdDays
			losses++
		}
	}

	if wins > 0 {
		avgWin = winSum / float64(wins)
		avgWinDays = winDaysSum / float64(wins)
	}
	if losses > 0 {
		avgLoss = lossSum / float64(losses)
		avgLossDays = lossDaysSum / float64(losses)
	}
	return avgWin, avgLoss, avgWinDays, avgLossDays
}

func computeProfitFactor(pm interface{ ClosedPositions() []*portfolio.Position }) float64 {
	closed := pm.ClosedPositions()
	var grossProfit, grossLoss float64

	for _, pos := range closed {
		if pos.RealizedPnL > 0 {
			grossProfit += pos.RealizedPnL
		} else if pos.RealizedPnL < 0 {
			grossLoss += -pos.RealizedPnL
		}
	}

	if grossLoss == 0 {
		if grossProfit > 0 {
			return math.Inf(1)
		}
		return 0
	}
	return grossProfit / grossLoss
}

func computeAnnualizedStdDev(curve []types.AccountValue) float64 {
	if len(curve) < 2 {
		return 0
	}

	n := len(curve) - 1
	returns := make([]float64, n)
	sum := 0.0

	for i := 1; i < len(curve); i++ {
		if curve[i-1].Value > 0 {
			returns[i-1] = (curve[i].Value - curve[i-1].Value) / curve[i-1].Value
		}
		sum += returns[i-1]
	}

	mean := sum / float64(n)
	sumSq := 0.0
	for _, r := range returns {
		d := r - mean
		sumSq += d * d
	}

	dailyStd := math.Sqrt(sumSq / float64(n))
	return dailyStd * math.Sqrt(tradingDaysPerYear)
}

func maxDrawdownFromEquityCurve(curve []types.AccountValue) float64 {
	if len(curve) == 0 {
		return 0
	}

	peak := curve[0].Value
	maxDD := 0.0

	for _, point := range curve {
		if point.Value > peak {
			peak = point.Value
		}
		if peak <= 0 {
			continue
		}
		dd := (peak - point.Value) / peak
		if dd > maxDD {
			maxDD = dd
		}
	}

	return maxDD
}
