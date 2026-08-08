package bench

import (
	"runtime"
	"sync"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

// MarketView is a bench-only row view: no map[string]Candle materialization.
type MarketView struct {
	Table *types.TimeseriesTable[core.Candle]
	Ts    time.Time
	Cols  []string
}

func (v MarketView) Candle(sym string) (core.Candle, bool) {
	return v.Table.GetValue(v.Ts, sym)
}

func (v MarketView) CloseAt(col int) float64 {
	c, ok := v.Table.GetValue(v.Ts, v.Cols[col])
	if !ok {
		return 0
	}
	return c.Close
}

func (v MarketView) SumCloses() float64 {
	var sum float64
	for _, col := range v.Cols {
		c, ok := v.Table.GetValue(v.Ts, col)
		if ok {
			sum += c.Close
		}
	}
	return sum
}

// FeaturePanel is a dense B×S×F float matrix (row-major: ((t*S)+col)*F + feat).
type FeaturePanel struct {
	B, S, F int
	Data    []float64
}

func NewFeaturePanel(b, s, f int) *FeaturePanel {
	return &FeaturePanel{B: b, S: s, F: f, Data: make([]float64, b*s*f)}
}

func (p *FeaturePanel) At(t, col, feat int) float64 {
	return p.Data[((t*p.S)+col)*p.F+feat]
}

func (p *FeaturePanel) Set(t, col, feat int, v float64) {
	p.Data[((t*p.S)+col)*p.F+feat] = v
}

// PrecomputeLookbackReturn fills feature 0 as (close[t]-close[t-lb])/close[t-lb].
func PrecomputeLookbackReturn(panel *FeaturePanel, closes []float64, s, lookback, feat int) {
	for t := 0; t < panel.B; t++ {
		src := t - lookback
		if src < 0 {
			src = 0
		}
		for c := 0; c < s; c++ {
			from := closes[src*s+c]
			to := closes[t*s+c]
			if from == 0 {
				panel.Set(t, c, feat, 0)
				continue
			}
			panel.Set(t, c, feat, (to-from)/from)
		}
	}
}

// OnTheFlyLookbackReturn computes the same feature for one bar without a panel.
func OnTheFlyLookbackReturn(closes []float64, s, t, lookback int) float64 {
	src := t - lookback
	if src < 0 {
		src = 0
	}
	var sum float64
	for c := 0; c < s; c++ {
		from := closes[src*s+c]
		to := closes[t*s+c]
		if from == 0 {
			continue
		}
		sum += (to - from) / from
	}
	return sum
}

// SumRowParallel sums a dense row using workers goroutines.
func SumRowParallel(row []float64, workers int) float64 {
	if workers < 1 {
		workers = 1
	}
	n := len(row)
	if workers > n {
		workers = n
	}
	chunk := (n + workers - 1) / workers
	partials := make([]float64, workers)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		start := w * chunk
		end := start + chunk
		if end > n {
			end = n
		}
		go func(id, lo, hi int) {
			defer wg.Done()
			var s float64
			for i := lo; i < hi; i++ {
				s += row[i]
			}
			partials[id] = s
		}(w, start, end)
	}
	wg.Wait()
	var sum float64
	for _, p := range partials {
		sum += p
	}
	return sum
}

// FillPrices copies closes from a candle map into prices (mirrors runner).
func FillPrices(data map[string]core.Candle, prices map[string]float64) {
	for symbol, candle := range data {
		prices[symbol] = candle.Close
	}
}

// LazyNextCloses fetches next-bar closes only for the given symbols.
func LazyNextCloses(table *types.TimeseriesTable[core.Candle], nextTs time.Time, syms []string, out map[string]float64) {
	for _, sym := range syms {
		c, ok := table.GetValue(nextTs, sym)
		if ok {
			out[sym] = c.Close
		}
	}
}

// EquityFull snapshots value + holdings (allocating).
func EquityFull(value, cash float64, instruments []string, prices map[string]float64) (float64, []string, []float64) {
	holdInstr := make([]string, 0, len(instruments))
	holdPx := make([]float64, 0, len(instruments))
	for _, inst := range instruments {
		if px, ok := prices[inst]; ok {
			holdInstr = append(holdInstr, inst)
			holdPx = append(holdPx, px)
		}
	}
	return value + cash, holdInstr, holdPx
}

// EquityValueOnly returns just the scalar equity.
func EquityValueOnly(value, cash float64) float64 {
	return value + cash
}

// MapPool is a sync.Pool of map[string]float64 for E8.
var MapPool = sync.Pool{
	New: func() interface{} {
		return make(map[string]float64, 64)
	},
}

func BorrowPriceMap() map[string]float64 {
	return MapPool.Get().(map[string]float64)
}

func ReturnPriceMap(m map[string]float64) {
	for k := range m {
		delete(m, k)
	}
	MapPool.Put(m)
}

// KeepAlive prevents the compiler from optimizing away sink values.
func KeepAlive(v float64) {
	runtime.KeepAlive(v)
}
