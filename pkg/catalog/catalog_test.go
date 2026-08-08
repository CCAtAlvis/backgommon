package catalog_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/CCAtAlvis/backgommon/pkg/catalog"
)

func TestRegisterPathConflict(t *testing.T) {
	dir := t.TempDir()
	c, err := catalog.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	schema := json.RawMessage(`{}`)
	desc := json.RawMessage(`{"name":"s"}`)
	bin1 := filepath.Join(dir, "bin1")
	bin2 := filepath.Join(dir, "bin2")
	if err := os.WriteFile(bin1, []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin2, []byte("y"), 0755); err != nil {
		t.Fatal(err)
	}

	if _, err := c.Register("sma", bin1, "0.1.0", schema, desc); err != nil {
		t.Fatal(err)
	}
	// same path refresh
	res, err := c.Register("sma", bin1, "0.2.0", schema, desc)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Updated || res.Strategy.Version != "0.2.0" {
		t.Fatalf("expected update: %+v", res)
	}
	// different path conflict
	_, err = c.Register("sma", bin2, "0.1.0", schema, desc)
	if err == nil {
		t.Fatal("expected conflict")
	}
	if _, ok := err.(*catalog.ErrNameConflict); !ok {
		t.Fatalf("want ErrNameConflict, got %T %v", err, err)
	}
	// override name ok
	if _, err := c.Register("sma_other", bin2, "0.1.0", schema, desc); err != nil {
		t.Fatal(err)
	}
}

func TestIngestSweep(t *testing.T) {
	dir := t.TempDir()
	c, err := catalog.Open(filepath.Join(dir, "data"))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	sweepDir := filepath.Join(dir, "sweep_x")
	if err := os.MkdirAll(sweepDir, 0755); err != nil {
		t.Fatal(err)
	}
	summary := `{
  "sweep_dir": "` + sweepDir + `",
  "sort_by": "sharpe_ratio",
  "axes": [],
  "metric_keys": ["sharpe_ratio"],
  "trials": [
    {"index": 0, "params": {"short_window": 5}, "metrics": {"sharpe_ratio": 1.2}, "duration_ms": 10, "report_html": "viewer.html?trial=000"}
  ]
}`
	if err := os.WriteFile(filepath.Join(sweepDir, "summary.json"), []byte(summary), 0644); err != nil {
		t.Fatal(err)
	}
	art, err := c.IngestDir(sweepDir, "sma_crossover", 0)
	if err != nil {
		t.Fatal(err)
	}
	if art.Kind != catalog.ArtifactSweep {
		t.Fatalf("kind %s", art.Kind)
	}
	if len(art.Trials) != 1 {
		t.Fatalf("trials %d", len(art.Trials))
	}
}
