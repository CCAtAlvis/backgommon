package bench

import (
	"fmt"
	"testing"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
	"github.com/CCAtAlvis/backgommon/pkg/risk"
	"github.com/CCAtAlvis/backgommon/pkg/runner"
	"github.com/CCAtAlvis/backgommon/pkg/strategy"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

const (
	benchS = 2000
	benchB = 1000
)

func BenchmarkE0_FillRow(b *testing.B) {
	s := NewSynth(benchS, benchB)
	dst := make(map[string]core.Candle, benchS)
	b.ReportAllocs()
	b.ResetTimer()
	var n int
	for i := 0; i < b.N; i++ {
		for r := 0; r < s.B; r++ {
			if !s.Table.FillRow(s.Times[r], dst) {
				b.Fatal("FillRow failed")
			}
			n++
		}
	}
	b.StopTimer()
	reportPerBar(b, n)
}

func BenchmarkE0_DualFillRow(b *testing.B) {
	s := NewSynth(benchS, benchB)
	cur := make(map[string]core.Candle, benchS)
	nxt := make(map[string]core.Candle, benchS)
	b.ReportAllocs()
	b.ResetTimer()
	var n int
	for i := 0; i < b.N; i++ {
		for r := 0; r < s.B-1; r++ {
			_ = s.Table.FillRow(s.Times[r], cur)
			_ = s.Table.FillRow(s.Times[r+1], nxt)
			n++
		}
	}
	b.StopTimer()
	reportPerBar(b, n)
}

func BenchmarkE0_FillPrices(b *testing.B) {
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

func BenchmarkE0_GetValueAll(b *testing.B) {
	s := NewSynth(benchS, benchB)
	cols := s.Symbols
	b.ReportAllocs()
	b.ResetTimer()
	var sink float64
	var n int
	for i := 0; i < b.N; i++ {
		for r := 0; r < s.B; r++ {
			ts := s.Times[r]
			for _, col := range cols {
				c, ok := s.Table.GetValue(ts, col)
				if ok {
					sink += c.Close
				}
			}
			n++
		}
	}
	KeepAlive(sink)
	b.StopTimer()
	reportPerBar(b, n)
}

func BenchmarkE0_PositionsCopy(b *testing.B) {
	p := portfolio.New(&portfolio.Settings{InitialCapital: 1e9, DefaultLeverage: 1})
	for i := 0; i < 20; i++ {
		ord := portfolio.NewOrder(fmt.Sprintf("S%04d", i), portfolio.Long, portfolio.Entry, 10, 1)
		ord.Price = 100
		ord.FilledAt = time.Now()
		if err := p.ProcessOrder(ord); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.Positions()
	}
}

func BenchmarkE0_EquityFull(b *testing.B) {
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

type noopFullScanStrategy struct {
	strategy.BaseStrategy
	sink float64
}

func (s *noopFullScanStrategy) OnTick(data map[string]core.Candle) []portfolio.Order {
	for _, c := range data {
		s.sink += c.Close
	}
	return nil
}

func BenchmarkE0_RunnerFullScan(b *testing.B) {
	const bars = 200
	s := NewSynth(benchS, bars)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		strat := &noopFullScanStrategy{}
		r := runner.New(strat,
			runner.WithPortfolio(portfolio.New(&portfolio.Settings{InitialCapital: 1e6, DefaultLeverage: 1})),
			runner.WithRiskManager(risk.New(&risk.Settings{})),
			runner.WithData(s.Table),
		)
		b.StartTimer()
		if err := r.Start(); err != nil {
			b.Fatal(err)
		}
		KeepAlive(strat.sink)
	}
	nsPerBar := float64(b.Elapsed().Nanoseconds()) / float64(b.N) / float64(bars)
	b.ReportMetric(nsPerBar, "ns/bar")
}

func reportPerBar(b *testing.B, totalBars int) {
	if totalBars == 0 {
		return
	}
	nsPerBar := float64(b.Elapsed().Nanoseconds()) / float64(totalBars)
	b.ReportMetric(nsPerBar, "ns/bar")
	b.ReportMetric(nsPerBar/float64(benchS), "ns/cell")
}

// silence unused import if interfaces needed later
var _ interfaces.Strategy = (*noopFullScanStrategy)(nil)
var _ *types.TimeseriesTable[core.Candle]
