package sweep_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/sweep"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

func TestDeriveAxesFromGrid(t *testing.T) {
	type cfg struct {
		EntryMomentum   float64
		EntryBand       float64
		PersistenceDays int
		Fixed           string
	}
	base := cfg{Fixed: "same"}
	cases := sweep.Expand(base,
		[]sweep.Mutator[cfg]{
			func(c *cfg) { c.EntryMomentum = 0.10 },
			func(c *cfg) { c.EntryMomentum = 0.15 },
			func(c *cfg) { c.EntryMomentum = 0.20 },
		},
		[]sweep.Mutator[cfg]{
			func(c *cfg) { c.EntryBand = 0.03 },
			func(c *cfg) { c.EntryBand = 0.05 },
		},
	)
	if len(cases) != 6 {
		t.Fatalf("cases=%d", len(cases))
	}

	table := types.NewTimeseriesTable[core.Candle]([]string{"X"})
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	_ = table.AddRow(ts, map[string]core.Candle{"X": {Time: ts, Close: 1}})

	out := t.TempDir()
	result, err := sweep.Run(sweep.Config[cfg]{
		Cases:     cases,
		Workers:   1,
		OutputDir: out,
		SortBy:    "sharpe_ratio",
		Data:      sweep.DataMode[cfg]{Shared: table},
		RunTrial: func(c cfg, data *types.TimeseriesTable[core.Candle], reportDir string) (*types.Results, error) {
			_ = os.MkdirAll(reportDir, 0755)
			_ = os.WriteFile(filepath.Join(reportDir, "report.html"), []byte("<html></html>"), 0644)
			return &types.Results{SharpeRatio: c.EntryMomentum + c.EntryBand}, nil
		},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	raw, err := os.ReadFile(result.SummaryJSON)
	if err != nil {
		t.Fatal(err)
	}
	var summary struct {
		Axes []struct {
			Name   string        `json:"name"`
			Values []interface{} `json:"values"`
		} `json:"axes"`
		MetricKeys []string `json:"metric_keys"`
	}
	if err := json.Unmarshal(raw, &summary); err != nil {
		t.Fatal(err)
	}
	if len(summary.Axes) != 2 {
		t.Fatalf("axes=%v want 2 (Fixed should be excluded)", summary.Axes)
	}
	names := map[string]int{}
	for _, a := range summary.Axes {
		names[a.Name] = len(a.Values)
	}
	if names["EntryMomentum"] != 3 || names["EntryBand"] != 2 {
		t.Fatalf("axis values: %+v", names)
	}
	if names["Fixed"] != 0 && names["PersistenceDays"] != 0 {
		// PersistenceDays was never set (0 for all) — not an axis
	}
	if _, ok := names["Fixed"]; ok {
		t.Fatal("Fixed should not be an axis")
	}

	dash, err := os.ReadFile(result.DashboardHTML)
	if err != nil {
		t.Fatal(err)
	}
	html := string(dash)
	if !strings.Contains(html, "Sweep Dashboard") {
		t.Fatal("dashboard missing title")
	}
	if !strings.Contains(html, "EntryMomentum") {
		t.Fatal("dashboard missing axis name")
	}
	if !strings.Contains(html, `"index":0`) && !strings.Contains(html, `"index": 0`) {
		// compact JSON from marshal has no space
		if !strings.Contains(html, `"index":0`) {
			t.Fatal("dashboard missing trial data")
		}
	}
	if !strings.Contains(html, "Axis Analysis") || !strings.Contains(html, "Compare") {
		t.Fatal("dashboard missing view tabs")
	}
}

func TestDashboardWrittenOnRun(t *testing.T) {
	table := types.NewTimeseriesTable[core.Candle]([]string{"X"})
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	_ = table.AddRow(ts, map[string]core.Candle{"X": {Time: ts, Close: 1}})

	result, err := sweep.Run(sweep.Config[testCfg]{
		Cases:     []testCfg{{A: 1}, {A: 2}},
		Workers:   1,
		OutputDir: t.TempDir(),
		Data:      sweep.DataMode[testCfg]{Shared: table},
		RunTrial: func(cfg testCfg, data *types.TimeseriesTable[core.Candle], reportDir string) (*types.Results, error) {
			_ = os.MkdirAll(reportDir, 0755)
			return &types.Results{SharpeRatio: float64(cfg.A)}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.DashboardHTML == "" {
		t.Fatal("DashboardHTML empty")
	}
	st, err := os.Stat(result.DashboardHTML)
	if err != nil || st.Size() < 100 {
		t.Fatalf("dashboard missing or tiny: %v size=%v", err, st)
	}
}
