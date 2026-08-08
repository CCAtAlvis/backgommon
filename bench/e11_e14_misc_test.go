package bench

import (
	"testing"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
	"github.com/CCAtAlvis/backgommon/pkg/strategy"
)

// E11: naive unrolled sum (SIMD stand-in) — measure if hand-tuned loop helps SoA.
func BenchmarkE11_SumUnrolled4(b *testing.B) {
	s := NewSynthSoA(benchS, benchB)
	b.ReportAllocs()
	b.ResetTimer()
	var sink float64
	var n int
	for i := 0; i < b.N; i++ {
		for r := 0; r < s.B; r++ {
			row := s.RowCloses(r)
			var sum float64
			j := 0
			for ; j+3 < len(row); j += 4 {
				sum += row[j] + row[j+1] + row[j+2] + row[j+3]
			}
			for ; j < len(row); j++ {
				sum += row[j]
			}
			sink += sum
			n++
		}
	}
	KeepAlive(sink)
	b.StopTimer()
	reportPerBar(b, n)
}

// E12: Candle with vs without indicator map present (copy cost proxy).
func BenchmarkE12_CopyCandle_NoIndicators(b *testing.B) {
	c := core.Candle{Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 100}
	b.ReportAllocs()
	b.ResetTimer()
	var sink float64
	for i := 0; i < b.N; i++ {
		cp := c // value copy
		sink += cp.Close
	}
	KeepAlive(sink)
}

func BenchmarkE12_CopyCandle_WithIndicators(b *testing.B) {
	c := core.Candle{Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 100}
	c.SetIndicator("mcap", 1e9)
	c.SetIndicator("sma_20", 1.2)
	b.ReportAllocs()
	b.ResetTimer()
	var sink float64
	for i := 0; i < b.N; i++ {
		cp := c.Clone()
		sink += cp.Close
	}
	KeepAlive(sink)
}

// E13: day-lifecycle type assert + truncate every bar vs cached.
type dayLifecycle interface {
	OnDayStart(time.Time)
	OnDayEnd(time.Time)
}

type dayStrat struct {
	strategy.BaseStrategy
	starts int
}

func (d *dayStrat) OnDayStart(t time.Time) { d.starts++ }
func (d *dayStrat) OnDayEnd(t time.Time)   {}

func BenchmarkE13_TypeAssertEveryBar(b *testing.B) {
	var strat interfaces.Strategy = &dayStrat{}
	ts := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	b.ReportAllocs()
	b.ResetTimer()
	var sink int
	for i := 0; i < b.N; i++ {
		if dl, ok := strat.(dayLifecycle); ok {
			day := ts.AddDate(0, 0, i%365).Truncate(24 * time.Hour)
			dl.OnDayStart(day)
			sink++
		}
	}
	KeepAlive(float64(sink))
}

func BenchmarkE13_CachedLifecycle(b *testing.B) {
	strat := &dayStrat{}
	var dl dayLifecycle = strat
	ts := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	b.ReportAllocs()
	b.ResetTimer()
	var sink int
	for i := 0; i < b.N; i++ {
		day := ts.AddDate(0, 0, i%365)
		dl.OnDayStart(day)
		sink++
	}
	KeepAlive(float64(sink))
}

// E14: placeholder — PGO requires build flags; document in RESULTS.
func TestE14_PGONote(t *testing.T) {
	t.Log("PGO: build with 'go test -c -o bench.test ./bench' then " +
		"'go test -bench=E3_SoA -cpuprofile=cpu.pprof' and rebuild with -pgo=cpu.pprof. " +
		"Typical gain 5-15%; measure on your machine before adopting in CI.")
}
