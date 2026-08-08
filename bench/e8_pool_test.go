package bench

import (
	"sync"
	"testing"

	"github.com/CCAtAlvis/backgommon/pkg/core"
)

func BenchmarkE8_ReuseMap(b *testing.B) {
	s := NewSynth(benchS, 1)
	data := make(map[string]core.Candle, benchS)
	_ = s.Table.FillRow(s.Times[0], data)
	prices := make(map[string]float64, benchS)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FillPrices(data, prices)
	}
}

func BenchmarkE8_PoolMap(b *testing.B) {
	s := NewSynth(benchS, 1)
	data := make(map[string]core.Candle, benchS)
	_ = s.Table.FillRow(s.Times[0], data)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		prices := BorrowPriceMap()
		FillPrices(data, prices)
		ReturnPriceMap(prices)
	}
}

func BenchmarkE8_NewMapEachBar(b *testing.B) {
	s := NewSynth(benchS, 1)
	data := make(map[string]core.Candle, benchS)
	_ = s.Table.FillRow(s.Times[0], data)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		prices := make(map[string]float64, benchS)
		FillPrices(data, prices)
	}
}

// Ensure pool type stays linked
var _ sync.Pool
