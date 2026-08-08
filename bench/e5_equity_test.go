package bench

import (
	"fmt"
	"testing"
)

func BenchmarkE5_EquityFull_perBar(b *testing.B) {
	instruments := make([]string, 20)
	prices := make(map[string]float64, 20)
	for i := 0; i < 20; i++ {
		instruments[i] = fmt.Sprintf("S%04d", i)
		prices[instruments[i]] = 100 + float64(i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	var sink float64
	for i := 0; i < b.N; i++ {
		v, _, _ := EquityFull(1e6, 1e5, instruments, prices)
		sink += v
	}
	KeepAlive(sink)
}

func BenchmarkE5_EquityValueOnly_perBar(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	var sink float64
	for i := 0; i < b.N; i++ {
		sink += EquityValueOnly(1e6, 1e5)
	}
	KeepAlive(sink)
}

func BenchmarkE5_EquityEveryN10(b *testing.B) {
	instruments := make([]string, 20)
	prices := make(map[string]float64, 20)
	for i := 0; i < 20; i++ {
		instruments[i] = fmt.Sprintf("S%04d", i)
		prices[instruments[i]] = 100 + float64(i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	var sink float64
	for i := 0; i < b.N; i++ {
		if i%10 == 0 {
			v, _, _ := EquityFull(1e6, 1e5, instruments, prices)
			sink += v
		} else {
			sink += EquityValueOnly(1e6, 1e5)
		}
	}
	KeepAlive(sink)
}
