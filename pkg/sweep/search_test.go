package sweep

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

type tinyCfg struct {
	A int
	B float64
}

func tinySpace() Space[tinyCfg] {
	base := tinyCfg{A: 0, B: 0}
	return SpaceFromAxes(base,
		CategoricalAxis[tinyCfg]{Name: "A", Choices: []Mutator[tinyCfg]{
			func(c *tinyCfg) { c.A = 1 },
			func(c *tinyCfg) { c.A = 2 },
			func(c *tinyCfg) { c.A = 3 },
		}},
		CategoricalAxis[tinyCfg]{Name: "B", Choices: []Mutator[tinyCfg]{
			func(c *tinyCfg) { c.B = 0.1 },
			func(c *tinyCfg) { c.B = 0.2 },
		}},
	)
}

func TestSpaceDecodeSample(t *testing.T) {
	s := tinySpace()
	if s.NumCombinations() != 6 {
		t.Fatalf("combinations: got %d", s.NumCombinations())
	}
	cfg, err := s.Decode([]int{2, 1})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.A != 3 || cfg.B != 0.2 {
		t.Fatalf("decode: %+v", cfg)
	}
	rng := rand.New(rand.NewSource(1))
	_, idx := s.SampleRandom(rng)
	if len(idx) != 2 {
		t.Fatalf("idx len %d", len(idx))
	}
}

func TestEvaluateGoals(t *testing.T) {
	r := &types.Results{SharpeRatio: 1.2, CAGR: 0.15, MaxDrawdown: 0.12}
	g := &Goals{
		Constraints: []Constraint{MaxDrawdownLTE(0.15), MinSharpe(1.0), MinCAGR(0.10)},
		Targets:     []Target{{Metric: "sharpe_ratio", Value: 1.5, OneSided: false}},
	}
	gs := EvaluateGoals(r, g)
	if !gs.Feasible {
		t.Fatal("expected feasible")
	}
	if gs.Distance <= 0 {
		t.Fatalf("expected positive distance to sharpe 1.5, got %v", gs.Distance)
	}
	r.MaxDrawdown = 0.20
	gs = EvaluateGoals(r, g)
	if gs.Feasible {
		t.Fatal("expected infeasible on DD")
	}
}

func TestRunSearchRandom(t *testing.T) {
	table := types.NewTimeseriesTable[core.Candle]([]string{"X"})
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	_ = table.AddRow(ts, map[string]core.Candle{"X": {Time: ts, Close: 100}})

	dir := t.TempDir()
	space := tinySpace()
	res, err := RunSearch(SearchConfig[tinyCfg]{
		Space:     space,
		Suggestor: SuggestRandom[tinyCfg](rand.New(rand.NewSource(42))),
		MaxTrials: 8,
		Workers:   2,
		OutputDir: dir,
		SortBy:    "sharpe_ratio",
		Data:      DataMode[tinyCfg]{Shared: table},
		RunTrial: func(cfg tinyCfg, data *types.TimeseriesTable[core.Candle], reportDir string) (*types.Results, error) {
			// Prefer A=3,B=0.2
			sharpe := float64(cfg.A) + cfg.B
			return &types.Results{
				SharpeRatio: sharpe,
				CAGR:        0.1,
				MaxDrawdown: 0.05,
				TotalTrades: 10,
			}, nil
		},
		Goals: &Goals{
			Constraints:    []Constraint{MaxDrawdownLTE(0.10)},
			RankFeasibleBy: "sharpe_ratio",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Trials) != 8 {
		t.Fatalf("trials %d", len(res.Trials))
	}
	best := res.Trials[0]
	if best.Err != nil || best.Results == nil {
		t.Fatalf("best failed: %+v", best.Err)
	}
	if best.Results.SharpeRatio < 3.0 {
		t.Fatalf("expected high sharpe near 3.2, got %v params=%+v", best.Results.SharpeRatio, best.Params)
	}
}

func TestSuggestTPE(t *testing.T) {
	space := tinySpace()
	sug := SuggestTPE[tinyCfg](rand.New(rand.NewSource(7)))
	sug.NStartup = 5
	hist := []Observation{}
	for i := 0; i < 30; i++ {
		_, idx := space.SampleRandom(rand.New(rand.NewSource(int64(i + 1))))
		cfg, _ := space.Decode(idx)
		m := float64(cfg.A) + cfg.B
		hist = append(hist, Observation{Indices: idx, Metric: m, OK: true})
	}
	batch, ok := sug.Suggest(space, hist, 4)
	if !ok || len(batch) != 4 {
		t.Fatalf("batch %v ok=%v", batch, ok)
	}
	// Should skew toward high A
	high := 0
	for _, inds := range batch {
		cfg, err := space.Decode(inds)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.A >= 2 {
			high++
		}
	}
	if high < 2 {
		t.Fatalf("expected TPE to prefer high A, high=%d batch=%v", high, batch)
	}
}

func TestGoalDistanceOneSided(t *testing.T) {
	t0 := Target{Metric: "sharpe_ratio", Value: 1.0, OneSided: true}
	if e := targetError(t0, 1.5); e != 0 {
		t.Fatalf("one-sided should be 0, got %v", e)
	}
	if e := targetError(t0, 0.5); e <= 0 || math.IsInf(e, 0) {
		t.Fatalf("expected positive error, got %v", e)
	}
}
