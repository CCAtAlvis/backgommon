package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/CCAtAlvis/backgommon/pkg/catalog"
	"github.com/CCAtAlvis/backgommon/pkg/jobs"
	"github.com/CCAtAlvis/backgommon/pkg/serve"
)

// Server is the catalog-mode HTTP server.
type Server struct {
	Cat    *catalog.Catalog
	Addr   string
	Jobs   *jobs.Runner
	mu     sync.Mutex
	procs  map[int64]*os.Process
	cancel map[int64]bool
}

// New creates a daemon server.
func New(cat *catalog.Catalog, addr string) *Server {
	s := &Server{
		Cat:    cat,
		Addr:   addr,
		procs:  map[int64]*os.Process{},
		cancel: map[int64]bool{},
	}
	s.Jobs = &jobs.Runner{
		Cat: cat,
		OnStart: func(jobID int64, p *os.Process) {
			s.mu.Lock()
			s.procs[jobID] = p
			s.mu.Unlock()
		},
		OnEnd: func(jobID int64) {
			s.mu.Lock()
			delete(s.procs, jobID)
			s.mu.Unlock()
		},
		Canceled: func(jobID int64) bool {
			s.mu.Lock()
			defer s.mu.Unlock()
			return s.cancel[jobID]
		},
	}
	return s
}

// Handler returns the root mux.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/strategies", s.handleStrategies)
	mux.HandleFunc("/api/strategies/", s.handleStrategy)
	mux.HandleFunc("/api/register", s.handleRegister)
	mux.HandleFunc("/api/unregister", s.handleUnregister)
	mux.HandleFunc("/api/jobs", s.handleJobs)
	mux.HandleFunc("/api/jobs/", s.handleJob)
	mux.HandleFunc("/api/run", s.handleRun)
	mux.HandleFunc("/api/sweep", s.handleSweep)
	mux.HandleFunc("/api/ingest", s.handleIngest)
	mux.HandleFunc("/api/artifacts", s.handleArtifacts)
	mux.HandleFunc("/api/artifacts/", s.handleArtifact)
	mux.Handle("/artifacts/", http.StripPrefix("/artifacts/", http.HandlerFunc(s.handleArtifactFiles)))
	mux.HandleFunc("/", s.handleUI)
	return mux
}

// ListenAndServe starts the daemon.
func (s *Server) ListenAndServe() error {
	fmt.Printf("backgommon daemon at http://%s/\n", s.Addr)
	fmt.Printf("  data dir: %s\n", s.Cat.DataDir)
	fmt.Printf("  catalog:  http://%s/\n", s.Addr)
	return http.ListenAndServe(s.Addr, s.Handler())
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"ok": true, "data_dir": s.Cat.DataDir})
}

func (s *Server) handleStrategies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	list, err := s.Cat.ListStrategies()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if list == nil {
		list = []catalog.Strategy{}
	}
	writeJSON(w, list)
}

func (s *Server) handleStrategy(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/strategies/")
	name = strings.Trim(name, "/")
	if name == "" {
		http.NotFound(w, r)
		return
	}
	st, err := s.Cat.GetStrategy(name)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}
	writeJSON(w, st)
}

type registerReq struct {
	Path string `json:"path"`
	Name string `json:"name,omitempty"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	res, err := jobs.Register(s.Cat, req.Path, req.Name)
	if err != nil {
		if nc, ok := err.(*catalog.ErrNameConflict); ok {
			w.WriteHeader(http.StatusConflict)
			writeJSON(w, map[string]any{"error": nc.Error(), "existing_path": nc.ExistingPath, "new_path": nc.NewPath})
			return
		}
		http.Error(w, err.Error(), 400)
		return
	}
	writeJSON(w, res)
}

type nameReq struct {
	Name string `json:"name"`
}

func (s *Server) handleUnregister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req nameReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err := s.Cat.Unregister(req.Name); err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	list, err := s.Cat.ListJobs(100)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if list == nil {
		list = []catalog.Job{}
	}
	writeJSON(w, list)
}

func (s *Server) handleJob(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/jobs/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	if len(parts) == 2 && parts[1] == "cancel" && r.Method == http.MethodPost {
		s.cancelJob(w, id)
		return
	}
	j, err := s.Cat.GetJob(id)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}
	writeJSON(w, j)
}

func (s *Server) cancelJob(w http.ResponseWriter, id int64) {
	s.mu.Lock()
	s.cancel[id] = true
	p := s.procs[id]
	s.mu.Unlock()
	if p != nil {
		_ = p.Kill()
	}
	_ = s.Cat.UpdateJobStatus(id, catalog.JobCanceled, map[string]any{"error": "canceled"})
	writeJSON(w, map[string]any{"ok": true})
}

type runReq struct {
	Strategy string          `json:"strategy"`
	Config   json.RawMessage `json:"config"`
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req runReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	st, err := s.Cat.GetStrategy(req.Strategy)
	if err != nil {
		http.Error(w, "strategy not found", 404)
		return
	}
	if req.Config == nil {
		req.Config = json.RawMessage(`{}`)
	}
	cfgBytes, _ := json.Marshal(map[string]any{"config": json.RawMessage(req.Config)})
	jobID, err := s.Cat.CreateJob(catalog.JobRun, st.Name, st.Version, st.AbsPath, string(cfgBytes))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	go s.Jobs.ExecuteRun(jobID, st, req.Config)
	writeJSON(w, map[string]any{"job_id": jobID})
}

type sweepAPIReq struct {
	Strategy string          `json:"strategy"`
	Base     json.RawMessage `json:"base"`
	Cases    json.RawMessage `json:"cases,omitempty"`
	Axes     json.RawMessage `json:"axes,omitempty"`
	Workers  int             `json:"workers,omitempty"`
	SortBy   string          `json:"sort_by,omitempty"`
}

func (s *Server) handleSweep(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req sweepAPIReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	st, err := s.Cat.GetStrategy(req.Strategy)
	if err != nil {
		http.Error(w, "strategy not found", 404)
		return
	}
	cfgBytes, _ := json.Marshal(req)
	jobID, err := s.Cat.CreateJob(catalog.JobSweep, st.Name, st.Version, st.AbsPath, string(cfgBytes))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	go s.Jobs.ExecuteSweep(jobID, st, req.Base, req.Cases, req.Axes, req.Workers, req.SortBy)
	writeJSON(w, map[string]any{"job_id": jobID})
}

type ingestReq struct {
	Path         string `json:"path"`
	StrategyName string `json:"strategy_name,omitempty"`
}

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req ingestReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	art, err := s.Cat.IngestDir(req.Path, req.StrategyName, 0)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	writeJSON(w, art)
}

func (s *Server) handleArtifacts(w http.ResponseWriter, r *http.Request) {
	list, err := s.Cat.ListArtifacts(100)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if list == nil {
		list = []catalog.Artifact{}
	}
	writeJSON(w, list)
}

func (s *Server) handleArtifact(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/artifacts/")
	idStr = strings.Trim(idStr, "/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	a, err := s.Cat.GetArtifact(id)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}
	writeJSON(w, a)
}

func (s *Server) handleArtifactFiles(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/"), "/", 2)
	if len(parts) < 1 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	a, err := s.Cat.GetArtifact(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	rel := ""
	if len(parts) == 2 {
		rel = parts[1]
	}
	h := serve.Handler(a.Path)
	r2 := r.Clone(r.Context())
	r2.URL.Path = "/" + rel
	h.ServeHTTP(w, r2)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
