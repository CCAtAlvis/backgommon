package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/data"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
	"github.com/CCAtAlvis/backgommon/pkg/risk"
	"github.com/CCAtAlvis/backgommon/pkg/runner"
	"github.com/CCAtAlvis/backgommon/pkg/sweep"
	"github.com/CCAtAlvis/backgommon/pkg/types"
	"github.com/CCAtAlvis/backgommon/pkg/worker"
)

const workerName = "sma_crossover"
const workerVersion = "0.1.0"

func tryWorkerMode() bool {
	return worker.Main(worker.Handlers{
		Describe: describeWorker,
		Run:      runWorker,
		Sweep:    sweepWorker,
	})
}

func describeWorker() (*worker.DescribeResponse, error) {
	schema := json.RawMessage(`{
  "type": "object",
  "properties": {
    "symbol": {"type": "string", "title": "Symbol", "default": ""},
    "short_window": {"type": "integer", "title": "Short window", "default": 20},
    "long_window": {"type": "integer", "title": "Long window", "default": 50},
    "quantity": {"type": "integer", "title": "Quantity", "default": 10},
    "data_path": {"type": "string", "title": "Data path (empty = synthetic)", "default": ""},
    "initial_capital": {"type": "number", "title": "Initial capital", "default": 100000}
  }
}`)
	return &worker.DescribeResponse{
		Name:        workerName,
		Version:     workerVersion,
		Description: "Long-only SMA crossover reference strategy",
		Schema:      schema,
	}, nil
}

type workerConfig struct {
	Symbol          string  `json:"symbol"`
	ShortWindow     int     `json:"short_window"`
	LongWindow      int     `json:"long_window"`
	Quantity        int     `json:"quantity"`
	DataPath        string  `json:"data_path"`
	DataSymbol      string  `json:"data_symbol"`
	InitialCapital  float64 `json:"initial_capital"`
	CustomField     string  `json:"custom_field"`
}

func parseWorkerConfig(raw json.RawMessage) (workerConfig, error) {
	cfg := workerConfig{
		ShortWindow:    20,
		LongWindow:     50,
		Quantity:       10,
		InitialCapital: 100000,
		CustomField:    "sma-crossover-worker",
	}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

func (c workerConfig) toStrategy() SMAStrategyConfig {
	return SMAStrategyConfig{
		Symbol:      c.Symbol,
		ShortWindow: c.ShortWindow,
		LongWindow:  c.LongWindow,
		Quantity:    c.Quantity,
		CustomField: c.CustomField,
	}
}

func loadTable(cfg workerConfig) (*types.TimeseriesTable[core.Candle], string, error) {
	if cfg.DataPath != "" {
		return data.LoadTableFromFile(cfg.DataPath, cfg.DataSymbol)
	}
	candles := data.GenerateSyntheticTrend(data.DefaultSyntheticTrendConfig())
	symbol := "SYNTH"
	if cfg.Symbol != "" {
		symbol = cfg.Symbol
	}
	t, err := data.LoadSingleSymbolTable(symbol, candles)
	return t, symbol, err
}

func runWorker(req worker.RunRequest) (*worker.RunResponse, error) {
	cfg, err := parseWorkerConfig(req.Config)
	if err != nil {
		return &worker.RunResponse{OK: false, Error: err.Error()}, nil
	}
	table, symbol, err := loadTable(cfg)
	if err != nil {
		return &worker.RunResponse{OK: false, Error: err.Error()}, nil
	}
	if cfg.Symbol == "" {
		cfg.Symbol = symbol
	}
	stratCfg := cfg.toStrategy()
	strat, err := NewSMACrossoverStrategy(stratCfg)
	if err != nil {
		return &worker.RunResponse{OK: false, Error: err.Error()}, nil
	}
	outDir := req.OutputDir
	if outDir == "" {
		outDir = "./output"
	}
	_ = os.MkdirAll(outDir, 0755)

	capital := cfg.InitialCapital
	if capital <= 0 {
		capital = 100000
	}
	r := runner.New(
		strat,
		runner.WithPortfolio(portfolio.New(&portfolio.Settings{InitialCapital: capital, EnableShorts: false})),
		runner.WithRiskManager(risk.New(&risk.Settings{
			MaxPortfolioDrawdownRate: 0.2, MaxLeverage: 1.0, MaxPositionAllocationRate: 1.0,
		})),
		runner.WithData(table),
		runner.WithIndicators(strat.Indicators()),
		runner.WithOutputDir(outDir),
		runner.WithReportSettings(stratCfg),
	)
	if err := r.Start(); err != nil {
		return &worker.RunResponse{OK: false, Error: err.Error()}, nil
	}
	metrics := map[string]float64{}
	if r.Results != nil {
		metrics["sharpe_ratio"] = r.Results.SharpeRatio
		metrics["cagr"] = r.Results.CAGR
		metrics["returns"] = r.Results.Returns
		metrics["max_drawdown"] = r.Results.MaxDrawdown
		metrics["final_capital"] = r.Results.FinalCapital
		if r.Results.Metrics != nil {
			for k, v := range r.Results.Metrics {
				metrics[k] = v
			}
		}
	}
	// Find newest run_* under outDir if ExportRun created a timestamped folder
	reportDir := outDir
	if entries, err := os.ReadDir(outDir); err == nil {
		var newest string
		var newestMod int64
		for _, e := range entries {
			if !e.IsDir() || len(e.Name()) < 4 || e.Name()[:4] != "run_" {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			if info.ModTime().UnixNano() >= newestMod {
				newestMod = info.ModTime().UnixNano()
				newest = filepath.Join(outDir, e.Name())
			}
		}
		if newest != "" {
			reportDir = newest
		}
	}
	return &worker.RunResponse{OK: true, ReportDir: reportDir, Metrics: metrics}, nil
}

func sweepWorker(req worker.SweepRequest) (*worker.SweepResponse, error) {
	baseCfg, err := parseWorkerConfig(req.Base)
	if err != nil {
		return &worker.SweepResponse{OK: false, Error: err.Error()}, nil
	}
	table, symbol, err := loadTable(baseCfg)
	if err != nil {
		return &worker.SweepResponse{OK: false, Error: err.Error()}, nil
	}
	if baseCfg.Symbol == "" {
		baseCfg.Symbol = symbol
	}

	var cases []SMAStrategyConfig
	if len(req.Cases) > 0 && string(req.Cases) != "null" && string(req.Cases) != "[]" {
		var wcases []workerConfig
		if err := json.Unmarshal(req.Cases, &wcases); err != nil {
			return &worker.SweepResponse{OK: false, Error: "cases: " + err.Error()}, nil
		}
		for _, wc := range wcases {
			if wc.Symbol == "" {
				wc.Symbol = baseCfg.Symbol
			}
			if wc.Quantity == 0 {
				wc.Quantity = baseCfg.Quantity
			}
			if wc.ShortWindow == 0 {
				wc.ShortWindow = baseCfg.ShortWindow
			}
			if wc.LongWindow == 0 {
				wc.LongWindow = baseCfg.LongWindow
			}
			cases = append(cases, wc.toStrategy())
		}
	} else {
		base := baseCfg.toStrategy()
		cases = sweep.Expand(base,
			[]sweep.Mutator[SMAStrategyConfig]{
				func(c *SMAStrategyConfig) { c.ShortWindow = 5 },
				func(c *SMAStrategyConfig) { c.ShortWindow = 10 },
			},
			[]sweep.Mutator[SMAStrategyConfig]{
				func(c *SMAStrategyConfig) { c.LongWindow = 20 },
				func(c *SMAStrategyConfig) { c.LongWindow = 30 },
			},
		)
	}
	valid := cases[:0]
	for _, c := range cases {
		if c.ShortWindow < c.LongWindow && c.ShortWindow > 0 && c.Quantity > 0 {
			valid = append(valid, c)
		}
	}
	if len(valid) == 0 {
		return &worker.SweepResponse{OK: false, Error: "no valid cases"}, nil
	}

	outDir := req.OutputDir
	if outDir == "" {
		outDir = "./output"
	}
	workers := req.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
		if workers > 4 {
			workers = 4
		}
	}
	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "sharpe_ratio"
	}
	capital := baseCfg.InitialCapital
	if capital <= 0 {
		capital = 100000
	}
	portfolioSettings := &portfolio.Settings{InitialCapital: capital, EnableShorts: false}
	riskSettings := &risk.Settings{
		MaxPortfolioDrawdownRate: 0.2, MaxLeverage: 1.0, MaxPositionAllocationRate: 1.0,
	}

	result, err := sweep.Run(sweep.Config[SMAStrategyConfig]{
		Cases:     valid,
		Workers:   workers,
		OutputDir: outDir,
		SortBy:    sortBy,
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
		return &worker.SweepResponse{OK: false, Error: err.Error()}, nil
	}
	return &worker.SweepResponse{
		OK:          true,
		SweepDir:    result.SweepDir,
		SummaryPath: result.SummaryJSON,
	}, nil
}
