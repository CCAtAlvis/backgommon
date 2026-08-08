package bench

import (
	"fmt"
	"testing"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

func BenchmarkE4_PositionsCopy_x3(b *testing.B) {
	p := openN(b, 20)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Mimic runner + risk + strategy each calling Positions().
		_ = p.Positions()
		_ = p.Positions()
		_ = p.Positions()
	}
}

func BenchmarkE4_ForEachOpen_x3(b *testing.B) {
	p := openN(b, 20)
	b.ReportAllocs()
	b.ResetTimer()
	var sink int
	for i := 0; i < b.N; i++ {
		for j := 0; j < 3; j++ {
			forEachOpen(p, func(pos *portfolio.Position) {
				sink += pos.Quantity
			})
		}
	}
	KeepAlive(float64(sink))
}

// forEachOpen is the E4 prototype: iterate without copying the map.
// Uses Positions() once then ranges — wait, that still copies.
// True zero-copy needs internal access; we simulate via a retained snapshot
// built once outside the hot path (interest-set style), OR range a shared slice.
func forEachOpen(p *portfolio.Portfolio, fn func(*portfolio.Position)) {
	// Prototype of ideal API: walk a pre-built slice of open positions.
	// Here we rebuild from Positions for correctness of count, but the
	// *benchmarked* path for zero-copy uses openSlice below.
	for _, pos := range p.Positions() {
		fn(pos)
	}
}

func BenchmarkE4_OpenSlice_x3(b *testing.B) {
	p := openN(b, 20)
	// Build slice once (what a zero-copy API would maintain).
	snap := p.Positions()
	slice := make([]*portfolio.Position, 0, len(snap))
	for _, pos := range snap {
		slice = append(slice, pos)
	}
	b.ReportAllocs()
	b.ResetTimer()
	var sink int
	for i := 0; i < b.N; i++ {
		for j := 0; j < 3; j++ {
			for _, pos := range slice {
				sink += pos.Quantity
			}
		}
	}
	KeepAlive(float64(sink))
}

func openN(b *testing.B, n int) *portfolio.Portfolio {
	b.Helper()
	p := portfolio.New(&portfolio.Settings{InitialCapital: 1e9, DefaultLeverage: 1})
	for i := 0; i < n; i++ {
		ord := portfolio.NewOrder(fmt.Sprintf("S%04d", i), portfolio.Long, portfolio.Entry, 10, 1)
		ord.Price = 100
		ord.FilledAt = time.Now()
		if err := p.ProcessOrder(ord); err != nil {
			b.Fatal(err)
		}
	}
	return p
}
