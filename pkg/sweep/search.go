package sweep

import (
	"fmt"
	"math"
	"math/rand"
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

// SearchFidelity configures Hyperband-lite: evaluate on a cheap window, then promote top-K.
type SearchFidelity[T any] struct {
	// Cheap mutates a config copy for the screening stage.
	Cheap func(cfg T) T
	// Full mutates a config copy for the promotion stage. Nil = use survivor params as-is.
	Full func(cfg T) T
	// ScreenFraction is the fraction of MaxTrials used in the cheap stage (default 0.7).
	ScreenFraction float64
	// PromoteTopK survivors re-run at full fidelity (default max(10, MaxTrials/20)).
	PromoteTopK int
}

// SearchConfig configures budgeted parameter search (random / TPE / custom Suggestor).
type SearchConfig[T any] struct {
	Space     Space[T]
	Suggestor Suggestor[T]
	MaxTrials int
	Workers   int
	OutputDir string
	SortBy    string // default sharpe_ratio; used when Goals do not redefine ranking
	FailFast  bool
	Progress  bool
	Data      DataMode[T]
	RunTrial  func(cfg T, data *types.TimeseriesTable[core.Candle], reportDir string) (*types.Results, error)
	Goals     *Goals
	Fidelity  *SearchFidelity[T]
	Seed      int64 // for Random/TPE when Suggestor is nil; 0 = time-based
}

// RunSearch evaluates up to MaxTrials configs from Suggestor (or random).
func RunSearch[T any](cfg SearchConfig[T]) (*Result[T], error) {
	if cfg.RunTrial == nil {
		return nil, fmt.Errorf("sweep: RunTrial is required")
	}
	if cfg.OutputDir == "" {
		return nil, fmt.Errorf("sweep: OutputDir is required")
	}
	if cfg.MaxTrials <= 0 {
		return nil, fmt.Errorf("sweep: MaxTrials must be > 0")
	}
	if len(cfg.Space.Axes) == 0 {
		return nil, fmt.Errorf("sweep: Space has no axes")
	}
	if err := cfg.Data.validate(); err != nil {
		return nil, fmt.Errorf("sweep: %w", err)
	}
	if err := ValidateGoals(cfg.Goals); err != nil {
		return nil, err
	}

	workers := cfg.Workers
	if workers <= 0 {
		workers = 1
	}
	sortBy := cfg.SortBy
	if sortBy == "" {
		sortBy = "sharpe_ratio"
	}
	suggestor := cfg.Suggestor
	if suggestor == nil {
		seed := cfg.Seed
		if seed == 0 {
			seed = time.Now().UnixNano()
		}
		suggestor = SuggestRandom[T](rand.New(rand.NewSource(seed)))
	}

	ts := time.Now().Format("2006-01-02T15-04-05.000")
	sweepDir := filepath.Join(cfg.OutputDir, "search_"+ts)
	trialsDir := filepath.Join(sweepDir, "trials")
	if err := os.MkdirAll(trialsDir, 0755); err != nil {
		return nil, fmt.Errorf("sweep: create dir: %w", err)
	}
	if cfg.Data.Shared != nil {
		_ = cfg.Data.Shared.Rows()
	}

	var all []searchTrial[T]
	if cfg.Fidelity != nil && cfg.Fidelity.Cheap != nil {
		var err error
		all, err = runSearchWithFidelity(cfg, suggestor, workers, sortBy, sweepDir, trialsDir)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		all, err = runSearchLoop(cfg, suggestor, workers, sortBy, trialsDir, cfg.MaxTrials, nil, 0)
		if err != nil {
			return nil, err
		}
	}

	trials := make([]Trial[T], len(all))
	for i, st := range all {
		trials[i] = st.Trial
	}
	rankSearchTrials(trials, sortBy, cfg.Goals)

	summary, err := buildSummary(sweepDir, sortBy, trials)
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
	if err := writeSummaryCSV(summaryCSV, trials); err != nil {
		return nil, err
	}
	if err := writeDashboardHTML(dashboardHTML, summary); err != nil {
		return nil, err
	}
	if err := output.WriteViewerHTML(viewerHTML); err != nil {
		return nil, fmt.Errorf("sweep: write viewer.html: %w", err)
	}

	if cfg.Progress {
		outmgr.Printf("sweep search (%s): %d trials → %s\n", suggestor.Name(), len(trials), sweepDir)
	}

	return &Result[T]{
		SweepDir:      sweepDir,
		Trials:        trials,
		SummaryJSON:   summaryJSON,
		SummaryCSV:    summaryCSV,
		DashboardHTML: dashboardHTML,
	}, nil
}

type searchTrial[T any] struct {
	Trial   Trial[T]
	Indices []int
}

func runSearchWithFidelity[T any](
	cfg SearchConfig[T],
	suggestor Suggestor[T],
	workers int,
	sortBy string,
	sweepDir, trialsDir string,
) ([]searchTrial[T], error) {
	fid := cfg.Fidelity
	frac := fid.ScreenFraction
	if frac <= 0 || frac >= 1 {
		frac = 0.7
	}
	screenN := int(math.Ceil(float64(cfg.MaxTrials) * frac))
	if screenN < 1 {
		screenN = 1
	}
	if screenN > cfg.MaxTrials {
		screenN = cfg.MaxTrials
	}
	promoteK := fid.PromoteTopK
	if promoteK <= 0 {
		promoteK = cfg.MaxTrials / 20
		if promoteK < 10 {
			promoteK = 10
		}
	}
	if promoteK > screenN {
		promoteK = screenN
	}

	cheapPrepare := func(c T) T { return fid.Cheap(c) }
	screen, err := runSearchLoop(cfg, suggestor, workers, sortBy, trialsDir, screenN, cheapPrepare, 0)
	if err != nil {
		return nil, err
	}
	// Rank screen results and take top-K configs (by indices / params).
	screenTrials := make([]Trial[T], len(screen))
	for i, st := range screen {
		screenTrials[i] = st.Trial
	}
	rankSearchTrials(screenTrials, sortBy, cfg.Goals)

	type promo struct {
		cfg     T
		indices []int
	}
	seen := map[string]bool{}
	var promos []promo
	for _, t := range screenTrials {
		if len(promos) >= promoteK {
			break
		}
		if t.Err != nil || t.Results == nil {
			continue
		}
		gs := EvaluateGoals(t.Results, cfg.Goals)
		if cfg.Goals != nil && len(cfg.Goals.Constraints) > 0 && !gs.Feasible {
			continue
		}
		key := fmt.Sprintf("%v", t.Params)
		if seen[key] {
			continue
		}
		seen[key] = true
		// Find indices from screen list
		idx := []int(nil)
		for _, st := range screen {
			if st.Trial.Index == t.Index {
				idx = st.Indices
				break
			}
		}
		c := t.Params
		if fid.Full != nil {
			c = fid.Full(c)
		}
		promos = append(promos, promo{cfg: c, indices: idx})
	}

	if cfg.Progress {
		outmgr.Printf("sweep fidelity: screened %d, promoting %d to full\n", len(screen), len(promos))
	}

	// Re-run promotions as a fixed case list via internal eval (not suggestor).
	full := make([]searchTrial[T], 0, len(promos))
	offset := screenN
	for i, p := range promos {
		st, err := evalOne(cfg, workers, trialsDir, offset+i, p.cfg, p.indices)
		if err != nil {
			return nil, err
		}
		full = append(full, st)
		if cfg.Progress {
			status := "ok"
			if st.Trial.Err != nil {
				status = st.Trial.Err.Error()
			}
			outmgr.Printf("sweep: promote %d/%d done in %s: %s\n",
				i+1, len(promos), st.Trial.Duration.Round(time.Millisecond), status)
		}
	}
	// Return screen + full (full overwrites interest); keep all for summary.
	return append(screen, full...), nil
}

func runSearchLoop[T any](
	cfg SearchConfig[T],
	suggestor Suggestor[T],
	workers int,
	sortBy string,
	trialsDir string,
	maxTrials int,
	prepare func(T) T,
	indexOffset int,
) ([]searchTrial[T], error) {
	history := make([]Observation, 0, maxTrials)
	out := make([]searchTrial[T], 0, maxTrials)
	var histMu sync.Mutex
	nextIndex := indexOffset

	for len(out) < maxTrials {
		remaining := maxTrials - len(out)
		batchHint := workers
		if batchHint > remaining {
			batchHint = remaining
		}
		histMu.Lock()
		histCopy := append([]Observation(nil), history...)
		histMu.Unlock()

		batch, ok := suggestor.Suggest(cfg.Space, histCopy, batchHint)
		if !ok || len(batch) == 0 {
			break
		}
		if len(batch) > remaining {
			batch = batch[:remaining]
		}

		type job struct {
			local int
			idx   int
			inds  []int
			cfg   T
		}
		jobs := make([]job, len(batch))
		for i, inds := range batch {
			c, err := cfg.Space.Decode(inds)
			if err != nil {
				return nil, err
			}
			if prepare != nil {
				c = prepare(c)
			}
			jobs[i] = job{local: i, idx: nextIndex + i, inds: inds, cfg: c}
		}
		nextIndex += len(batch)

		results := make([]searchTrial[T], len(jobs))
		var wg sync.WaitGroup
		jobCh := make(chan job, len(jobs))
		for _, j := range jobs {
			jobCh <- j
		}
		close(jobCh)

		wg.Add(workers)
		for w := 0; w < workers; w++ {
			go func() {
				defer wg.Done()
				for j := range jobCh {
					st, _ := evalOne(cfg, 1, trialsDir, j.idx, j.cfg, j.inds)
					results[j.local] = st
					if cfg.Progress {
						status := "ok"
						if st.Trial.Err != nil {
							status = st.Trial.Err.Error()
						}
						outmgr.Printf("sweep: trial %d/%d done in %s: %s\n",
							len(out)+j.local+1, maxTrials, st.Trial.Duration.Round(time.Millisecond), status)
					}
				}
			}()
		}
		wg.Wait()

		for _, st := range results {
			out = append(out, st)
			m, okm := metricForSuggestor(st.Trial, sortBy, cfg.Goals)
			histMu.Lock()
			history = append(history, Observation{Indices: st.Indices, Metric: m, OK: okm})
			histMu.Unlock()
			if cfg.FailFast && st.Trial.Err != nil {
				return out, nil
			}
		}
	}
	return out, nil
}

func evalOne[T any](
	cfg SearchConfig[T],
	_ int,
	trialsDir string,
	index int,
	caseCfg T,
	indices []int,
) (searchTrial[T], error) {
	start := time.Now()
	reportDir := filepath.Join(trialsDir, fmt.Sprintf("trial_%03d", index))
	trial := Trial[T]{
		Index:      index,
		Params:     caseCfg,
		ReportDir:  reportDir,
		ReportHTML: "viewer.html?trial=" + fmt.Sprintf("%03d", index),
	}
	data, err := cfg.Data.resolve(caseCfg)
	if err != nil {
		trial.Err = err
		trial.Duration = time.Since(start)
		return searchTrial[T]{Trial: trial, Indices: indices}, nil
	}
	results, err := cfg.RunTrial(caseCfg, data, reportDir)
	trial.Results = results
	trial.Err = err
	trial.Duration = time.Since(start)
	return searchTrial[T]{Trial: trial, Indices: indices}, nil
}

func rankSearchTrials[T any](trials []Trial[T], sortBy string, goals *Goals) {
	if goals == nil || (len(goals.Constraints) == 0 && len(goals.Targets) == 0) {
		Rank(trials, sortBy)
		return
	}
	type scored struct {
		i        int
		feasible bool
		dist     float64
		metric   float64
		metricOK bool
	}
	scores := make([]scored, len(trials))
	rankMetric := sortBy
	useDist := len(goals.Targets) > 0 && goals.RankFeasibleBy == ""
	if goals.RankFeasibleBy != "" {
		rankMetric = goals.RankFeasibleBy
	}
	for i, t := range trials {
		gs := EvaluateGoals(t.Results, goals)
		if t.Err != nil {
			gs.Feasible = false
		}
		mv, mok := metricValue(t, rankMetric)
		if rankMetric == "max_drawdown" || rankMetric == "drawdown" {
			mv = -mv
		}
		scores[i] = scored{i: i, feasible: gs.Feasible, dist: gs.Distance, metric: mv, metricOK: mok}
	}
	sort.SliceStable(scores, func(a, b int) bool {
		sa, sb := scores[a], scores[b]
		if sa.feasible != sb.feasible {
			return sa.feasible
		}
		if !sa.feasible {
			return sa.i < sb.i
		}
		if useDist {
			if sa.dist != sb.dist {
				return sa.dist < sb.dist
			}
		} else {
			if sa.metricOK != sb.metricOK {
				return sa.metricOK
			}
			if sa.metric != sb.metric {
				return sa.metric > sb.metric
			}
		}
		return sa.i < sb.i
	})
	cp := append([]Trial[T](nil), trials...)
	for i, s := range scores {
		trials[i] = cp[s.i]
	}
}
