package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/CCAtAlvis/backgommon/pkg/types"
)

// Report is the top-level JSON structure exported by the framework.
type Report struct {
	Meta        ReportMeta        `json:"meta"`
	Metrics     ReportMetrics     `json:"metrics"`
	EquityCurve []EquityCurvePoint `json:"equity_curve"`
}

// ReportMeta holds high-level information about the backtest run.
type ReportMeta struct {
	StartTime      string  `json:"start_time"`
	EndTime        string  `json:"end_time"`
	InitialCapital float64 `json:"initial_capital"`
	FinalCapital   float64 `json:"final_capital"`
}

// ReportMetrics holds computed performance metrics.
type ReportMetrics struct {
	Returns        float64 `json:"returns"`
	CAGR           float64 `json:"cagr"`
	MaxDrawdown    float64 `json:"max_drawdown"`
	SharpeRatio    float64 `json:"sharpe_ratio"`
	SortinoRatio   float64 `json:"sortino_ratio"`
	TotalTrades    int     `json:"total_trades"`
	WinningTrades  int     `json:"winning_trades"`
	LosingTrades   int     `json:"losing_trades"`
	WinRate        float64 `json:"win_rate"`
	AvgWinPercent  float64 `json:"avg_win_percent"`
	AvgLossPercent float64 `json:"avg_loss_percent"`

	RiskFreeRate       float64 `json:"risk_free_rate"`
	RiskFreeRateSource string  `json:"risk_free_rate_source"`

	Custom map[string]float64 `json:"custom,omitempty"`
}

// EquityCurvePoint represents a single snapshot in the equity curve.
type EquityCurvePoint struct {
	Time          string                  `json:"time"`
	Value         float64                 `json:"value"`
	Cash          float64                 `json:"cash"`
	OpenPositions int                     `json:"open_positions"`
	UnrealizedPnL float64                 `json:"unrealized_pnl"`
	Holdings      []types.HoldingSnapshot `json:"holdings,omitempty"`
}

// BuildReport constructs a Report from framework results and equity curve.
func BuildReport(results *types.Results, curve []types.AccountValue) Report {
	r := Report{
		Meta: ReportMeta{
			InitialCapital: results.InitialCapital,
			FinalCapital:   results.FinalCapital,
		},
		Metrics: ReportMetrics{
			Returns:            results.Returns,
			CAGR:               results.CAGR,
			MaxDrawdown:        results.MaxDrawdown,
			SharpeRatio:        results.SharpeRatio,
			SortinoRatio:       results.SortinoRatio,
			TotalTrades:        results.TotalTrades,
			WinningTrades:      results.WinningTrades,
			LosingTrades:       results.LosingTrades,
			AvgWinPercent:      results.AvgWinPercent,
			AvgLossPercent:     results.AvgLossPercent,
			RiskFreeRate:       results.RiskFreeRate,
			RiskFreeRateSource: results.RiskFreeRateSource,
			Custom:             results.Metrics,
		},
	}

	if !results.StartTime.IsZero() {
		r.Meta.StartTime = results.StartTime.Format("2006-01-02")
	}
	if !results.EndTime.IsZero() {
		r.Meta.EndTime = results.EndTime.Format("2006-01-02")
	}

	if results.TotalTrades > 0 {
		r.Metrics.WinRate = float64(results.WinningTrades) / float64(results.TotalTrades)
	}

	r.EquityCurve = make([]EquityCurvePoint, len(curve))
	for i, pt := range curve {
		r.EquityCurve[i] = EquityCurvePoint{
			Time:          pt.Time.Format("2006-01-02"),
			Value:         pt.Value,
			Cash:          pt.Cash,
			OpenPositions: pt.OpenPositions,
			UnrealizedPnL: pt.UnrealizedPnL,
			Holdings:      pt.Holdings,
		}
	}

	return r
}

// ExportEquityCurveJSON writes only the equity curve to a JSON file.
func ExportEquityCurveJSON(curve []types.AccountValue, path string) error {
	points := make([]EquityCurvePoint, len(curve))
	for i, pt := range curve {
		points[i] = EquityCurvePoint{
			Time:          pt.Time.Format("2006-01-02"),
			Value:         pt.Value,
			Cash:          pt.Cash,
			OpenPositions: pt.OpenPositions,
			UnrealizedPnL: pt.UnrealizedPnL,
			Holdings:      pt.Holdings,
		}
	}
	return writeJSON(points, path)
}

// ExportResultsJSON writes the full report (meta + metrics + equity curve) to a JSON file.
func ExportResultsJSON(results *types.Results, curve []types.AccountValue, path string) error {
	if results == nil {
		return fmt.Errorf("results is nil")
	}
	report := BuildReport(results, curve)
	return writeJSON(report, path)
}

func writeJSON(v interface{}, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}
