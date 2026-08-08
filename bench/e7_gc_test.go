package bench

import (
	"runtime"
	"runtime/debug"
	"testing"

	"github.com/CCAtAlvis/backgommon/pkg/core"
)

func BenchmarkE7_W1_DefaultGC(b *testing.B) {
	benchmarkW1Body(b)
}

func BenchmarkE7_W1_DisabledGC(b *testing.B) {
	old := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(old)
	s := NewSynth(benchS, benchB)
	cur := make(map[string]core.Candle, benchS)
	prices := make(map[string]float64, benchS)
	instruments := s.Symbols[:20]
	b.ReportAllocs()
	b.ResetTimer()
	var sink float64
	var n int
	var maxAlloc uint64
	for i := 0; i < b.N; i++ {
		for r := 0; r < s.B; r++ {
			_ = s.Table.FillRow(s.Times[r], cur)
			FillPrices(cur, prices)
			v, _, _ := EquityFull(1e6, 1e5, instruments, prices)
			sink += v
			n++
		}
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		if ms.Alloc > maxAlloc {
			maxAlloc = ms.Alloc
		}
	}
	KeepAlive(sink)
	b.StopTimer()
	reportPerBar(b, n)
	b.ReportMetric(float64(maxAlloc)/1e6, "maxAlloc_MB")
}

func benchmarkW1Body(b *testing.B) {
	s := NewSynth(benchS, benchB)
	cur := make(map[string]core.Candle, benchS)
	prices := make(map[string]float64, benchS)
	instruments := s.Symbols[:20]
	b.ReportAllocs()
	runtime.GC()
	b.ResetTimer()
	var sink float64
	var n int
	for i := 0; i < b.N; i++ {
		for r := 0; r < s.B; r++ {
			_ = s.Table.FillRow(s.Times[r], cur)
			FillPrices(cur, prices)
			v, _, _ := EquityFull(1e6, 1e5, instruments, prices)
			sink += v
			n++
		}
	}
	KeepAlive(sink)
	b.StopTimer()
	reportPerBar(b, n)
}
