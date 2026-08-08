package bench

import (
	"testing"

	"github.com/CCAtAlvis/backgommon/pkg/core"
)

func BenchmarkE1_LazyNext_0orders(b *testing.B) {
	benchmarkLazyNext(b, 0)
}

func BenchmarkE1_LazyNext_1order(b *testing.B) {
	benchmarkLazyNext(b, 1)
}

func BenchmarkE1_LazyNext_20orders(b *testing.B) {
	benchmarkLazyNext(b, 20)
}

func BenchmarkE1_FullNextFillRow(b *testing.B) {
	s := NewSynth(benchS, benchB)
	nxt := make(map[string]core.Candle, benchS)
	b.ReportAllocs()
	b.ResetTimer()
	var n int
	for i := 0; i < b.N; i++ {
		for r := 0; r < s.B-1; r++ {
			_ = s.Table.FillRow(s.Times[r+1], nxt)
			n++
		}
	}
	b.StopTimer()
	reportPerBar(b, n)
}

func benchmarkLazyNext(b *testing.B, nOrders int) {
	s := NewSynth(benchS, benchB)
	syms := make([]string, nOrders)
	for i := 0; i < nOrders; i++ {
		syms[i] = s.Symbols[i]
	}
	out := make(map[string]float64, nOrders)
	b.ReportAllocs()
	b.ResetTimer()
	var n int
	for i := 0; i < b.N; i++ {
		for r := 0; r < s.B-1; r++ {
			LazyNextCloses(s.Table, s.Times[r+1], syms, out)
			n++
		}
	}
	b.StopTimer()
	reportPerBar(b, n)
}
