package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CCAtAlvis/backgommon/pkg/sweep"
)

// IngestDir imports an existing run or sweep folder into the catalog.
// Detects sweep via summary.json / dashboard.html; otherwise treats as single run.
func (c *Catalog) IngestDir(dir, strategyName string, jobID int64) (*Artifact, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(abs)
	if err != nil || !st.IsDir() {
		return nil, fmt.Errorf("ingest: %s is not a directory", abs)
	}

	summaryPath := filepath.Join(abs, "summary.json")
	if _, err := os.Stat(summaryPath); err == nil {
		return c.ingestSweep(abs, summaryPath, strategyName, jobID)
	}
	// Single-run: look for results.json(.zst) or viewer.html
	return c.ingestRun(abs, strategyName, jobID)
}

func (c *Catalog) ingestSweep(abs, summaryPath, strategyName string, jobID int64) (*Artifact, error) {
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		return nil, err
	}
	var summary sweep.Summary
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, fmt.Errorf("parse summary.json: %w", err)
	}

	var topMetrics json.RawMessage = json.RawMessage(`{}`)
	if len(summary.Trials) > 0 && summary.Trials[0].Metrics != nil {
		b, _ := json.Marshal(summary.Trials[0].Metrics)
		topMetrics = b
	}

	trials := make([]Trial, 0, len(summary.Trials))
	for _, t := range summary.Trials {
		params := t.Params
		if params == nil {
			params = json.RawMessage(`{}`)
		}
		metrics := json.RawMessage(`{}`)
		if t.Metrics != nil {
			b, _ := json.Marshal(t.Metrics)
			metrics = b
		}
		trials = append(trials, Trial{
			Index:       t.Index,
			ParamsJSON:  params,
			MetricsJSON: metrics,
			ReportHTML:  t.ReportHTML,
			Error:       t.Error,
		})
	}

	title := filepath.Base(abs)
	id, err := c.AddArtifact(Artifact{
		Kind:         ArtifactSweep,
		StrategyName: strategyName,
		JobID:        jobID,
		Path:         abs,
		Title:        title,
		MetricsJSON:  topMetrics,
		SummaryJSON:  string(data),
	}, trials)
	if err != nil {
		return nil, err
	}
	return c.GetArtifact(id)
}

func (c *Catalog) ingestRun(abs, strategyName string, jobID int64) (*Artifact, error) {
	metrics := json.RawMessage(`{}`)
	// Prefer results.json then results.json.zst via loose parse of config if present
	for _, name := range []string{"results.json", "config.json"} {
		p := filepath.Join(abs, name)
		if b, err := os.ReadFile(p); err == nil && name == "results.json" {
			var wrap struct {
				Metrics map[string]float64 `json:"metrics"`
			}
			if json.Unmarshal(b, &wrap) == nil && wrap.Metrics != nil {
				metrics, _ = json.Marshal(wrap.Metrics)
			}
		}
		_ = p
	}
	// Also try loading metrics from a thin results if only zst exists — leave empty if not readable here.

	title := filepath.Base(abs)
	if strategyName == "" {
		strategyName = guessStrategyFromPath(abs)
	}
	id, err := c.AddArtifact(Artifact{
		Kind:         ArtifactRun,
		StrategyName: strategyName,
		JobID:        jobID,
		Path:         abs,
		Title:        title,
		MetricsJSON:  metrics,
	}, nil)
	if err != nil {
		return nil, err
	}
	return c.GetArtifact(id)
}

func guessStrategyFromPath(abs string) string {
	base := filepath.Base(abs)
	if strings.HasPrefix(base, "run_") || strings.HasPrefix(base, "sweep_") {
		return ""
	}
	return base
}
