package sweep_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/sweep"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

type cfg2 struct {
	A, B int
}

func TestRunStaged(t *testing.T) {
	table := types.NewTimeseriesTable[core.Candle]([]string{"X"})
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	_ = table.AddRow(ts, map[string]core.Candle{"X": {Time: ts, Close: 1}})

	base := cfg2{}
	axA := sweep.CategoricalAxis[cfg2]{Name: "A", Choices: []sweep.Mutator[cfg2]{
		func(c *cfg2) { c.A = 1 },
		func(c *cfg2) { c.A = 5 },
	}}
	axB := sweep.CategoricalAxis[cfg2]{Name: "B", Choices: []sweep.Mutator[cfg2]{
		func(c *cfg2) { c.B = 1 },
		func(c *cfg2) { c.B = 5 },
	}}

	res, err := sweep.RunStaged(sweep.StagedConfig[cfg2]{
		Base: base,
		Stages: []sweep.Stage[cfg2]{
			{Name: "A", Space: sweep.FreezeBase(base, axA), MaxTrials: 6, Suggestor: sweep.SuggestRandom[cfg2](rand.New(rand.NewSource(1)))},
			{Name: "B", Space: sweep.FreezeBase(base, axB), MaxTrials: 6, Suggestor: sweep.SuggestRandom[cfg2](rand.New(rand.NewSource(2)))},
		},
		Workers:   2,
		OutputDir: t.TempDir(),
		SortBy:    "sharpe_ratio",
		Data:      sweep.DataMode[cfg2]{Shared: table},
		RunTrial: func(c cfg2, data *types.TimeseriesTable[core.Candle], reportDir string) (*types.Results, error) {
			return &types.Results{SharpeRatio: float64(c.A + c.B), MaxDrawdown: 0.01}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Trials[0].Params.A != 5 {
		t.Fatalf("expected stage A to freeze A=5, got %+v", res.Trials[0].Params)
	}
	if res.Trials[0].Params.B != 1 && res.Trials[0].Params.B != 5 {
		t.Fatalf("expected B in {1,5}, got %+v", res.Trials[0].Params)
	}
	// With enough trials, B=5 (higher sharpe) should win.
	res2, err := sweep.RunStaged(sweep.StagedConfig[cfg2]{
		Base: base,
		Stages: []sweep.Stage[cfg2]{
			{Name: "A", Space: sweep.FreezeBase(base, axA), MaxTrials: 20, Suggestor: sweep.SuggestRandom[cfg2](rand.New(rand.NewSource(1)))},
			{Name: "B", Space: sweep.FreezeBase(base, axB), MaxTrials: 20, Suggestor: sweep.SuggestRandom[cfg2](rand.New(rand.NewSource(2)))},
		},
		Workers:   2,
		OutputDir: t.TempDir(),
		SortBy:    "sharpe_ratio",
		Data:      sweep.DataMode[cfg2]{Shared: table},
		RunTrial: func(c cfg2, data *types.TimeseriesTable[core.Candle], reportDir string) (*types.Results, error) {
			return &types.Results{SharpeRatio: float64(c.A + c.B), MaxDrawdown: 0.01}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res2.Trials[0].Params.A != 5 || res2.Trials[0].Params.B != 5 {
		t.Fatalf("expected {5,5}, got %+v", res2.Trials[0].Params)
	}
}
