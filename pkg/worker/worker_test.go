package worker_test

import (
	"os"
	"testing"

	"github.com/CCAtAlvis/backgommon/pkg/worker"
)

func TestClientDescribeSMA(t *testing.T) {
	sma := os.Getenv("BACKGOMMON_TEST_WORKER")
	if sma == "" {
		sma = "/tmp/sma_crossover"
	}
	if _, err := os.Stat(sma); err != nil {
		t.Skip("worker binary not found; build examples/strategies/sma_crossover first")
	}
	c := &worker.Client{Binary: sma}
	d, err := c.Describe()
	if err != nil {
		t.Fatal(err)
	}
	if d.Name != "sma_crossover" {
		t.Fatalf("name %q", d.Name)
	}
	if d.Version == "" {
		t.Fatal("expected version")
	}
}
