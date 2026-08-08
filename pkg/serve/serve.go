// Package serve provides a minimal static file server that decompresses
// results.json.zst on the fly when clients request results.json.
package serve

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/CCAtAlvis/backgommon/pkg/output"
)

// Handler serves files from root. Requests for path/to/results.json also try
// path/to/results.json.zst and stream the decompressed JSON.
func Handler(root string) http.Handler {
	root = filepath.Clean(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		rel := strings.TrimPrefix(r.URL.Path, "/")
		if rel == "" {
			rel = "dashboard.html"
			if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
				rel = "viewer.html"
			}
		}
		// Prevent path escape.
		full := filepath.Join(root, filepath.FromSlash(rel))
		if !strings.HasPrefix(full, root+string(os.PathSeparator)) && full != root {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		if strings.HasSuffix(rel, "results.json") {
			if serveResultsJSON(w, r, full) {
				return
			}
		}

		http.ServeFile(w, r, full)
	})
}

func serveResultsJSON(w http.ResponseWriter, r *http.Request, jsonPath string) bool {
	if st, err := os.Stat(jsonPath); err == nil && !st.IsDir() {
		http.ServeFile(w, r, jsonPath)
		return true
	}
	zstPath := jsonPath + ".zst"
	data, err := output.ReadZSTD(zstPath)
	if err != nil {
		return false
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.Header().Set("Cache-Control", "no-cache")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return true
	}
	_, _ = w.Write(data)
	return true
}

// ListenAndServe starts an HTTP server on addr serving root.
func ListenAndServe(addr, root string) error {
	fmt.Printf("Serving %s at http://%s/\n", root, addr)
	fmt.Printf("  dashboard: http://%s/dashboard.html\n", addr)
	fmt.Printf("  viewer:    http://%s/viewer.html?trial=000\n", addr)
	return http.ListenAndServe(addr, Handler(root))
}
