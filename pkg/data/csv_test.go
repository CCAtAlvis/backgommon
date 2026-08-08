package data

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCandlesFromCSV(t *testing.T) {
	path := filepath.Join("testdata", "sample.csv")
	content := "date,open,high,low,close,volume\n2024-01-02,100,105,99,102,1000\n2024-01-03,102,108,101,107,1200\n"
	if err := os.MkdirAll("testdata", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	candles, err := LoadCandlesFromCSV(path, "TEST")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(candles) != 2 || candles[1].Close != 107 {
		t.Fatalf("unexpected candles")
	}
}
