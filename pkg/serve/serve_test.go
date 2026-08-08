package serve_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/CCAtAlvis/backgommon/pkg/output"
	"github.com/CCAtAlvis/backgommon/pkg/serve"
)

func TestServeResultsJSONFromZstd(t *testing.T) {
	root := t.TempDir()
	trial := filepath.Join(root, "trials", "trial_000")
	if err := os.MkdirAll(trial, 0755); err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"meta":{"initial_capital":1},"metrics":{},"all_orders":[]}`)
	if err := output.WriteZSTD(filepath.Join(trial, "results.json.zst"), payload); err != nil {
		t.Fatal(err)
	}
	if err := output.WriteViewerHTML(filepath.Join(root, "viewer.html")); err != nil {
		t.Fatal(err)
	}

	h := serve.Handler(root)
	req := httptest.NewRequest(http.MethodGet, "/trials/trial_000/results.json", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("content-type %q", ct)
	}
	body, _ := io.ReadAll(rr.Body)
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
}
