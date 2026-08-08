package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/output"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

func BenchmarkE6_ExportFull(b *testing.B) {
	data := fakeReport(5000, 20)
	dir := b.TempDir()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out := filepath.Join(dir, fmt.Sprintf("full_%d", i))
		if err := output.ExportRunTo(out, data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkE6_ExportMetricsOnly(b *testing.B) {
	data := fakeReport(5000, 20)
	dir := b.TempDir()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out := filepath.Join(dir, fmt.Sprintf("metrics_%d", i))
		if err := os.MkdirAll(out, 0755); err != nil {
			b.Fatal(err)
		}
		metrics := map[string]interface{}{
			"returns":      data.Results.Returns,
			"cagr":         data.Results.CAGR,
			"max_drawdown": data.Results.MaxDrawdown,
			"sharpe_ratio": data.Results.SharpeRatio,
			"total_trades": data.Results.TotalTrades,
		}
		raw, err := json.Marshal(metrics)
		if err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(out, "metrics.json"), raw, 0644); err != nil {
			b.Fatal(err)
		}
	}
}

func fakeReport(bars, holdings int) output.ReportData {
	curve := make([]types.AccountValue, bars)
	base := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < bars; i++ {
		hs := make([]types.HoldingSnapshot, holdings)
		for j := 0; j < holdings; j++ {
			hs[j] = types.HoldingSnapshot{
				Instrument: fmt.Sprintf("S%04d", j),
				Quantity:   10,
				Price:      100,
				AvgEntry:   90,
			}
		}
		curve[i] = types.AccountValue{
			Time:          base.AddDate(0, 0, i),
			Value:         1e6 + float64(i),
			Cash:          1e5,
			OpenPositions: holdings,
			Holdings:      hs,
		}
	}
	return output.ReportData{
		Results: &types.Results{
			Returns:     0.5,
			CAGR:        0.1,
			MaxDrawdown: 0.2,
			SharpeRatio: 1.2,
			TotalTrades: 100,
		},
		EquityCurve: curve,
		Settings:    map[string]string{"bench": "e6"},
	}
}
