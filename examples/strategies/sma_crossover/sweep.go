package main

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
	"github.com/CCAtAlvis/backgommon/pkg/risk"
	"github.com/CCAtAlvis/backgommon/pkg/runner"
	"github.com/CCAtAlvis/backgommon/pkg/sweep"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

// runSMASweep demonstrates Clone data mode + parallel workers + per-trial HTML reports.
// Indicator windows differ per trial, so each trial gets a deep-copied table.
func runSMASweep(base SMAStrategyConfig, table *types.TimeseriesTable[core.Candle], outputDir string) {
	cases := sweep.Expand(base,
		[]sweep.Mutator[SMAStrategyConfig]{
			func(c *SMAStrategyConfig) { c.ShortWindow = 5 },
			func(c *SMAStrategyConfig) { c.ShortWindow = 10 },
		},
		[]sweep.Mutator[SMAStrategyConfig]{
			func(c *SMAStrategyConfig) { c.LongWindow = 20 },
			func(c *SMAStrategyConfig) { c.LongWindow = 30 },
		},
	)

	// Drop invalid short >= long combinations (should not happen with this grid).
	valid := cases[:0]
	for _, c := range cases {
		if c.ShortWindow < c.LongWindow {
			valid = append(valid, c)
		}
	}

	portfolioSettings := &portfolio.Settings{
		InitialCapital: 100000,
		EnableShorts:   false,
	}
	riskSettings := &risk.Settings{
		MaxPortfolioDrawdownRate:  0.2,
		MaxLeverage:               1.0,
		MaxPositionAllocationRate: 1.0,
		EnableStopLoss:            false,
		EnableTakeProfit:          false,
		EnableTrailingStop:        false,
	}

	workers := runtime.NumCPU()
	if workers > 4 {
		workers = 4
	}

	fmt.Printf("SMA sweep: %d cases, %d workers, Clone data mode\n", len(valid), workers)

	result, err := sweep.Run(sweep.Config[SMAStrategyConfig]{
		Cases:     valid,
		Workers:   workers,
		OutputDir: outputDir,
		SortBy:    "sharpe_ratio",
		Progress:  true,
		Data:      sweep.DataMode[SMAStrategyConfig]{Clone: table},
		RunTrial: func(cfg SMAStrategyConfig, data *types.TimeseriesTable[core.Candle], reportDir string) (*types.Results, error) {
			strat, err := NewSMACrossoverStrategy(cfg)
			if err != nil {
				return nil, err
			}
			r := runner.New(
				strat,
				runner.WithPortfolio(portfolio.New(portfolioSettings)),
				runner.WithRiskManager(risk.New(riskSettings)),
				runner.WithData(data),
				runner.WithIndicators(strat.Indicators()),
				runner.WithReportDir(reportDir),
				runner.WithReportSettings(cfg),
			)
			if err := r.Start(); err != nil {
				return nil, err
			}
			return r.Results, nil
		},
	})
	if err != nil {
		log.Fatalf("sweep failed: %v", err)
	}

	fmt.Printf("\nSweep complete: %s\n", result.SweepDir)
	fmt.Printf("Dashboard: %s\n", result.DashboardHTML)
	fmt.Printf("Summary: %s\n", result.SummaryJSON)
	fmt.Println("\nTop trials by Sharpe:")
	n := 5
	if len(result.Trials) < n {
		n = len(result.Trials)
	}
	for i := 0; i < n; i++ {
		tr := result.Trials[i]
		if tr.Err != nil || tr.Results == nil {
			fmt.Printf("  #%d trial_%03d short=%d long=%d ERROR: %v\n", i+1, tr.Index, tr.Params.ShortWindow, tr.Params.LongWindow, tr.Err)
			continue
		}
		fmt.Printf("  #%d trial_%03d short=%d long=%d sharpe=%.3f cagr=%.2f%% dd=%.2f%% report=%s\n",
			i+1, tr.Index, tr.Params.ShortWindow, tr.Params.LongWindow,
			tr.Results.SharpeRatio, tr.Results.CAGR*100, tr.Results.MaxDrawdown*100,
			filepath.Join(result.SweepDir, tr.ReportHTML),
		)
	}
}
