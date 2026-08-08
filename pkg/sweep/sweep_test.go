package sweep_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/sweep"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

type testCfg struct {
	A int
	B float64
}

func TestExpandCartesian(t *testing.T) {
	base := testCfg{A: 0, B: 0}
	cases := sweep.Expand(base,
		[]sweep.Mutator[testCfg]{
			func(c *testCfg) { c.A = 1 },
			func(c *testCfg) { c.A = 2 },
			func(c *testCfg) { c.A = 3 },
		},
		[]sweep.Mutator[testCfg]{
			func(c *testCfg) { c.B = 0.1 },
			func(c *testCfg) { c.B = 0.2 },
		},
	)
	if len(cases) != 6 {
		t.Fatalf("got %d cases, want 6", len(cases))
	}
	seen := map[string]bool{}
	for _, c := range cases {
		key := filepath.Join(
			string(rune('0'+c.A)),
			"",
		)
		_ = key
		seen[string(rune('0'+byte(c.A)))+":"+formatF(c.B)] = true
	}
	for _, a := range []int{1, 2, 3} {
		for _, b := range []float64{0.1, 0.2} {
			k := string(rune('0'+byte(a))) + ":" + formatF(b)
			if !seen[k] {
				t.Fatalf("missing combo A=%d B=%v", a, b)
			}
		}
	}
}

func formatF(v float64) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestExpandMultiFieldMutator(t *testing.T) {
	base := testCfg{}
	cases := sweep.Expand(base,
		[]sweep.Mutator[testCfg]{
			func(c *testCfg) { c.A = 1; c.B = 0.5 },
			func(c *testCfg) { c.A = 2; c.B = 0.9 },
		},
	)
	if len(cases) != 2 {
		t.Fatalf("got %d", len(cases))
	}
	if cases[0].A != 1 || cases[0].B != 0.5 {
		t.Fatalf("first: %+v", cases[0])
	}
	if cases[1].A != 2 || cases[1].B != 0.9 {
		t.Fatalf("second: %+v", cases[1])
	}
}

func TestExpandNoAxes(t *testing.T) {
	base := testCfg{A: 7}
	cases := sweep.Expand(base)
	if len(cases) != 1 || cases[0].A != 7 {
		t.Fatalf("got %+v", cases)
	}
}

func TestDataModeValidation(t *testing.T) {
	table := types.NewTimeseriesTable[core.Candle]([]string{"X"})
	_, err := sweep.Run(sweep.Config[testCfg]{
		Cases:     []testCfg{{}},
		OutputDir: t.TempDir(),
		Data:      sweep.DataMode[testCfg]{}, // none set
		RunTrial: func(cfg testCfg, data *types.TimeseriesTable[core.Candle], reportDir string) (*types.Results, error) {
			return &types.Results{}, nil
		},
	})
	if err == nil {
		t.Fatal("expected error when no data mode set")
	}

	_, err = sweep.Run(sweep.Config[testCfg]{
		Cases:     []testCfg{{}},
		OutputDir: t.TempDir(),
		Data: sweep.DataMode[testCfg]{
			Shared: table,
			Clone:  table,
		},
		RunTrial: func(cfg testCfg, data *types.TimeseriesTable[core.Candle], reportDir string) (*types.Results, error) {
			return &types.Results{}, nil
		},
	})
	if err == nil {
		t.Fatal("expected error when multiple data modes set")
	}
}

func TestRankSharpeAndDrawdown(t *testing.T) {
	trials := []sweep.Trial[testCfg]{
		{Index: 0, Results: &types.Results{SharpeRatio: 0.5, MaxDrawdown: 0.3}},
		{Index: 1, Results: &types.Results{SharpeRatio: 1.2, MaxDrawdown: 0.1}},
		{Index: 2, Err: os.ErrInvalid},
		{Index: 3, Results: &types.Results{SharpeRatio: 0.9, MaxDrawdown: 0.2}},
	}
	sweep.Rank(trials, "sharpe_ratio")
	if trials[0].Index != 1 || trials[1].Index != 3 || trials[2].Index != 0 || trials[3].Index != 2 {
		t.Fatalf("sharpe order: %+v %+v %+v %+v", trials[0].Index, trials[1].Index, trials[2].Index, trials[3].Index)
	}

	trials = []sweep.Trial[testCfg]{
		{Index: 0, Results: &types.Results{MaxDrawdown: 0.3}},
		{Index: 1, Results: &types.Results{MaxDrawdown: 0.1}},
		{Index: 2, Results: &types.Results{MaxDrawdown: 0.2}},
	}
	sweep.Rank(trials, "max_drawdown")
	if trials[0].Index != 1 || trials[1].Index != 2 || trials[2].Index != 0 {
		t.Fatalf("drawdown order wrong")
	}
}

func TestRunSharedWritesSummaryAndReports(t *testing.T) {
	table := types.NewTimeseriesTable[core.Candle]([]string{"X"})
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	_ = table.AddRow(ts, map[string]core.Candle{"X": {Time: ts, Close: 100}})

	out := t.TempDir()
	cases := sweep.Expand(testCfg{},
		[]sweep.Mutator[testCfg]{
			func(c *testCfg) { c.A = 1 },
			func(c *testCfg) { c.A = 2 },
		},
	)

	result, err := sweep.Run(sweep.Config[testCfg]{
		Cases:     cases,
		Workers:   2,
		OutputDir: out,
		SortBy:    "sharpe_ratio",
		Progress:  false,
		Data:      sweep.DataMode[testCfg]{Shared: table},
		RunTrial: func(cfg testCfg, data *types.TimeseriesTable[core.Candle], reportDir string) (*types.Results, error) {
			if data != table {
				t.Errorf("Shared mode should pass same table pointer")
			}
			if err := os.MkdirAll(reportDir, 0755); err != nil {
				return nil, err
			}
			// Simulate runner.WithReportDir artifacts (compressed results)
			_ = os.WriteFile(filepath.Join(reportDir, "results.json.zst"), []byte("x"), 0644)
			return &types.Results{
				SharpeRatio: float64(cfg.A),
				Returns:     float64(cfg.A) * 0.1,
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.SweepDir == "" {
		t.Fatal("missing SweepDir")
	}
	if _, err := os.Stat(result.SummaryJSON); err != nil {
		t.Fatalf("summary.json: %v", err)
	}
	if _, err := os.Stat(result.SummaryCSV); err != nil {
		t.Fatalf("summary.csv: %v", err)
	}
	if len(result.Trials) != 2 {
		t.Fatalf("trials: %d", len(result.Trials))
	}
	// Ranked by sharpe: A=2 first
	if result.Trials[0].Params.A != 2 {
		t.Fatalf("expected best A=2 first, got %+v", result.Trials[0].Params)
	}
	if result.Trials[0].ReportHTML != "viewer.html?trial=001" {
		t.Fatalf("expected viewer.html?trial=001 for best trial (index 1), got %q", result.Trials[0].ReportHTML)
	}
	viewer := filepath.Join(result.SweepDir, "viewer.html")
	if _, err := os.Stat(viewer); err != nil {
		t.Fatalf("shared viewer.html missing: %v", err)
	}
	if _, err := os.Stat(result.DashboardHTML); err != nil {
		t.Fatalf("dashboard.html missing: %v", err)
	}
}

func TestRunCloneGivesDistinctTables(t *testing.T) {
	table := types.NewTimeseriesTable[core.Candle]([]string{"X"})
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	c := core.Candle{Time: ts, Close: 100}
	c.SetIndicator("seed", 1.0)
	_ = table.AddRow(ts, map[string]core.Candle{"X": c})

	seen := make(chan *types.TimeseriesTable[core.Candle], 2)
	_, err := sweep.Run(sweep.Config[testCfg]{
		Cases:     []testCfg{{A: 1}, {A: 2}},
		Workers:   1,
		OutputDir: t.TempDir(),
		Data:      sweep.DataMode[testCfg]{Clone: table},
		RunTrial: func(cfg testCfg, data *types.TimeseriesTable[core.Candle], reportDir string) (*types.Results, error) {
			_ = os.MkdirAll(reportDir, 0755)
			seen <- data
			if data == table {
				t.Error("Clone mode must not reuse template pointer")
			}
			candle, _ := data.GetValue(ts, "X")
			candle.SetIndicator("mutated", float64(cfg.A))
			_ = data.SetValue(ts, "X", candle)
			return &types.Results{SharpeRatio: 1}, nil
		},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	close(seen)
	orig, _ := table.GetValue(ts, "X")
	if orig.HasIndicator("mutated") {
		t.Fatal("clone mutation leaked into template")
	}
}
