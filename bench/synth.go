package bench

import (
	"fmt"
	"sync"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

// Synth holds a TimeseriesTable plus a dense SoA mirror of closes for A/B benches.
type Synth struct {
	Symbols []string
	Times   []time.Time
	Table   *types.TimeseriesTable[core.Candle]
	// Closes is row-major: closes[row*S+col]
	Closes []float64
	S      int
	B      int
}

var (
	synthCache   = map[string]*Synth{}
	synthCacheMu sync.Mutex
)

func cacheKey(symbols, bars int, withTable bool) string {
	t := 0
	if withTable {
		t = 1
	}
	return fmt.Sprintf("%d/%d/%d", symbols, bars, t)
}

// NewSynth builds (or returns cached) synthetic daily table with S symbols and B bars.
func NewSynth(symbols, bars int) *Synth {
	return getSynth(symbols, bars, true)
}

// NewSynthSoA builds dense closes only (no TimeseriesTable) for large-B SoA benches.
func NewSynthSoA(symbols, bars int) *Synth {
	return getSynth(symbols, bars, false)
}

func getSynth(symbols, bars int, withTable bool) *Synth {
	key := cacheKey(symbols, bars, withTable)
	synthCacheMu.Lock()
	defer synthCacheMu.Unlock()
	if s, ok := synthCache[key]; ok {
		return s
	}
	s := buildSynth(symbols, bars, withTable)
	synthCache[key] = s
	return s
}

func buildSynth(symbols, bars int, withTable bool) *Synth {
	cols := make([]string, symbols)
	for i := 0; i < symbols; i++ {
		cols[i] = fmt.Sprintf("S%04d", i)
	}
	times := make([]time.Time, bars)
	closes := make([]float64, bars*symbols)
	base := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

	var table *types.TimeseriesTable[core.Candle]
	if withTable {
		table = types.NewTimeseriesTable[core.Candle](cols)
	}

	for r := 0; r < bars; r++ {
		ts := base.AddDate(0, 0, r)
		times[r] = ts
		var row map[string]core.Candle
		if withTable {
			row = make(map[string]core.Candle, symbols)
		}
		for c := 0; c < symbols; c++ {
			px := 100.0 + float64(c)*0.01 + float64(r)*0.001
			closes[r*symbols+c] = px
			if withTable {
				row[cols[c]] = core.Candle{
					Time:   ts,
					Open:   px,
					High:   px * 1.01,
					Low:    px * 0.99,
					Close:  px,
					Volume: 1000,
				}
			}
		}
		if withTable {
			_ = table.AddRow(ts, row)
		}
	}
	return &Synth{
		Symbols: cols,
		Times:   times,
		Table:   table,
		Closes:  closes,
		S:       symbols,
		B:       bars,
	}
}

// RowCloses returns the dense close slice for row r (length S).
func (s *Synth) RowCloses(r int) []float64 {
	off := r * s.S
	return s.Closes[off : off+s.S]
}
