package bench

import (
	"testing"
)

func BenchmarkE9c_OnTheFlyLookback(b *testing.B) {
	s := NewSynthSoA(benchS, benchB)
	const lookback = 120 // ~6 months of daily bars
	b.ReportAllocs()
	b.ResetTimer()
	var sink float64
	var n int
	for i := 0; i < b.N; i++ {
		for t := 0; t < s.B; t++ {
			sink += OnTheFlyLookbackReturn(s.Closes, s.S, t, lookback)
			n++
		}
	}
	KeepAlive(sink)
	b.StopTimer()
	reportPerBar(b, n)
}

func BenchmarkE9a_PrecomputeThenRead(b *testing.B) {
	s := NewSynthSoA(benchS, benchB)
	const lookback = 120
	panel := NewFeaturePanel(s.B, s.S, 1)
	PrecomputeLookbackReturn(panel, s.Closes, s.S, lookback, 0)
	b.ReportAllocs()
	b.ResetTimer()
	var sink float64
	var n int
	for i := 0; i < b.N; i++ {
		for t := 0; t < s.B; t++ {
			var sum float64
			for c := 0; c < s.S; c++ {
				sum += panel.At(t, c, 0)
			}
			sink += sum
			n++
		}
	}
	KeepAlive(sink)
	b.StopTimer()
	reportPerBar(b, n)
	b.ReportMetric(float64(len(panel.Data)*8)/1e6, "panel_MB")
}

func BenchmarkE9b_Precompute3Lookbacks(b *testing.B) {
	s := NewSynthSoA(benchS, benchB)
	lookbacks := []int{60, 120, 180}
	panel := NewFeaturePanel(s.B, s.S, len(lookbacks))
	for fi, lb := range lookbacks {
		PrecomputeLookbackReturn(panel, s.Closes, s.S, lb, fi)
	}
	b.ReportAllocs()
	b.ResetTimer()
	var sink float64
	var n int
	for i := 0; i < b.N; i++ {
		for t := 0; t < s.B; t++ {
			var sum float64
			for c := 0; c < s.S; c++ {
				sum += panel.At(t, c, 0) + panel.At(t, c, 1) + panel.At(t, c, 2)
			}
			sink += sum
			n++
		}
	}
	KeepAlive(sink)
	b.StopTimer()
	reportPerBar(b, n)
	b.ReportMetric(float64(len(panel.Data)*8)/1e6, "panel_MB")
}

func BenchmarkE9_PrecomputeBuildCost(b *testing.B) {
	s := NewSynthSoA(benchS, benchB)
	const lookback = 120
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		panel := NewFeaturePanel(s.B, s.S, 1)
		PrecomputeLookbackReturn(panel, s.Closes, s.S, lookback, 0)
		KeepAlive(panel.At(0, 0, 0))
	}
}
