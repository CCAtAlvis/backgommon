// Package catalog is the SQLite-backed registry for strategies, jobs, and artifacts.
package catalog

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Catalog stores strategies, jobs, and ingested run/sweep artifacts.
type Catalog struct {
	db      *sql.DB
	DataDir string
}

// Open opens (or creates) catalog.db under dataDir.
func Open(dataDir string) (*Catalog, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir data dir: %w", err)
	}
	runs := filepath.Join(dataDir, "runs")
	if err := os.MkdirAll(runs, 0755); err != nil {
		return nil, fmt.Errorf("mkdir runs: %w", err)
	}
	dbPath := filepath.Join(dataDir, "catalog.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	c := &Catalog{db: db, DataDir: dataDir}
	if err := c.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return c, nil
}

func (c *Catalog) Close() error {
	if c.db == nil {
		return nil
	}
	return c.db.Close()
}

func (c *Catalog) migrate() error {
	_, err := c.db.Exec(`
CREATE TABLE IF NOT EXISTS strategies (
  name TEXT PRIMARY KEY,
  abs_path TEXT NOT NULL,
  version TEXT NOT NULL DEFAULT '',
  schema_json TEXT NOT NULL DEFAULT '{}',
  describe_json TEXT NOT NULL DEFAULT '{}',
  registered_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS jobs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  type TEXT NOT NULL,
  strategy_name TEXT NOT NULL,
  strategy_version TEXT NOT NULL DEFAULT '',
  binary_path TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  config_json TEXT NOT NULL DEFAULT '{}',
  progress TEXT NOT NULL DEFAULT '',
  report_dir TEXT NOT NULL DEFAULT '',
  error TEXT NOT NULL DEFAULT '',
  pid INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  started_at TEXT NOT NULL DEFAULT '',
  finished_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS artifacts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  kind TEXT NOT NULL,
  strategy_name TEXT NOT NULL DEFAULT '',
  job_id INTEGER NOT NULL DEFAULT 0,
  path TEXT NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  metrics_json TEXT NOT NULL DEFAULT '{}',
  summary_json TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS trials (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  artifact_id INTEGER NOT NULL,
  trial_index INTEGER NOT NULL,
  params_json TEXT NOT NULL DEFAULT '{}',
  metrics_json TEXT NOT NULL DEFAULT '{}',
  report_html TEXT NOT NULL DEFAULT '',
  error TEXT NOT NULL DEFAULT '',
  FOREIGN KEY(artifact_id) REFERENCES artifacts(id)
);
CREATE INDEX IF NOT EXISTS idx_jobs_created ON jobs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_artifacts_created ON artifacts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_trials_artifact ON trials(artifact_id);
`)
	return err
}

func now() string { return time.Now().UTC().Format(time.RFC3339Nano) }

// Strategy is a registered worker binary.
type Strategy struct {
	Name         string          `json:"name"`
	AbsPath      string          `json:"abs_path"`
	Version      string          `json:"version"`
	SchemaJSON   json.RawMessage `json:"schema"`
	DescribeJSON json.RawMessage `json:"describe"`
	RegisteredAt string          `json:"registered_at"`
	UpdatedAt    string          `json:"updated_at"`
}

// RegisterResult is returned from Register.
type RegisterResult struct {
	Strategy Strategy `json:"strategy"`
	Updated  bool     `json:"updated"`
}

// ErrNameConflict is returned when a name is owned by a different binary path.
type ErrNameConflict struct {
	Name        string
	ExistingPath string
	NewPath     string
}

func (e *ErrNameConflict) Error() string {
	return fmt.Sprintf("strategy %q already registered at %s (tried %s); use --name to override or unregister first",
		e.Name, e.ExistingPath, e.NewPath)
}

// Register upserts a strategy. Same name + same abs path refreshes metadata.
// Same name + different path returns *ErrNameConflict.
func (c *Catalog) Register(name, absPath, version string, schema, describe json.RawMessage) (*RegisterResult, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("strategy name is required")
	}
	absPath, err := filepath.Abs(absPath)
	if err != nil {
		return nil, err
	}
	if schema == nil {
		schema = json.RawMessage(`{}`)
	}
	if describe == nil {
		describe = json.RawMessage(`{}`)
	}
	if version == "" {
		version = "0.0.0"
	}

	existing, err := c.GetStrategy(name)
	if err == nil {
		if existing.AbsPath != absPath {
			return nil, &ErrNameConflict{Name: name, ExistingPath: existing.AbsPath, NewPath: absPath}
		}
		ts := now()
		_, err = c.db.Exec(`UPDATE strategies SET version=?, schema_json=?, describe_json=?, updated_at=? WHERE name=?`,
			version, string(schema), string(describe), ts, name)
		if err != nil {
			return nil, err
		}
		existing.Version = version
		existing.SchemaJSON = schema
		existing.DescribeJSON = describe
		existing.UpdatedAt = ts
		return &RegisterResult{Strategy: *existing, Updated: true}, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	ts := now()
	_, err = c.db.Exec(`INSERT INTO strategies(name, abs_path, version, schema_json, describe_json, registered_at, updated_at)
		VALUES(?,?,?,?,?,?,?)`, name, absPath, version, string(schema), string(describe), ts, ts)
	if err != nil {
		return nil, err
	}
	return &RegisterResult{Strategy: Strategy{
		Name: name, AbsPath: absPath, Version: version,
		SchemaJSON: schema, DescribeJSON: describe,
		RegisteredAt: ts, UpdatedAt: ts,
	}}, nil
}

// Unregister removes a strategy by name.
func (c *Catalog) Unregister(name string) error {
	res, err := c.db.Exec(`DELETE FROM strategies WHERE name=?`, name)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("strategy %q not found", name)
	}
	return nil
}

// GetStrategy returns one strategy or sql.ErrNoRows.
func (c *Catalog) GetStrategy(name string) (*Strategy, error) {
	row := c.db.QueryRow(`SELECT name, abs_path, version, schema_json, describe_json, registered_at, updated_at FROM strategies WHERE name=?`, name)
	var s Strategy
	var schema, describe string
	if err := row.Scan(&s.Name, &s.AbsPath, &s.Version, &schema, &describe, &s.RegisteredAt, &s.UpdatedAt); err != nil {
		return nil, err
	}
	s.SchemaJSON = json.RawMessage(schema)
	s.DescribeJSON = json.RawMessage(describe)
	return &s, nil
}

// ListStrategies returns all registered strategies ordered by name.
func (c *Catalog) ListStrategies() ([]Strategy, error) {
	rows, err := c.db.Query(`SELECT name, abs_path, version, schema_json, describe_json, registered_at, updated_at FROM strategies ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Strategy
	for rows.Next() {
		var s Strategy
		var schema, describe string
		if err := rows.Scan(&s.Name, &s.AbsPath, &s.Version, &schema, &describe, &s.RegisteredAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		s.SchemaJSON = json.RawMessage(schema)
		s.DescribeJSON = json.RawMessage(describe)
		out = append(out, s)
	}
	return out, rows.Err()
}

// Job statuses.
const (
	JobQueued   = "queued"
	JobRunning  = "running"
	JobDone     = "done"
	JobFailed   = "failed"
	JobCanceled = "canceled"
)

// Job types.
const (
	JobRun   = "run"
	JobSweep = "sweep"
)

// Job is a queued or finished backtest/sweep.
type Job struct {
	ID              int64  `json:"id"`
	Type            string `json:"type"`
	StrategyName    string `json:"strategy_name"`
	StrategyVersion string `json:"strategy_version"`
	BinaryPath      string `json:"binary_path"`
	Status          string `json:"status"`
	ConfigJSON      string `json:"config_json"`
	Progress        string `json:"progress"`
	ReportDir       string `json:"report_dir"`
	Error           string `json:"error"`
	PID             int    `json:"pid"`
	CreatedAt       string `json:"created_at"`
	StartedAt       string `json:"started_at"`
	FinishedAt      string `json:"finished_at"`
}

// CreateJob inserts a queued job and returns its id.
func (c *Catalog) CreateJob(typ, strategyName, version, binaryPath, configJSON string) (int64, error) {
	res, err := c.db.Exec(`INSERT INTO jobs(type, strategy_name, strategy_version, binary_path, status, config_json, created_at)
		VALUES(?,?,?,?,?,?,?)`, typ, strategyName, version, binaryPath, JobQueued, configJSON, now())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateJobStatus updates status fields.
func (c *Catalog) UpdateJobStatus(id int64, status string, fields map[string]any) error {
	sets := []string{"status=?"}
	args := []any{status}
	if v, ok := fields["progress"]; ok {
		sets = append(sets, "progress=?")
		args = append(args, v)
	}
	if v, ok := fields["report_dir"]; ok {
		sets = append(sets, "report_dir=?")
		args = append(args, v)
	}
	if v, ok := fields["error"]; ok {
		sets = append(sets, "error=?")
		args = append(args, v)
	}
	if v, ok := fields["pid"]; ok {
		sets = append(sets, "pid=?")
		args = append(args, v)
	}
	if status == JobRunning {
		sets = append(sets, "started_at=?")
		args = append(args, now())
	}
	if status == JobDone || status == JobFailed || status == JobCanceled {
		sets = append(sets, "finished_at=?")
		args = append(args, now())
	}
	args = append(args, id)
	_, err := c.db.Exec(`UPDATE jobs SET `+strings.Join(sets, ", ")+` WHERE id=?`, args...)
	return err
}

// GetJob returns a job by id.
func (c *Catalog) GetJob(id int64) (*Job, error) {
	row := c.db.QueryRow(`SELECT id, type, strategy_name, strategy_version, binary_path, status, config_json, progress, report_dir, error, pid, created_at, started_at, finished_at FROM jobs WHERE id=?`, id)
	var j Job
	if err := row.Scan(&j.ID, &j.Type, &j.StrategyName, &j.StrategyVersion, &j.BinaryPath, &j.Status, &j.ConfigJSON, &j.Progress, &j.ReportDir, &j.Error, &j.PID, &j.CreatedAt, &j.StartedAt, &j.FinishedAt); err != nil {
		return nil, err
	}
	return &j, nil
}

// ListJobs returns recent jobs (newest first).
func (c *Catalog) ListJobs(limit int) ([]Job, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := c.db.Query(`SELECT id, type, strategy_name, strategy_version, binary_path, status, config_json, progress, report_dir, error, pid, created_at, started_at, finished_at FROM jobs ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.Type, &j.StrategyName, &j.StrategyVersion, &j.BinaryPath, &j.Status, &j.ConfigJSON, &j.Progress, &j.ReportDir, &j.Error, &j.PID, &j.CreatedAt, &j.StartedAt, &j.FinishedAt); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// Artifact kinds.
const (
	ArtifactRun   = "run"
	ArtifactSweep = "sweep"
)

// Artifact is an ingested or produced run/sweep folder.
type Artifact struct {
	ID           int64           `json:"id"`
	Kind         string          `json:"kind"`
	StrategyName string          `json:"strategy_name"`
	JobID        int64           `json:"job_id"`
	Path         string          `json:"path"`
	Title        string          `json:"title"`
	MetricsJSON  json.RawMessage `json:"metrics"`
	SummaryJSON  string          `json:"summary_json,omitempty"`
	CreatedAt    string          `json:"created_at"`
	Trials       []Trial         `json:"trials,omitempty"`
}

// Trial is one sweep trial row.
type Trial struct {
	ID         int64           `json:"id"`
	ArtifactID int64           `json:"artifact_id"`
	Index      int             `json:"index"`
	ParamsJSON json.RawMessage `json:"params"`
	MetricsJSON json.RawMessage `json:"metrics"`
	ReportHTML string          `json:"report_html"`
	Error      string          `json:"error"`
}

// AddArtifact inserts an artifact and optional trials.
func (c *Catalog) AddArtifact(a Artifact, trials []Trial) (int64, error) {
	abs, err := filepath.Abs(a.Path)
	if err != nil {
		return 0, err
	}
	a.Path = abs
	if a.MetricsJSON == nil {
		a.MetricsJSON = json.RawMessage(`{}`)
	}
	if a.CreatedAt == "" {
		a.CreatedAt = now()
	}
	res, err := c.db.Exec(`INSERT INTO artifacts(kind, strategy_name, job_id, path, title, metrics_json, summary_json, created_at)
		VALUES(?,?,?,?,?,?,?,?)`, a.Kind, a.StrategyName, a.JobID, a.Path, a.Title, string(a.MetricsJSON), a.SummaryJSON, a.CreatedAt)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	for _, t := range trials {
		if t.ParamsJSON == nil {
			t.ParamsJSON = json.RawMessage(`{}`)
		}
		if t.MetricsJSON == nil {
			t.MetricsJSON = json.RawMessage(`{}`)
		}
		_, err = c.db.Exec(`INSERT INTO trials(artifact_id, trial_index, params_json, metrics_json, report_html, error) VALUES(?,?,?,?,?,?)`,
			id, t.Index, string(t.ParamsJSON), string(t.MetricsJSON), t.ReportHTML, t.Error)
		if err != nil {
			return id, err
		}
	}
	return id, nil
}

// ListArtifacts returns recent artifacts.
func (c *Catalog) ListArtifacts(limit int) ([]Artifact, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := c.db.Query(`SELECT id, kind, strategy_name, job_id, path, title, metrics_json, summary_json, created_at FROM artifacts ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Artifact
	for rows.Next() {
		var a Artifact
		var metrics string
		if err := rows.Scan(&a.ID, &a.Kind, &a.StrategyName, &a.JobID, &a.Path, &a.Title, &metrics, &a.SummaryJSON, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.MetricsJSON = json.RawMessage(metrics)
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetArtifact returns an artifact with trials.
func (c *Catalog) GetArtifact(id int64) (*Artifact, error) {
	row := c.db.QueryRow(`SELECT id, kind, strategy_name, job_id, path, title, metrics_json, summary_json, created_at FROM artifacts WHERE id=?`, id)
	var a Artifact
	var metrics string
	if err := row.Scan(&a.ID, &a.Kind, &a.StrategyName, &a.JobID, &a.Path, &a.Title, &metrics, &a.SummaryJSON, &a.CreatedAt); err != nil {
		return nil, err
	}
	a.MetricsJSON = json.RawMessage(metrics)
	trials, err := c.ListTrials(id)
	if err != nil {
		return nil, err
	}
	a.Trials = trials
	return &a, nil
}

// ListTrials returns trials for an artifact.
func (c *Catalog) ListTrials(artifactID int64) ([]Trial, error) {
	rows, err := c.db.Query(`SELECT id, artifact_id, trial_index, params_json, metrics_json, report_html, error FROM trials WHERE artifact_id=? ORDER BY trial_index`, artifactID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Trial
	for rows.Next() {
		var t Trial
		var params, metrics string
		if err := rows.Scan(&t.ID, &t.ArtifactID, &t.Index, &params, &metrics, &t.ReportHTML, &t.Error); err != nil {
			return nil, err
		}
		t.ParamsJSON = json.RawMessage(params)
		t.MetricsJSON = json.RawMessage(metrics)
		out = append(out, t)
	}
	return out, rows.Err()
}

// RunsDir returns the default directory for new run/sweep outputs.
func (c *Catalog) RunsDir() string {
	return filepath.Join(c.DataDir, "runs")
}
