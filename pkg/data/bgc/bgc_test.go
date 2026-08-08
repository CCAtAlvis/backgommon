package bgc

import (
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
)

func TestRoundTripDaily(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "daily.bgc.zst")

	d0 := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	d1 := time.Date(2020, 1, 3, 0, 0, 0, 0, time.UTC)

	in := []SymbolSeries{
		{
			Symbol: "AAA-EQ",
			Attrs:  map[string]float64{AttrSharesOutstanding: 1e9},
			Bars: []core.Candle{
				{Time: d0, Open: 10, High: 11, Low: 9.5, Close: 10.5, Volume: 1000},
				{Time: d1, Open: 10.5, High: 12, Low: 10, Close: 11.5, Volume: 2000},
			},
		},
		{
			Symbol: "BBB-EQ",
			Attrs:  map[string]float64{AttrSharesOutstanding: 2e8},
			Bars: []core.Candle{
				{Time: d0, Open: 100, High: 101, Low: 99, Close: 100.5, Volume: 50},
			},
		},
	}

	if err := WriteFile(path, in, WriteOptions{}); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if !Exists(path) {
		t.Fatal("Exists expected true")
	}

	out, err := ReadFile(path, LoadOptions{})
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("symbols: got %d want 2", len(out))
	}
	// Sorted by symbol on write
	if out[0].Symbol != "AAA-EQ" || out[1].Symbol != "BBB-EQ" {
		t.Fatalf("symbol order: %+v", []string{out[0].Symbol, out[1].Symbol})
	}
	if out[0].Attrs[AttrSharesOutstanding] != 1e9 {
		t.Fatalf("shares attr: %v", out[0].Attrs)
	}
	assertCandleClose(t, out[0].Bars[0], d0, 10.5, 1000)
	assertCandleClose(t, out[0].Bars[1], d1, 11.5, 2000)

	table, err := LoadTable(path, LoadOptions{})
	if err != nil {
		t.Fatalf("LoadTable: %v", err)
	}
	c, ok := table.GetValue(d0, "AAA-EQ")
	if !ok || c.Close != float64(float32(10.5)) {
		t.Fatalf("table get: ok=%v close=%v", ok, c.Close)
	}
}

func TestRoundTripIntradayUnixNano(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "intraday.bgc.zst")

	base := time.Date(2024, 6, 1, 9, 15, 0, 0, time.UTC)
	bars := make([]core.Candle, 5)
	for i := range bars {
		bars[i] = core.Candle{
			Time:   base.Add(time.Duration(i) * time.Minute),
			Open:   float64(100 + i),
			High:   float64(101 + i),
			Low:    float64(99 + i),
			Close:  float64(100.5 + float64(i)),
			Volume: int64(1000 + i),
		}
	}

	in := []SymbolSeries{{Symbol: "INTRADAY-EQ", Bars: bars}}
	if err := WriteFile(path, in, WriteOptions{}); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	out, err := ReadFile(path, LoadOptions{})
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(out) != 1 || len(out[0].Bars) != 5 {
		t.Fatalf("got %+v", out)
	}
	for i, want := range bars {
		got := out[0].Bars[i]
		if !got.Time.Equal(want.Time) {
			t.Fatalf("bar %d time: got %v want %v", i, got.Time, want.Time)
		}
		if got.Time.UnixNano() != want.Time.UnixNano() {
			t.Fatalf("bar %d unixnano mismatch", i)
		}
		if math.Abs(got.Close-float64(float32(want.Close))) > 1e-5 {
			t.Fatalf("bar %d close: got %v want %v", i, got.Close, want.Close)
		}
	}
}

func TestLoadOptionsTimeWindow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "window.bgc.zst")

	t0 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2020, 1, 3, 0, 0, 0, 0, time.UTC)

	in := []SymbolSeries{{
		Symbol: "X-EQ",
		Bars: []core.Candle{
			{Time: t0, Close: 1, Volume: 1},
			{Time: t1, Close: 2, Volume: 2},
			{Time: t2, Close: 3, Volume: 3},
		},
	}}
	if err := WriteFile(path, in, WriteOptions{}); err != nil {
		t.Fatal(err)
	}

	out, err := ReadFile(path, LoadOptions{Start: t1, End: t1})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || len(out[0].Bars) != 1 || !out[0].Bars[0].Time.Equal(t1) {
		t.Fatalf("window filter failed: %+v", out)
	}
}

func assertCandleClose(t *testing.T, c core.Candle, wantTime time.Time, wantClose float64, wantVol int64) {
	t.Helper()
	if !c.Time.Equal(wantTime) {
		t.Fatalf("time: got %v want %v", c.Time, wantTime)
	}
	if math.Abs(c.Close-float64(float32(wantClose))) > 1e-5 {
		t.Fatalf("close: got %v want %v", c.Close, wantClose)
	}
	if c.Volume != wantVol {
		t.Fatalf("volume: got %d want %d", c.Volume, wantVol)
	}
}
