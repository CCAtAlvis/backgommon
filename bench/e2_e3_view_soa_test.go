package bench

import (
	"testing"

	"github.com/CCAtAlvis/backgommon/pkg/core"
)

func BenchmarkE2_MarketView_SumAll(b *testing.B) {
	s := NewSynth(benchS, benchB)
	cols := s.Symbols
	b.ReportAllocs()
	b.ResetTimer()
	var sink float64
	var n int
	for i := 0; i < b.N; i++ {
		for r := 0; r < s.B; r++ {
			v := MarketView{Table: s.Table, Ts: s.Times[r], Cols: cols}
			sink += v.SumCloses()
			n++
		}
	}
	KeepAlive(sink)
	b.StopTimer()
	reportPerBar(b, n)
}

func BenchmarkE3_SoA_SumRow(b *testing.B) {
	s := NewSynthSoA(benchS, benchB)
	b.ReportAllocs()
	b.ResetTimer()
	var sink float64
	var n int
	for i := 0; i < b.N; i++ {
		for r := 0; r < s.B; r++ {
			row := s.RowCloses(r)
			for _, px := range row {
				sink += px
			}
			n++
		}
	}
	KeepAlive(sink)
	b.StopTimer()
	reportPerBar(b, n)
}

func BenchmarkE3_SoA_SumRow_10k(b *testing.B) {
	s := NewSynthSoA(benchS, 10_000)
	b.ReportAllocs()
	b.ResetTimer()
	var sink float64
	var n int
	for i := 0; i < b.N; i++ {
		for r := 0; r < s.B; r++ {
			row := s.RowCloses(r)
			for _, px := range row {
				sink += px
			}
			n++
		}
	}
	KeepAlive(sink)
	b.StopTimer()
	reportPerBar(b, n)
}

func BenchmarkE3_SoA_SumRow_100k(b *testing.B) {
	s := NewSynthSoA(benchS, 100_000)
	b.ReportAllocs()
	b.ResetTimer()
	var sink float64
	var n int
	for i := 0; i < b.N; i++ {
		for r := 0; r < s.B; r++ {
			row := s.RowCloses(r)
			for _, px := range row {
				sink += px
			}
			n++
		}
	}
	KeepAlive(sink)
	b.StopTimer()
	reportPerBar(b, n)
}

func BenchmarkE3_FillRowThenScanMap(b *testing.B) {
	// Current-style full-universe touch: FillRow + range closes.
	s := NewSynth(benchS, benchB)
	dst := make(map[string]core.Candle, benchS)
	b.ReportAllocs()
	b.ResetTimer()
	var sink float64
	var n int
	for i := 0; i < b.N; i++ {
		for r := 0; r < s.B; r++ {
			_ = s.Table.FillRow(s.Times[r], dst)
			for _, c := range dst {
				sink += c.Close
			}
			n++
		}
	}
	KeepAlive(sink)
	b.StopTimer()
	reportPerBar(b, n)
}
