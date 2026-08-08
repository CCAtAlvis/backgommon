package sweep

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/CCAtAlvis/backgommon/pkg/types"
)

// Axis describes one parameter that took multiple values across the sweep.
type Axis struct {
	Name   string `json:"name"`
	Values []any  `json:"values"`
}

// Summary is the enriched sweep summary written to summary.json and embedded
// in dashboard.html.
type Summary struct {
	SweepDir   string         `json:"sweep_dir"`
	SortBy     string         `json:"sort_by"`
	Axes       []Axis         `json:"axes"`
	MetricKeys []string       `json:"metric_keys"`
	Trials     []SummaryTrial `json:"trials"`
}

// SummaryTrial is one trial row in the summary / dashboard payload.
type SummaryTrial struct {
	Index      int                `json:"index"`
	Params     json.RawMessage    `json:"params"`
	ParamsMap  map[string]any     `json:"-"` // filled when building; not serialized separately
	Metrics    map[string]float64 `json:"metrics,omitempty"`
	DurationMS int64              `json:"duration_ms"`
	ReportHTML string             `json:"report_html"`
	Error      string             `json:"error,omitempty"`
}

func buildSummary[T any](sweepDir, sortBy string, trials []Trial[T]) (*Summary, error) {
	out := &Summary{
		SweepDir: sweepDir,
		SortBy:   sortBy,
		Trials:   make([]SummaryTrial, 0, len(trials)),
	}

	metricKeySet := map[string]bool{}
	for _, t := range trials {
		paramsJSON, err := json.Marshal(t.Params)
		if err != nil {
			return nil, fmt.Errorf("marshal params for trial %d: %w", t.Index, err)
		}
		var paramsMap map[string]any
		if err := json.Unmarshal(paramsJSON, &paramsMap); err != nil {
			return nil, fmt.Errorf("unmarshal params for trial %d: %w", t.Index, err)
		}
		st := SummaryTrial{
			Index:      t.Index,
			Params:     paramsJSON,
			ParamsMap:  paramsMap,
			DurationMS: t.Duration.Milliseconds(),
			ReportHTML: t.ReportHTML,
		}
		if t.Err != nil {
			st.Error = t.Err.Error()
		}
		if t.Results != nil {
			st.Metrics = extractMetrics(t.Results)
			for k := range st.Metrics {
				metricKeySet[k] = true
			}
		}
		out.Trials = append(out.Trials, st)
	}

	out.Axes = deriveAxes(out.Trials)
	out.MetricKeys = orderedMetricKeys(metricKeySet, sortBy)
	return out, nil
}

func writeSummaryJSON(path string, summary *Summary) error {
	raw, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0644)
}

// deriveAxes finds param keys with more than one distinct value across trials.
func deriveAxes(trials []SummaryTrial) []Axis {
	valueSets := map[string]map[string]any{} // key -> canonical JSON -> value
	keyOrder := []string{}

	for _, t := range trials {
		for k, v := range t.ParamsMap {
			canon, err := json.Marshal(v)
			if err != nil {
				continue
			}
			cs := string(canon)
			if _, ok := valueSets[k]; !ok {
				valueSets[k] = map[string]any{}
				keyOrder = append(keyOrder, k)
			}
			valueSets[k][cs] = v
		}
	}

	sort.Strings(keyOrder)
	var axes []Axis
	for _, k := range keyOrder {
		set := valueSets[k]
		if len(set) <= 1 {
			continue
		}
		vals := make([]any, 0, len(set))
		canonKeys := make([]string, 0, len(set))
		for ck := range set {
			canonKeys = append(canonKeys, ck)
		}
		sort.Strings(canonKeys)
		for _, ck := range canonKeys {
			vals = append(vals, set[ck])
		}
		axes = append(axes, Axis{Name: k, Values: vals})
	}
	return axes
}

func orderedMetricKeys(set map[string]bool, sortBy string) []string {
	preferred := []string{
		"sharpe_ratio", "sortino_ratio", "cagr", "returns", "max_drawdown",
		"profit_factor", "final_capital", "total_trades", "winning_trades", "losing_trades", "win_rate",
	}
	seen := map[string]bool{}
	var out []string
	// Put sortBy first if present.
	if set[sortBy] {
		out = append(out, sortBy)
		seen[sortBy] = true
	}
	for _, k := range preferred {
		if set[k] && !seen[k] {
			out = append(out, k)
			seen[k] = true
		}
	}
	rest := make([]string, 0)
	for k := range set {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	return append(out, rest...)
}

func extractMetrics(r *types.Results) map[string]float64 {
	m := map[string]float64{
		"returns":        r.Returns,
		"cagr":           r.CAGR,
		"max_drawdown":   r.MaxDrawdown,
		"sharpe_ratio":   r.SharpeRatio,
		"sortino_ratio":  r.SortinoRatio,
		"profit_factor":  r.ProfitFactor,
		"final_capital":  r.FinalCapital,
		"total_trades":   float64(r.TotalTrades),
		"winning_trades": float64(r.WinningTrades),
		"losing_trades":  float64(r.LosingTrades),
	}
	if r.TotalTrades > 0 {
		m["win_rate"] = float64(r.WinningTrades) / float64(r.TotalTrades)
	}
	for k, v := range r.Metrics {
		m[k] = v
	}
	return m
}
