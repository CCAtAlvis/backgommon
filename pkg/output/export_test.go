package output_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/output"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

func TestExportRunTo_ZstdCompactNoNestedOrders(t *testing.T) {
	dir := t.TempDir()
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	curve := []types.AccountValue{
		{Time: start, Value: 100000, Cash: 100000, Holdings: []types.HoldingSnapshot{{Instrument: "A", Quantity: 1, Price: 10, AvgEntry: 9}}},
		{Time: start.Add(24 * time.Hour), Value: 101000, Cash: 50000, Holdings: []types.HoldingSnapshot{{Instrument: "A", Quantity: 1, Price: 11, AvgEntry: 9}}},
	}
	results := &types.Results{
		StartTime: start, EndTime: start.Add(24 * time.Hour),
		InitialCapital: 100000, FinalCapital: 101000,
		Returns: 0.01, CAGR: 0.01, SharpeRatio: 1.2,
	}
	err := output.ExportRunTo(dir, output.ReportData{
		Results:     results,
		EquityCurve: curve,
		Settings:    map[string]any{"Lookback": 20},
		Options: output.ExportOptions{
			Compress:      output.CompressZstd,
			OfflineDataJS: false,
			WriteViewer:   false,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	zst := filepath.Join(dir, "results.json.zst")
	if _, err := os.Stat(zst); err != nil {
		t.Fatalf("expected results.json.zst: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "results.json")); !os.IsNotExist(err) {
		t.Fatal("should not write plain results.json when using zstd")
	}
	if _, err := os.Stat(filepath.Join(dir, "data.js")); !os.IsNotExist(err) {
		t.Fatal("should not write data.js when OfflineDataJS=false")
	}

	raw, err := output.ReadZSTD(zst)
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]any
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	trades, _ := report["all_trades"].([]any)
	if trades == nil {
		// empty trades ok
	} else {
		for _, tr := range trades {
			m := tr.(map[string]any)
			if _, ok := m["orders"]; ok {
				t.Fatal("all_trades must not nest orders")
			}
		}
	}
	if _, ok := report["all_orders"]; !ok {
		t.Fatal("expected all_orders key")
	}
	cfg, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg) == 0 || cfg[0] != '{' {
		t.Fatalf("config.json should be compact JSON, got %q", cfg)
	}
}

func TestExportRunTo_OfflineDataJS(t *testing.T) {
	dir := t.TempDir()
	results := &types.Results{InitialCapital: 1, FinalCapital: 1}
	err := output.ExportRunTo(dir, output.ReportData{
		Results:     results,
		EquityCurve: nil,
		Options: output.ExportOptions{
			Compress:      output.CompressZstd,
			OfflineDataJS: true,
			WriteViewer:   true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"results.json.zst", "data.js", "report.html", "viewer.html"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	js, err := os.ReadFile(filepath.Join(dir, "data.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(js), "window.__REPORT__=") {
		t.Fatalf("data.js prefix: %q", js[:minInt(40, len(js))])
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
