package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"testing"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/output"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

type impactRow struct {
	ID             string
	Name           string
	NsPerBar       float64
	AllocsPerBar   float64 // approximate; 0 if unknown
	Proj10k        time.Duration
	Proj1M         time.Duration
	MemMB          float64
	Recommendation string
	Notes          string
}

func measureBars(bars int, fn func()) (elapsed time.Duration, nsPerBar float64) {
	runtime.GC()
	start := time.Now()
	fn()
	elapsed = time.Since(start)
	nsPerBar = float64(elapsed.Nanoseconds()) / float64(bars)
	return elapsed, nsPerBar
}

func proj(nsPerBar float64, bars int) time.Duration {
	return time.Duration(nsPerBar * float64(bars))
}

// TestImpactReport runs timed kernels once each and writes bench/RESULTS.md.
// Run: go test ./bench -run TestImpactReport -count=1 -timeout 60m
func TestImpactReport(t *testing.T) {
	const S = 2000
	const B = 500 // table-backed; enough for stable ns/bar, extrapolate up

	t.Log("building synthetic table (cached)...")
	synth := NewSynth(S, B)
	soa := NewSynthSoA(S, B)
	soa10k := NewSynthSoA(S, 10_000)

	var rows []impactRow

	// E0 FillRow
	{
		dst := make(map[string]core.Candle, S)
		_, ns := measureBars(B, func() {
			for r := 0; r < B; r++ {
				_ = synth.Table.FillRow(synth.Times[r], dst)
			}
		})
		rows = append(rows, impactRow{
			ID: "E0a", Name: "FillRow (reuse map)",
			NsPerBar: ns, Proj10k: proj(ns, 10_000), Proj1M: proj(ns, 1_000_000),
			Recommendation: "baseline",
			Notes:          "current runner materializes full universe every bar",
		})
	}

	// E0 Dual FillRow
	{
		cur := make(map[string]core.Candle, S)
		nxt := make(map[string]core.Candle, S)
		bars := B - 1
		_, ns := measureBars(bars, func() {
			for r := 0; r < bars; r++ {
				_ = synth.Table.FillRow(synth.Times[r], cur)
				_ = synth.Table.FillRow(synth.Times[r+1], nxt)
			}
		})
		rows = append(rows, impactRow{
			ID: "E0b", Name: "Dual FillRow (cur+next)",
			NsPerBar: ns, Proj10k: proj(ns, 10_000), Proj1M: proj(ns, 1_000_000),
			Recommendation: "baseline",
			Notes:          "mirrors Runner.Start today",
		})
	}

	// E0 GetValue all
	{
		_, ns := measureBars(B, func() {
			for r := 0; r < B; r++ {
				ts := synth.Times[r]
				for _, col := range synth.Symbols {
					c, ok := synth.Table.GetValue(ts, col)
					if ok {
						_ = c.Close
					}
				}
			}
		})
		rows = append(rows, impactRow{
			ID: "E0c", Name: "GetValue × S (MarketView-style on Table)",
			NsPerBar: ns, Proj10k: proj(ns, 10_000), Proj1M: proj(ns, 1_000_000),
			Recommendation: "defer as sole path",
			Notes:          "avoids map build but still O(S) interface asserts — slower than SoA",
		})
	}

	// E0 FillRow+scan
	{
		dst := make(map[string]core.Candle, S)
		_, ns := measureBars(B, func() {
			var sink float64
			for r := 0; r < B; r++ {
				_ = synth.Table.FillRow(synth.Times[r], dst)
				for _, c := range dst {
					sink += c.Close
				}
			}
			KeepAlive(sink)
		})
		rows = append(rows, impactRow{
			ID: "E0d", Name: "FillRow + scan map closes (W0 today)",
			NsPerBar: ns, Proj10k: proj(ns, 10_000), Proj1M: proj(ns, 1_000_000),
			Recommendation: "replace",
			Notes:          "full-universe touch via maps",
		})
	}

	// E1 lazy next
	{
		out := make(map[string]float64, 20)
		syms := synth.Symbols[:20]
		bars := B - 1
		_, ns := measureBars(bars, func() {
			for r := 0; r < bars; r++ {
				LazyNextCloses(synth.Table, synth.Times[r+1], syms, out)
			}
		})
		rows = append(rows, impactRow{
			ID: "E1", Name: "Lazy next GetValue × 20 orders",
			NsPerBar: ns, Proj10k: proj(ns, 10_000), Proj1M: proj(ns, 1_000_000),
			Recommendation: "implement",
			Notes:          "vs full next FillRow; huge when orders rare",
		})
	}

	// E2 MarketView sum
	{
		cols := synth.Symbols
		_, ns := measureBars(B, func() {
			var sink float64
			for r := 0; r < B; r++ {
				v := MarketView{Table: synth.Table, Ts: synth.Times[r], Cols: cols}
				sink += v.SumCloses()
			}
			KeepAlive(sink)
		})
		rows = append(rows, impactRow{
			ID: "E2", Name: "MarketView SumCloses (GetValue loop)",
			NsPerBar: ns, Proj10k: proj(ns, 10_000), Proj1M: proj(ns, 1_000_000),
			Recommendation: "implement with SoA backing",
			Notes:          "API shape good; backing must be dense for moonshot",
		})
	}

	// E3 SoA
	{
		_, ns := measureBars(B, func() {
			var sink float64
			for r := 0; r < B; r++ {
				for _, px := range soa.RowCloses(r) {
					sink += px
				}
			}
			KeepAlive(sink)
		})
		rows = append(rows, impactRow{
			ID: "E3a", Name: "SoA sum closes (dense float64)",
			NsPerBar: ns, Proj10k: proj(ns, 10_000), Proj1M: proj(ns, 1_000_000),
			MemMB:          float64(S*B*8) / 1e6,
			Recommendation: "implement (must-have)",
			Notes:          "main lever for full-universe-every-bar toward moonshot B",
		})
	}
	{
		const bars = 10_000
		_, ns := measureBars(bars, func() {
			var sink float64
			for r := 0; r < bars; r++ {
				for _, px := range soa10k.RowCloses(r) {
					sink += px
				}
			}
			KeepAlive(sink)
		})
		rows = append(rows, impactRow{
			ID: "E3b", Name: "SoA sum closes @10k bars measured",
			NsPerBar: ns, Proj10k: proj(ns, 10_000), Proj1M: proj(ns, 1_000_000),
			MemMB:          float64(S*bars*8) / 1e6,
			Recommendation: "implement (must-have)",
			Notes:          "measured wall for 10k; 1M extrapolated",
		})
	}

	// E4 positions
	{
		p := portfolio.New(&portfolio.Settings{InitialCapital: 1e9, DefaultLeverage: 1})
		for i := 0; i < 20; i++ {
			ord := portfolio.NewOrder(fmt.Sprintf("S%04d", i), portfolio.Long, portfolio.Entry, 10, 1)
			ord.Price = 100
			ord.FilledAt = time.Now()
			_ = p.ProcessOrder(ord)
		}
		snap := p.Positions()
		slice := make([]*portfolio.Position, 0, len(snap))
		for _, pos := range snap {
			slice = append(slice, pos)
		}
		const iters = 100_000
		_, nsCopy := measureBars(iters, func() {
			for i := 0; i < iters; i++ {
				_ = p.Positions()
				_ = p.Positions()
				_ = p.Positions()
			}
		})
		_, nsSlice := measureBars(iters, func() {
			var sink int
			for i := 0; i < iters; i++ {
				for j := 0; j < 3; j++ {
					for _, pos := range slice {
						sink += pos.Quantity
					}
				}
			}
			KeepAlive(float64(sink))
		})
		rows = append(rows, impactRow{
			ID: "E4a", Name: "Positions() copy ×3 / bar-equivalent",
			NsPerBar: nsCopy, Proj10k: proj(nsCopy, 10_000), Proj1M: proj(nsCopy, 1_000_000),
			Recommendation: "baseline",
			Notes:          "allocating map copy",
		})
		rows = append(rows, impactRow{
			ID: "E4b", Name: "Open slice iterate ×3 (zero-copy proto)",
			NsPerBar: nsSlice, Proj10k: proj(nsSlice, 10_000), Proj1M: proj(nsSlice, 1_000_000),
			Recommendation: "implement",
			Notes:          "ForEachOpenPosition / maintained slice",
		})
	}

	// E5 equity
	{
		instruments := synth.Symbols[:20]
		prices := make(map[string]float64, 20)
		for _, inst := range instruments {
			prices[inst] = 100
		}
		const iters = 200_000
		_, nsFull := measureBars(iters, func() {
			var sink float64
			for i := 0; i < iters; i++ {
				v, _, _ := EquityFull(1e6, 1e5, instruments, prices)
				sink += v
			}
			KeepAlive(sink)
		})
		_, nsVal := measureBars(iters, func() {
			var sink float64
			for i := 0; i < iters; i++ {
				sink += EquityValueOnly(1e6, 1e5)
			}
			KeepAlive(sink)
		})
		rows = append(rows, impactRow{
			ID: "E5a", Name: "Equity Full holdings snapshot",
			NsPerBar: nsFull, Proj10k: proj(nsFull, 10_000), Proj1M: proj(nsFull, 1_000_000),
			Recommendation: "baseline",
		})
		rows = append(rows, impactRow{
			ID: "E5b", Name: "Equity ValueOnly",
			NsPerBar: nsVal, Proj10k: proj(nsVal, 10_000), Proj1M: proj(nsVal, 1_000_000),
			Recommendation: "implement dial",
			Notes:          "WithEquityMode(ValueOnly|EveryN|Full)",
		})
	}

	// E6 export (one-shot wall times, not ns/bar)
	{
		data := fakeReport(5000, 20)
		dir := t.TempDir()
		t0 := time.Now()
		if err := output.ExportRunTo(filepath.Join(dir, "full"), data); err != nil {
			t.Fatal(err)
		}
		fullDur := time.Since(t0)
		t0 = time.Now()
		metricsDir := filepath.Join(dir, "metrics")
		_ = os.MkdirAll(metricsDir, 0755)
		raw, _ := json.Marshal(map[string]interface{}{
			"sharpe": data.Results.SharpeRatio, "cagr": data.Results.CAGR,
		})
		_ = os.WriteFile(filepath.Join(metricsDir, "metrics.json"), raw, 0644)
		metricsDur := time.Since(t0)
		rows = append(rows, impactRow{
			ID: "E6a", Name: "Export full HTML+JSON (5k equity pts)",
			NsPerBar: float64(fullDur.Nanoseconds()), Recommendation: "baseline",
			Notes: fmt.Sprintf("wall=%s (~35MB in go bench)", formatDur(fullDur)),
		})
		rows = append(rows, impactRow{
			ID: "E6b", Name: "Export metrics-only JSON",
			NsPerBar: float64(metricsDur.Nanoseconds()), Recommendation: "implement",
			Notes: fmt.Sprintf("wall=%s; ~140× cheaper than full in go bench", formatDur(metricsDur)),
		})
	}

	// E7 GC
	{
		runW1 := func() float64 {
			cur := make(map[string]core.Candle, S)
			prices := make(map[string]float64, S)
			instruments := synth.Symbols[:20]
			_, ns := measureBars(B, func() {
				var sink float64
				for r := 0; r < B; r++ {
					_ = synth.Table.FillRow(synth.Times[r], cur)
					FillPrices(cur, prices)
					v, _, _ := EquityFull(1e6, 1e5, instruments, prices)
					sink += v
				}
				KeepAlive(sink)
			})
			return ns
		}
		nsDef := runW1()
		old := debug.SetGCPercent(-1)
		nsOff := runW1()
		debug.SetGCPercent(old)
		rows = append(rows, impactRow{
			ID: "E7a", Name: "W1 FillRow+prices+equity (default GC)",
			NsPerBar: nsDef, Proj10k: proj(nsDef, 10_000), Proj1M: proj(nsDef, 1_000_000),
			Recommendation: "baseline",
		})
		rows = append(rows, impactRow{
			ID: "E7b", Name: "W1 same with GC percent=-1",
			NsPerBar: nsOff, Proj10k: proj(nsOff, 10_000), Proj1M: proj(nsOff, 1_000_000),
			Recommendation: "situational",
			Notes:          "CLI batch only; RSS unbounded; not a library default",
		})
	}

	// E9 feature panel
	{
		const lookback = 120
		panel := NewFeaturePanel(B, S, 1)
		PrecomputeLookbackReturn(panel, soa.Closes, S, lookback, 0)
		_, nsRead := measureBars(B, func() {
			var sink float64
			for t := 0; t < B; t++ {
				var sum float64
				for c := 0; c < S; c++ {
					sum += panel.At(t, c, 0)
				}
				sink += sum
			}
			KeepAlive(sink)
		})
		_, nsFly := measureBars(B, func() {
			var sink float64
			for t := 0; t < B; t++ {
				sink += OnTheFlyLookbackReturn(soa.Closes, S, t, lookback)
			}
			KeepAlive(sink)
		})
		buildStart := time.Now()
		p2 := NewFeaturePanel(B, S, 1)
		PrecomputeLookbackReturn(p2, soa.Closes, S, lookback, 0)
		buildDur := time.Since(buildStart)
		rows = append(rows, impactRow{
			ID: "E9a", Name: "Precomputed feature read (sum all)",
			NsPerBar: nsRead, Proj10k: proj(nsRead, 10_000), Proj1M: proj(nsRead, 1_000_000),
			MemMB:          float64(B*S*8) / 1e6,
			Recommendation: "implement as FeaturePanel",
			Notes:          fmt.Sprintf("build(%d bars)=%s; wins when feature is expensive; 1M×2000×1f64≈16GB", B, buildDur),
		})
		rows = append(rows, impactRow{
			ID: "E9c", Name: "On-the-fly lookback return (sum)",
			NsPerBar: nsFly, Proj10k: proj(nsFly, 10_000), Proj1M: proj(nsFly, 1_000_000),
			Recommendation: "baseline for E9",
			Notes:          "similar to E9a for cheap lookbacks on SoA; use panel for rank/ATR/sort",
		})
	}

	// E10 parallel
	{
		workers := runtime.GOMAXPROCS(0)
		_, nsSerial := measureBars(B, func() {
			var sink float64
			for r := 0; r < B; r++ {
				for _, px := range soa.RowCloses(r) {
					sink += px
				}
			}
			KeepAlive(sink)
		})
		_, nsPar := measureBars(B, func() {
			var sink float64
			for r := 0; r < B; r++ {
				sink += SumRowParallel(soa.RowCloses(r), workers)
			}
			KeepAlive(sink)
		})
		rows = append(rows, impactRow{
			ID: "E10a", Name: "SoA sum serial",
			NsPerBar: nsSerial, Proj10k: proj(nsSerial, 10_000), Proj1M: proj(nsSerial, 1_000_000),
			Recommendation: "baseline",
		})
		rows = append(rows, impactRow{
			ID: "E10b", Name: fmt.Sprintf("SoA sum parallel (%d workers)", workers),
			NsPerBar: nsPar, Proj10k: proj(nsPar, 10_000), Proj1M: proj(nsPar, 1_000_000),
			Recommendation: "defer for per-bar",
			Notes:          "often slower at S=2000 due to goroutine overhead; better for multi-bar batches",
		})
	}

	// E8 pool vs reuse (single bar fill prices)
	{
		data := make(map[string]core.Candle, S)
		_ = synth.Table.FillRow(synth.Times[0], data)
		const iters = 5_000
		_, nsReuse := measureBars(iters, func() {
			prices := make(map[string]float64, S)
			for i := 0; i < iters; i++ {
				FillPrices(data, prices)
			}
		})
		_, nsPool := measureBars(iters, func() {
			for i := 0; i < iters; i++ {
				prices := BorrowPriceMap()
				FillPrices(data, prices)
				ReturnPriceMap(prices)
			}
		})
		_, nsNew := measureBars(iters, func() {
			for i := 0; i < iters; i++ {
				prices := make(map[string]float64, S)
				FillPrices(data, prices)
			}
		})
		rows = append(rows, impactRow{
			ID: "E8a", Name: "fillPrices reused map",
			NsPerBar: nsReuse, Recommendation: "keep if maps remain",
		})
		rows = append(rows, impactRow{
			ID: "E8b", Name: "fillPrices sync.Pool",
			NsPerBar: nsPool, Recommendation: "reject if SoA lands",
			Notes: "pool overhead vs reuse",
		})
		rows = append(rows, impactRow{
			ID: "E8c", Name: "fillPrices new map each bar",
			NsPerBar: nsNew, Recommendation: "reject",
		})
	}

	md := renderImpactMarkdown(rows, S, B)
	outPath := filepath.Join(".", "RESULTS.generated.md")
	if err := os.WriteFile(outPath, []byte(md), 0644); err != nil {
		alt := filepath.Join("bench", "RESULTS.generated.md")
		if err2 := os.WriteFile(alt, []byte(md), 0644); err2 != nil {
			t.Fatalf("write RESULTS.generated.md: %v / %v", err, err2)
		}
		outPath = alt
	}
	t.Logf("wrote %s (see also RESULTS.md for curated campaign report)", outPath)
	t.Log("\n" + md)
}

func renderImpactMarkdown(rows []impactRow, S, B int) string {
	var b string
	b += "# Framework performance impact report\n\n"
	b += fmt.Sprintf("Generated by `go test ./bench -run TestImpactReport`.\n\n")
	b += fmt.Sprintf("- Machine GOMAXPROCS=%d\n", runtime.GOMAXPROCS(0))
	b += fmt.Sprintf("- Synthetic grid: S=%d symbols, timed at B=%d bars (extrapolate to 10k / 1M)\n", S, B)
	b += "- Stretch goal: 1M bars × ~2000 symbols under 1s (ideal <100ms)\n"
	b += "- **Projections are linear** from ns/bar; GC and memory may make 1M nonlinear.\n\n"
	b += "## Results\n\n"
	b += "| ID | Experiment | ns/bar | proj 10k | proj 1M | mem | Recommendation | Notes |\n"
	b += "|----|------------|--------|----------|---------|-----|----------------|-------|\n"
	for _, r := range rows {
		mem := "-"
		if r.MemMB > 0 {
			mem = fmt.Sprintf("%.1f MB", r.MemMB)
		}
		b += fmt.Sprintf("| %s | %s | %.0f | %s | %s | %s | **%s** | %s |\n",
			r.ID, r.Name, r.NsPerBar, formatDur(r.Proj10k), formatDur(r.Proj1M), mem, r.Recommendation, r.Notes)
	}
	b += "\n## Decision summary\n\n"
	b += decisionSummary(rows)
	b += "\n## Moonshot distance\n\n"
	b += moonshotBlurb(rows)
	b += "\n## How to reproduce\n\n"
	b += "```bash\n"
	b += "cd backgommon\n"
	b += "go test ./bench -run TestImpactReport -count=1 -timeout 60m -v\n"
	b += "go test ./bench -bench=. -benchmem -count=1 -timeout 60m\n"
	b += "```\n"
	return b
}

func formatDur(d time.Duration) string {
	if d <= 0 {
		return "-"
	}
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.1fµs", float64(d.Nanoseconds())/1e3)
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1e6)
	}
	if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	return d.Round(time.Millisecond).String()
}

func decisionSummary(rows []impactRow) string {
	return "" +
		"- **Must implement for full-universe moonshot:** E3 SoA dense panels + E2 MarketView API (backed by SoA, not GetValue).\n" +
		"- **Clear framework wins:** E1 lazy next-bar, E4 zero-copy positions, E5 equity dial, E6 metrics-only export (see go benches).\n" +
		"- **Situational:** E7 GC off for short CLI sweeps only; E9 FeaturePanel when features stable across trials.\n" +
		"- **Defer/reject as primary:** E8 map pooling if SoA lands; E10 per-bar parallel at S=2000 (overhead); E11–E14 micro-opts after E3.\n"
}

func moonshotBlurb(rows []impactRow) string {
	var soaNs float64
	var mapNs float64
	for _, r := range rows {
		if r.ID == "E3a" {
			soaNs = r.NsPerBar
		}
		if r.ID == "E0d" {
			mapNs = r.NsPerBar
		}
	}
	s := fmt.Sprintf("- Map-path W0 (E0d): ~%.0f ns/bar → 1M bars ≈ %s\n", mapNs, formatDur(proj(mapNs, 1_000_000)))
	s += fmt.Sprintf("- SoA sum (E3a): ~%.0f ns/bar → 1M bars ≈ %s\n", soaNs, formatDur(proj(soaNs, 1_000_000)))
	if soaNs > 0 {
		budget1s := 1e9 / soaNs
		s += fmt.Sprintf("- At SoA scan rate, bars fitting in 1s (scan-only, no portfolio): ~%.0f (need %.0fx more for 1M in 1s)\n",
			budget1s, 1_000_000/budget1s)
		s += "- Closing the gap further needs: batch multi-bar kernels, SIMD, and/or not scanning all symbols every bar.\n"
	}
	return s
}
