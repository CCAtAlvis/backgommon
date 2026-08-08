// Package sweep runs typed parameter grids and budgeted searches over backtests:
// Expand a Cartesian product, or RunSearch / RunStaged with random/TPE suggestors,
// optional Goals (constraints / targets), and optional fidelity screening.
// Trials export per-trial reports and a ranked summary.
package sweep

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/outmgr"
	"github.com/CCAtAlvis/backgommon/pkg/output"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

// Mutator applies a typed parameter change to a config copy.
type Mutator[T any] func(*T)

// Expand builds the Cartesian product of mutator axes applied to base.
// Each axis is a list of alternatives; empty axes are skipped.
// With no axes, Expand returns a single-element slice containing base.
func Expand[T any](base T, axes ...[]Mutator[T]) []T {
	results := []T{base}
	for _, axis := range axes {
		if len(axis) == 0 {
			continue
		}
		next := make([]T, 0, len(results)*len(axis))
		for _, current := range results {
			for _, mut := range axis {
				cfg := current
				mut(&cfg)
				next = append(next, cfg)
			}
		}
		results = next
	}
	return results
}

// DataMode selects how market data is provided to each trial.
// Exactly one of Shared, Clone, or Factory must be set.
type DataMode[T any] struct {
	// Shared reuses one read-only table across all trials (and workers).
	Shared *types.TimeseriesTable[core.Candle]
	// Clone deep-copies this template table once per trial.
	Clone *types.TimeseriesTable[core.Candle]
	// Factory builds or loads a table for each trial from that trial's config.
	Factory func(cfg T) (*types.TimeseriesTable[core.Candle], error)
}

func (d DataMode[T]) validate() error {
	n := 0
	if d.Shared != nil {
		n++
	}
	if d.Clone != nil {
		n++
	}
	if d.Factory != nil {
		n++
	}
	if n != 1 {
		return fmt.Errorf("DataMode: exactly one of Shared, Clone, Factory must be set (got %d)", n)
	}
	return nil
}

func (d DataMode[T]) resolve(cfg T) (*types.TimeseriesTable[core.Candle], error) {
	switch {
	case d.Shared != nil:
		return d.Shared, nil
	case d.Clone != nil:
		return types.CloneCandleTable(d.Clone), nil
	case d.Factory != nil:
		return d.Factory(cfg)
	default:
		return nil, fmt.Errorf("DataMode: no data source configured")
	}
}

// Config configures a parameter sweep over typed cases T.
type Config[T any] struct {
	Cases     []T
	Workers   int    // default 1
	OutputDir string // parent; creates sweep_<timestamp>/
	SortBy    string // default "sharpe_ratio"
	FailFast  bool   // stop scheduling new trials after first error
	Progress  bool   // log trial completion to stdout

	Data DataMode[T]

	// RunTrial wires strategy/portfolio/risk/runner for one case.
	// reportDir is a unique empty folder for this trial's HTML/JSON export.
	RunTrial func(cfg T, data *types.TimeseriesTable[core.Candle], reportDir string) (*types.Results, error)
}

// Trial is one completed (or failed) parameter combination.
type Trial[T any] struct {
	Index      int
	Params     T
	Results    *types.Results
	Duration   time.Duration
	ReportDir  string
	ReportHTML string // relative URL from sweep root, e.g. viewer.html?trial=000
	Err        error
}

// Result is the outcome of a full sweep.
type Result[T any] struct {
	SweepDir      string
	Trials        []Trial[T] // ranked by SortBy when writing summary
	SummaryJSON   string
	SummaryCSV    string
	DashboardHTML string
}

// Run executes all Cases, writes per-trial reports under OutputDir/sweep_*/trials/,
// and writes ranked summary.json / summary.csv at the sweep root.
func Run[T any](cfg Config[T]) (*Result[T], error) {
	if len(cfg.Cases) == 0 {
		return nil, fmt.Errorf("sweep: Cases must not be empty")
	}
	if cfg.RunTrial == nil {
		return nil, fmt.Errorf("sweep: RunTrial is required")
	}
	if cfg.OutputDir == "" {
		return nil, fmt.Errorf("sweep: OutputDir is required")
	}
	if err := cfg.Data.validate(); err != nil {
		return nil, fmt.Errorf("sweep: %w", err)
	}

	workers := cfg.Workers
	if workers <= 0 {
		workers = 1
	}
	sortBy := cfg.SortBy
	if sortBy == "" {
		sortBy = "sharpe_ratio"
	}

	ts := time.Now().Format("2006-01-02T15-04-05.000")
	sweepDir := filepath.Join(cfg.OutputDir, "sweep_"+ts)
	trialsDir := filepath.Join(sweepDir, "trials")
	if err := os.MkdirAll(trialsDir, 0755); err != nil {
		return nil, fmt.Errorf("sweep: create dir: %w", err)
	}

	// Shared tables may still be "dirty" (unsorted). Force a single-threaded
	// Rows() pass so parallel workers do not race on timestamp sorting.
	if cfg.Data.Shared != nil {
		_ = cfg.Data.Shared.Rows()
	}

	type job struct {
		index int
		case_ T
	}
	jobs := make(chan job, len(cfg.Cases))
	for i, c := range cfg.Cases {
		jobs <- job{index: i, case_: c}
	}
	close(jobs)

	trials := make([]Trial[T], len(cfg.Cases))
	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		failOnce sync.Once
		failed   bool
	)

	runOne := func(workerID int, j job) {
		start := time.Now()
		reportDir := filepath.Join(trialsDir, fmt.Sprintf("trial_%03d", j.index))
		trial := Trial[T]{
			Index:      j.index,
			Params:     j.case_,
			ReportDir:  reportDir,
			ReportHTML: "viewer.html?trial=" + fmt.Sprintf("%03d", j.index),
		}

		if cfg.FailFast {
			mu.Lock()
			abort := failed
			mu.Unlock()
			if abort {
				trial.Err = fmt.Errorf("skipped: fail-fast after prior error")
				trials[j.index] = trial
				return
			}
		}

		data, err := cfg.Data.resolve(j.case_)
		if err != nil {
			trial.Err = err
			trial.Duration = time.Since(start)
			trials[j.index] = trial
			if cfg.FailFast {
				failOnce.Do(func() {
					mu.Lock()
					failed = true
					mu.Unlock()
				})
			}
			return
		}

		results, err := cfg.RunTrial(j.case_, data, reportDir)
		trial.Results = results
		trial.Err = err
		trial.Duration = time.Since(start)
		trials[j.index] = trial

		if err != nil && cfg.FailFast {
			failOnce.Do(func() {
				mu.Lock()
				failed = true
				mu.Unlock()
			})
		}

		if cfg.Progress {
			status := "ok"
			if err != nil {
				status = err.Error()
			}
			outmgr.Printf("sweep: trial %d/%d done in %s (worker %d): %s\n",
				j.index+1, len(cfg.Cases), trial.Duration.Round(time.Millisecond), workerID, status)
		}
	}

	wg.Add(workers)
	for w := 0; w < workers; w++ {
		workerID := w + 1
		go func(id int) {
			defer wg.Done()
			for j := range jobs {
				runOne(id, j)
			}
		}(workerID)
	}
	wg.Wait()

	ranked := append([]Trial[T](nil), trials...)
	Rank(ranked, sortBy)

	summary, err := buildSummary(sweepDir, sortBy, ranked)
	if err != nil {
		return nil, err
	}

	summaryJSON := filepath.Join(sweepDir, "summary.json")
	summaryCSV := filepath.Join(sweepDir, "summary.csv")
	dashboardHTML := filepath.Join(sweepDir, "dashboard.html")
	viewerHTML := filepath.Join(sweepDir, "viewer.html")
	if err := writeSummaryJSON(summaryJSON, summary); err != nil {
		return nil, err
	}
	if err := writeSummaryCSV(summaryCSV, ranked); err != nil {
		return nil, err
	}
	if err := writeDashboardHTML(dashboardHTML, summary); err != nil {
		return nil, err
	}
	if err := output.WriteViewerHTML(viewerHTML); err != nil {
		return nil, fmt.Errorf("sweep: write viewer.html: %w", err)
	}

	return &Result[T]{
		SweepDir:      sweepDir,
		Trials:        ranked,
		SummaryJSON:   summaryJSON,
		SummaryCSV:    summaryCSV,
		DashboardHTML: dashboardHTML,
	}, nil
}

// Rank sorts trials in place by the named metric. Higher is better except for
// max_drawdown (lower is better). Failed trials (Err != nil or nil Results) sort last.
func Rank[T any](trials []Trial[T], metric string) {
	asc := metric == "max_drawdown"
	sort.SliceStable(trials, func(i, j int) bool {
		a, aOK := metricValue(trials[i], metric)
		b, bOK := metricValue(trials[j], metric)
		if aOK != bOK {
			return aOK // successful before failed
		}
		if !aOK {
			return trials[i].Index < trials[j].Index
		}
		if asc {
			return a < b
		}
		return a > b
	})
}

func metricValue[T any](t Trial[T], metric string) (float64, bool) {
	if t.Err != nil || t.Results == nil {
		return 0, false
	}
	r := t.Results
	switch metric {
	case "sharpe_ratio", "sharpe":
		return r.SharpeRatio, true
	case "sortino_ratio", "sortino":
		return r.SortinoRatio, true
	case "cagr":
		return r.CAGR, true
	case "returns", "return":
		return r.Returns, true
	case "max_drawdown", "drawdown":
		return r.MaxDrawdown, true
	case "profit_factor":
		return r.ProfitFactor, true
	case "final_capital":
		return r.FinalCapital, true
	default:
		if r.Metrics != nil {
			if v, ok := r.Metrics[metric]; ok {
				return v, true
			}
		}
		return 0, false
	}
}

func writeSummaryCSV[T any](path string, trials []Trial[T]) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	header := []string{
		"index", "error", "duration_ms", "report_html",
		"returns", "cagr", "max_drawdown", "sharpe_ratio", "sortino_ratio",
		"profit_factor", "final_capital", "total_trades", "params_json",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, t := range trials {
		paramsJSON, _ := json.Marshal(t.Params)
		errStr := ""
		if t.Err != nil {
			errStr = t.Err.Error()
		}
		var returns, cagr, dd, sharpe, sortino, pf, finalCap, trades string
		if t.Results != nil {
			returns = fmt.Sprintf("%g", t.Results.Returns)
			cagr = fmt.Sprintf("%g", t.Results.CAGR)
			dd = fmt.Sprintf("%g", t.Results.MaxDrawdown)
			sharpe = fmt.Sprintf("%g", t.Results.SharpeRatio)
			sortino = fmt.Sprintf("%g", t.Results.SortinoRatio)
			pf = fmt.Sprintf("%g", t.Results.ProfitFactor)
			finalCap = fmt.Sprintf("%g", t.Results.FinalCapital)
			trades = fmt.Sprintf("%d", t.Results.TotalTrades)
		}
		row := []string{
			fmt.Sprintf("%d", t.Index),
			errStr,
			fmt.Sprintf("%d", t.Duration.Milliseconds()),
			t.ReportHTML,
			returns, cagr, dd, sharpe, sortino, pf, finalCap, trades,
			string(paramsJSON),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}
