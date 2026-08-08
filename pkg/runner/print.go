package runner

import (
	"fmt"
	"math"

	"github.com/CCAtAlvis/backgommon/pkg/outmgr"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

// PrintResults writes a human-readable backtest summary via outmgr (buffered
// until Flush). Covers period, capital, returns, risk-adjusted ratios, and
// win/loss statistics. Safe to call with a nil Results pointer.
func PrintResults(r *types.Results) {
	if r == nil {
		outmgr.Println("No results available.")
		return
	}

	outmgr.Println("\n--- Backtest Results ---")
	outmgr.Printf("Period:          %s → %s\n", r.StartTime.Format("2006-01-02"), r.EndTime.Format("2006-01-02"))
	outmgr.Printf("Initial:         %.2f\n", r.InitialCapital)
	outmgr.Printf("Final:           %.2f\n", r.FinalCapital)
	outmgr.Printf("Return:          %.2f%%\n", r.Returns*100)
	outmgr.Printf("CAGR:            %.2f%%\n", r.CAGR*100)
	outmgr.Printf("Max Drawdown:    %.2f%%\n", r.MaxDrawdown*100)

	if !math.IsNaN(r.SharpeRatio) && r.SharpeRatio != 0 {
		outmgr.Printf("Sharpe Ratio:    %.3f\n", r.SharpeRatio)
	}
	if !math.IsNaN(r.SortinoRatio) && r.SortinoRatio != 0 {
		outmgr.Printf("Sortino Ratio:   %.3f\n", r.SortinoRatio)
	}
	if r.AnnualizedStdDev > 0 {
		outmgr.Printf("Std Dev (Ann.):  %.2f%%\n", r.AnnualizedStdDev*100)
	}
	if r.ProfitFactor > 0 && !math.IsInf(r.ProfitFactor, 0) {
		outmgr.Printf("Profit Factor:   %.2f\n", r.ProfitFactor)
	}

	rfrLabel := fmt.Sprintf("%.1f%% (%s)", r.RiskFreeRate*100, r.RiskFreeRateSource)
	outmgr.Printf("Risk-Free Rate:  %s\n", rfrLabel)

	outmgr.Printf("Total Trades:    %d\n", r.TotalTrades)
	if r.SkippedOrders > 0 {
		outmgr.Printf("Skipped Orders:  %d\n", r.SkippedOrders)
	}
	if r.TotalTrades > 0 {
		winRate := float64(r.WinningTrades) / float64(r.TotalTrades) * 100
		outmgr.Printf("Win Rate:        %.2f%% (%d W / %d L)\n", winRate, r.WinningTrades, r.LosingTrades)
	}

	if r.AvgWinPercent != 0 {
		outmgr.Printf("Avg Win:         +%.2f%%\n", r.AvgWinPercent*100)
	}
	if r.AvgLossPercent != 0 {
		outmgr.Printf("Avg Loss:        %.2f%%\n", r.AvgLossPercent*100)
	}

	if r.TotalCosts > 0 {
		outmgr.Printf("Total Costs:     %.2f\n", r.TotalCosts)
		outmgr.Printf("  Brokerage:     %.2f\n", r.TotalBrokerage)
		outmgr.Printf("  Transaction Tax: %.2f\n", r.TotalTransactionTax)
		outmgr.Printf("  Capital Gains Tax: %.2f\n", r.TotalCapitalGainsTax)
	}
}
