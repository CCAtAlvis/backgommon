package daemon

import (
	_ "embed"
	"net/http"
	"strings"
)

//go:embed catalog.html
var catalogHTML string

func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && !strings.HasPrefix(r.URL.Path, "/app") {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(catalogHTML))
}
