package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/data"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
	"github.com/CCAtAlvis/backgommon/pkg/risk"
	"github.com/CCAtAlvis/backgommon/pkg/runner"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

func main() {
	if tryWorkerMode() {
		return
	}

	dataPath := flag.String("data", "", "path to OHLCV file (.json or .csv); uses synthetic data if empty")
	dataSymbol := flag.String("symbol", "", "symbol name (required for .csv)")
	shortWindow := flag.Int("short", 20, "short SMA period (use 50 with 200+ bars of history)")
	longWindow := flag.Int("long", 50, "long SMA period (classic 50/200 needs years of data)")
	quantity := flag.Int("qty", 10, "shares per trade")
	doSweep := flag.Bool("sweep", false, "run a small SMA window parameter sweep (Clone mode + workers)")
	outputDir := flag.String("output", "./output", "directory for sweep reports (used with -sweep)")
	flag.Parse()

	cfg := SMAStrategyConfig{
		ShortWindow: *shortWindow,
		LongWindow:  *longWindow,
		Quantity:    *quantity,
		CustomField: "sma-crossover-example",
	}

	var table *types.TimeseriesTable[core.Candle]
	var symbol string

	if *dataPath != "" {
		t, sym, err := data.LoadTableFromFile(*dataPath, *dataSymbol)
		if err != nil {
			log.Fatalf("load data: %v", err)
		}
		table = t
		symbol = sym
		fmt.Printf("Loaded %d bars for %s from %s\n", len(table.Rows()), symbol, *dataPath)
	} else {
		candles := data.GenerateSyntheticTrend(data.DefaultSyntheticTrendConfig())
		symbol = "SYNTH"
		t, err := data.LoadSingleSymbolTable(symbol, candles)
		if err != nil {
			log.Fatalf("build synthetic table: %v", err)
		}
		table = t
		fmt.Printf("Using synthetic data: %d bars for %s\n", len(table.Rows()), symbol)
	}

	cfg.Symbol = symbol

	if *doSweep {
		runSMASweep(cfg, table, *outputDir)
		return
	}

	strat, err := NewSMACrossoverStrategy(cfg)
	if err != nil {
		log.Fatalf("strategy: %v", err)
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

	r := runner.New(
		strat,
		runner.WithPortfolio(portfolio.New(portfolioSettings)),
		runner.WithRiskManager(risk.New(riskSettings)),
		runner.WithData(table),
		runner.WithIndicators(strat.Indicators()),
	)

	if err := r.Start(); err != nil {
		log.Fatalf("backtest failed: %v", err)
	}

	runner.PrintResults(r.Results)

	fmt.Println("\nOpen Positions:")
	if len(r.Portfolio.Positions()) == 0 {
		fmt.Println("  (none)")
	}
	for sym, pos := range r.Portfolio.Positions() {
		fmt.Printf("  %s: qty=%d open=%.2f unrealized=%.2f\n", sym, pos.Quantity, pos.OpenPrice, pos.UnrealizedPnL)
	}

	if cfg.CustomField != "" {
		fmt.Printf("\nStrategy tag: %s\n", cfg.CustomField)
	}

	if *dataPath == "" {
		fmt.Fprintln(os.Stderr, "\nTip: pass -data /path/to/file.json or -data file.csv -symbol MY_SYM")
		fmt.Fprintln(os.Stderr, "     or -sweep to demo parameter search with Clone data mode")
	}
}
