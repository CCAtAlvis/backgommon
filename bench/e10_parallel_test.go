package bench

import (
	"runtime"
	"testing"
)

func BenchmarkE10_SumRow_Serial(b *testing.B) {
	s := NewSynthSoA(benchS, benchB)
	b.ReportAllocs()
	b.ResetTimer()
	var sink float64
	var n int
	for i := 0; i < b.N; i++ {
		for r := 0; r < s.B; r++ {
			row := s.RowCloses(r)
			var sum float64
			for _, px := range row {
				sum += px
			}
			sink += sum
			n++
		}
	}
	KeepAlive(sink)
	b.StopTimer()
	reportPerBar(b, n)
}

func BenchmarkE10_SumRow_Parallel(b *testing.B) {
	s := NewSynthSoA(benchS, benchB)
	workers := runtime.GOMAXPROCS(0)
	b.ReportAllocs()
	b.ResetTimer()
	var sink float64
	var n int
	for i := 0; i < b.N; i++ {
		for r := 0; r < s.B; r++ {
			sink += SumRowParallel(s.RowCloses(r), workers)
			n++
		}
	}
	KeepAlive(sink)
	b.StopTimer()
	reportPerBar(b, n)
	b.ReportMetric(float64(workers), "workers")
}
