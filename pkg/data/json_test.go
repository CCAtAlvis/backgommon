package data

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCandlesFromJSON(t *testing.T) {
	path := filepath.Join("testdata", "sample.json")
	raw := `{"symbol":"SYM:TEST-EQ","candles":[{"date":"2024-01-02","open":100,"high":105,"low":99,"close":102,"volume":1000},{"date":"2024-01-03","open":102,"high":108,"low":101,"close":107,"volume":1200}]}`
	if err := os.MkdirAll("testdata", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	symbol, candles, err := LoadCandlesFromJSON(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if symbol != "SYM:TEST-EQ" {
		t.Fatalf("symbol: got %q", symbol)
	}
	if len(candles) != 2 || candles[1].Close != 107 {
		t.Fatalf("unexpected candles: len=%d close=%.2f", len(candles), candles[1].Close)
	}
}
