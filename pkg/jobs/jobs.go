// Package jobs runs strategy worker binaries and records results in the catalog.
package jobs

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/CCAtAlvis/backgommon/pkg/catalog"
	"github.com/CCAtAlvis/backgommon/pkg/worker"
)

// ProcHook is called when a worker process starts (for cancel tracking).
type ProcHook func(jobID int64, p *os.Process)

// Runner executes run/sweep jobs against the catalog.
type Runner struct {
	Cat     *catalog.Catalog
	OnStart ProcHook
	OnEnd   func(jobID int64)
	Canceled func(jobID int64) bool
}

// Register describes a binary and upserts it into the catalog.
func Register(cat *catalog.Catalog, binaryPath, nameOverride string) (*catalog.RegisterResult, error) {
	abs, err := filepath.Abs(binaryPath)
	if err != nil {
		return nil, err
	}
	if st, err := os.Stat(abs); err != nil || st.IsDir() {
		return nil, fmt.Errorf("binary not found: %s", abs)
	}
	desc, err := (&worker.Client{Binary: abs}).Describe()
	if err != nil {
		return nil, err
	}
	name := desc.Name
	if nameOverride != "" {
		name = nameOverride
	}
	schema := desc.Schema
	if schema == nil {
		schema = json.RawMessage(`{}`)
	}
	describeBytes, _ := json.Marshal(desc)
	return cat.Register(name, abs, desc.Version, schema, describeBytes)
}

// QueueRun creates a job and runs it synchronously.
func (r *Runner) QueueRun(strategy string, config json.RawMessage) (int64, error) {
	st, err := r.Cat.GetStrategy(strategy)
	if err != nil {
		return 0, fmt.Errorf("strategy not found: %s", strategy)
	}
	if config == nil {
		config = json.RawMessage(`{}`)
	}
	cfgBytes, _ := json.Marshal(map[string]any{"config": json.RawMessage(config)})
	jobID, err := r.Cat.CreateJob(catalog.JobRun, st.Name, st.Version, st.AbsPath, string(cfgBytes))
	if err != nil {
		return 0, err
	}
	r.ExecuteRun(jobID, st, config)
	return jobID, nil
}

// QueueSweep creates a sweep job and runs it synchronously.
func (r *Runner) QueueSweep(strategy string, base, cases, axes json.RawMessage, workers int, sortBy string) (int64, error) {
	st, err := r.Cat.GetStrategy(strategy)
	if err != nil {
		return 0, fmt.Errorf("strategy not found: %s", strategy)
	}
	body := map[string]any{
		"strategy": strategy,
		"base":     base,
		"workers":  workers,
		"sort_by":  sortBy,
	}
	if len(cases) > 0 {
		body["cases"] = cases
	}
	if len(axes) > 0 {
		body["axes"] = axes
	}
	cfgBytes, _ := json.Marshal(body)
	jobID, err := r.Cat.CreateJob(catalog.JobSweep, st.Name, st.Version, st.AbsPath, string(cfgBytes))
	if err != nil {
		return 0, err
	}
	r.ExecuteSweep(jobID, st, base, cases, axes, workers, sortBy)
	return jobID, nil
}

// ExecuteRun runs a queued run job.
func (r *Runner) ExecuteRun(jobID int64, st *catalog.Strategy, config json.RawMessage) {
	_ = r.Cat.UpdateJobStatus(jobID, catalog.JobRunning, map[string]any{"progress": "starting"})
	outDir := filepath.Join(r.Cat.RunsDir(), fmt.Sprintf("job_%d", jobID))
	_ = os.MkdirAll(outDir, 0755)
	req := worker.RunRequest{Config: config, OutputDir: outDir}

	cmd, stdout, stderr, err := worker.StartRun(st.AbsPath, req)
	if err != nil {
		_ = r.Cat.UpdateJobStatus(jobID, catalog.JobFailed, map[string]any{"error": err.Error()})
		return
	}
	if r.OnStart != nil {
		r.OnStart(jobID, cmd.Process)
	}
	_ = r.Cat.UpdateJobStatus(jobID, catalog.JobRunning, map[string]any{"pid": cmd.Process.Pid, "progress": "running"})

	outBytes, _ := io.ReadAll(stdout)
	errBytes, _ := io.ReadAll(stderr)
	waitErr := cmd.Wait()
	if r.OnEnd != nil {
		r.OnEnd(jobID)
	}
	if r.Canceled != nil && r.Canceled(jobID) {
		return
	}

	var resp worker.RunResponse
	_ = json.Unmarshal(outBytes, &resp)
	if waitErr != nil || !resp.OK {
		msg := resp.Error
		if msg == "" {
			msg = string(errBytes)
		}
		if msg == "" && waitErr != nil {
			msg = waitErr.Error()
		}
		_ = r.Cat.UpdateJobStatus(jobID, catalog.JobFailed, map[string]any{"error": msg, "progress": string(errBytes)})
		return
	}
	reportDir := resp.ReportDir
	if reportDir == "" {
		reportDir = outDir
	}
	metrics, _ := json.Marshal(resp.Metrics)
	_, _ = r.Cat.AddArtifact(catalog.Artifact{
		Kind:         catalog.ArtifactRun,
		StrategyName: st.Name,
		JobID:        jobID,
		Path:         reportDir,
		Title:        filepath.Base(reportDir),
		MetricsJSON:  metrics,
	}, nil)
	_ = r.Cat.UpdateJobStatus(jobID, catalog.JobDone, map[string]any{"report_dir": reportDir, "progress": "done"})
}

// ExecuteSweep runs a queued sweep job.
func (r *Runner) ExecuteSweep(jobID int64, st *catalog.Strategy, base, cases, axes json.RawMessage, workers int, sortBy string) {
	_ = r.Cat.UpdateJobStatus(jobID, catalog.JobRunning, map[string]any{"progress": "starting"})
	outDir := filepath.Join(r.Cat.RunsDir(), fmt.Sprintf("job_%d", jobID))
	_ = os.MkdirAll(outDir, 0755)
	if base == nil {
		base = json.RawMessage(`{}`)
	}
	wreq := worker.SweepRequest{
		Base: base, Cases: cases, Axes: axes,
		OutputDir: outDir, Workers: workers, SortBy: sortBy,
	}

	cmd, stdout, stderr, err := worker.StartSweep(st.AbsPath, wreq)
	if err != nil {
		_ = r.Cat.UpdateJobStatus(jobID, catalog.JobFailed, map[string]any{"error": err.Error()})
		return
	}
	if r.OnStart != nil {
		r.OnStart(jobID, cmd.Process)
	}
	_ = r.Cat.UpdateJobStatus(jobID, catalog.JobRunning, map[string]any{"pid": cmd.Process.Pid, "progress": "running"})

	outBytes, _ := io.ReadAll(stdout)
	errBytes, _ := io.ReadAll(stderr)
	waitErr := cmd.Wait()
	if r.OnEnd != nil {
		r.OnEnd(jobID)
	}
	if r.Canceled != nil && r.Canceled(jobID) {
		return
	}

	var resp worker.SweepResponse
	_ = json.Unmarshal(outBytes, &resp)
	if waitErr != nil || !resp.OK {
		msg := resp.Error
		if msg == "" {
			msg = string(errBytes)
		}
		if msg == "" && waitErr != nil {
			msg = waitErr.Error()
		}
		_ = r.Cat.UpdateJobStatus(jobID, catalog.JobFailed, map[string]any{"error": msg, "progress": string(errBytes)})
		return
	}
	sweepDir := resp.SweepDir
	if sweepDir == "" {
		sweepDir = outDir
	}
	art, err := r.Cat.IngestDir(sweepDir, st.Name, jobID)
	if err != nil {
		_ = r.Cat.UpdateJobStatus(jobID, catalog.JobFailed, map[string]any{"error": "ingest: " + err.Error(), "report_dir": sweepDir})
		return
	}
	_ = r.Cat.UpdateJobStatus(jobID, catalog.JobDone, map[string]any{
		"report_dir": sweepDir,
		"progress":   fmt.Sprintf("artifact %d", art.ID),
	})
}
