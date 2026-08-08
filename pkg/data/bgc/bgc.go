// Package bgc implements a compact binary candle cache (BGC) with optional zstd compression.
//
// On-disk format v1 (little-endian), typically stored as *.bgc.zst:
//
//	magic "BGC1" | version u16 | flags u16
//	nSymbols u32 | nCandlesTotal u64
//	per symbol (caller should sort for determinism):
//	  nameLen u16 | name UTF-8
//	  nAttrs u16 | (attrNameLen u16 | attrName UTF-8 | float64)*
//	  n u32
//	  times int64[n]   // Unix nanoseconds UTC
//	  open, high, low, close float32[n]
//	  volume int64[n]
//
// Timestamps are UnixNano so daily and intraday bars share one format.
// Candle indicators are not stored; optional per-symbol float attrs (e.g. shares_outstanding)
// cover strategy-specific metadata.
package bgc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/types"
	"github.com/klauspost/compress/zstd"
)

const (
	magic   = "BGC1"
	version = uint16(1)

	// AttrSharesOutstanding is a conventional symbol attribute used by mcap strategies.
	AttrSharesOutstanding = "shares_outstanding"
)

// SymbolSeries is one instrument's OHLCV bars plus optional float attributes.
type SymbolSeries struct {
	Symbol string
	Attrs  map[string]float64
	Bars   []core.Candle
}

// WriteOptions configures WriteFile. Zero value uses defaults.
type WriteOptions struct {
	// CompressionLevel is passed to zstd (0 = default speed).
	CompressionLevel zstd.EncoderLevel
}

// LoadOptions configures ReadFile / LoadTable.
// Start/End filter bars when set (inclusive). Zero means unbounded.
type LoadOptions struct {
	Start time.Time
	End   time.Time
}

// Exists reports whether path exists and is non-empty.
func Exists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Size() > 0
}

// WriteFile encodes series to a zstd-compressed BGC file at path.
func WriteFile(path string, series []SymbolSeries, opts WriteOptions) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	raw, err := encode(series)
	if err != nil {
		return err
	}

	level := opts.CompressionLevel
	if level == 0 {
		level = zstd.SpeedDefault
	}
	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(level))
	if err != nil {
		return fmt.Errorf("zstd writer: %w", err)
	}
	compressed := enc.EncodeAll(raw, make([]byte, 0, len(raw)/2))
	enc.Close()

	if err := os.WriteFile(path, compressed, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// ReadFile decompresses and decodes a BGC file into symbol series.
// Bars outside LoadOptions.Start/End (when set) are omitted.
func ReadFile(path string, opts LoadOptions) ([]SymbolSeries, error) {
	compressed, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	dec, err := zstd.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf("zstd reader: %w", err)
	}
	defer dec.Close()

	raw, err := dec.DecodeAll(compressed, nil)
	if err != nil {
		return nil, fmt.Errorf("zstd decode: %w", err)
	}

	series, err := decode(raw)
	if err != nil {
		return nil, err
	}
	return filterSeries(series, opts), nil
}

// LoadTable reads a BGC file into a TimeseriesTable[core.Candle].
// Symbol attributes are not attached to candles; use ReadFile if attrs are needed.
func LoadTable(path string, opts LoadOptions) (*types.TimeseriesTable[core.Candle], error) {
	series, err := ReadFile(path, opts)
	if err != nil {
		return nil, err
	}
	return SeriesToTable(series)
}

// SeriesToTable builds a TimeseriesTable from in-memory symbol series.
func SeriesToTable(series []SymbolSeries) (*types.TimeseriesTable[core.Candle], error) {
	if len(series) == 0 {
		return nil, fmt.Errorf("no symbol series")
	}

	symbols := make([]string, 0, len(series))
	timeSet := make(map[time.Time]struct{})
	for _, s := range series {
		if s.Symbol == "" {
			return nil, fmt.Errorf("empty symbol name")
		}
		symbols = append(symbols, s.Symbol)
		for _, c := range s.Bars {
			timeSet[c.Time] = struct{}{}
		}
	}

	times := make([]time.Time, 0, len(timeSet))
	for t := range timeSet {
		times = append(times, t)
	}
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })

	table := types.NewTimeseriesTable[core.Candle](symbols)
	for _, t := range times {
		if err := table.CreateRow(t); err != nil {
			return nil, fmt.Errorf("create row %v: %w", t, err)
		}
	}

	for _, s := range series {
		for _, c := range s.Bars {
			if err := table.SetValue(c.Time, s.Symbol, c); err != nil {
				return nil, fmt.Errorf("set %s @ %v: %w", s.Symbol, c.Time, err)
			}
		}
	}
	return table, nil
}

func encode(series []SymbolSeries) ([]byte, error) {
	// Copy and sort for deterministic output without mutating caller.
	sorted := make([]SymbolSeries, len(series))
	copy(sorted, series)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Symbol < sorted[j].Symbol })

	var total uint64
	for _, s := range sorted {
		total += uint64(len(s.Bars))
	}

	buf := &bytes.Buffer{}
	if _, err := buf.WriteString(magic); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, version); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(0)); err != nil { // flags
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(len(sorted))); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, total); err != nil {
		return nil, err
	}

	for _, s := range sorted {
		if err := writeSymbol(buf, s); err != nil {
			return nil, fmt.Errorf("encode %s: %w", s.Symbol, err)
		}
	}
	return buf.Bytes(), nil
}

func writeSymbol(w io.Writer, s SymbolSeries) error {
	name := []byte(s.Symbol)
	if len(name) > 65535 {
		return fmt.Errorf("symbol name too long")
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(len(name))); err != nil {
		return err
	}
	if _, err := w.Write(name); err != nil {
		return err
	}

	attrKeys := make([]string, 0, len(s.Attrs))
	for k := range s.Attrs {
		attrKeys = append(attrKeys, k)
	}
	sort.Strings(attrKeys)
	if len(attrKeys) > 65535 {
		return fmt.Errorf("too many attrs")
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(len(attrKeys))); err != nil {
		return err
	}
	for _, k := range attrKeys {
		kb := []byte(k)
		if len(kb) > 65535 {
			return fmt.Errorf("attr name too long: %s", k)
		}
		if err := binary.Write(w, binary.LittleEndian, uint16(len(kb))); err != nil {
			return err
		}
		if _, err := w.Write(kb); err != nil {
			return err
		}
		if err := binary.Write(w, binary.LittleEndian, s.Attrs[k]); err != nil {
			return err
		}
	}

	n := uint32(len(s.Bars))
	if err := binary.Write(w, binary.LittleEndian, n); err != nil {
		return err
	}

	for _, c := range s.Bars {
		if err := binary.Write(w, binary.LittleEndian, c.Time.UTC().UnixNano()); err != nil {
			return err
		}
	}
	for _, c := range s.Bars {
		if err := binary.Write(w, binary.LittleEndian, float32(c.Open)); err != nil {
			return err
		}
	}
	for _, c := range s.Bars {
		if err := binary.Write(w, binary.LittleEndian, float32(c.High)); err != nil {
			return err
		}
	}
	for _, c := range s.Bars {
		if err := binary.Write(w, binary.LittleEndian, float32(c.Low)); err != nil {
			return err
		}
	}
	for _, c := range s.Bars {
		if err := binary.Write(w, binary.LittleEndian, float32(c.Close)); err != nil {
			return err
		}
	}
	for _, c := range s.Bars {
		if err := binary.Write(w, binary.LittleEndian, c.Volume); err != nil {
			return err
		}
	}
	return nil
}

func decode(raw []byte) ([]SymbolSeries, error) {
	r := bytes.NewReader(raw)
	magicBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, magicBuf); err != nil {
		return nil, fmt.Errorf("read magic: %w", err)
	}
	if string(magicBuf) != magic {
		return nil, fmt.Errorf("bad magic %q want %q", magicBuf, magic)
	}

	var ver, flags uint16
	if err := binary.Read(r, binary.LittleEndian, &ver); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &flags); err != nil {
		return nil, err
	}
	if ver != version {
		return nil, fmt.Errorf("unsupported version %d", ver)
	}
	_ = flags

	var nSymbols uint32
	var nCandlesTotal uint64
	if err := binary.Read(r, binary.LittleEndian, &nSymbols); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &nCandlesTotal); err != nil {
		return nil, err
	}

	out := make([]SymbolSeries, 0, nSymbols)
	var gotCandles uint64
	for i := uint32(0); i < nSymbols; i++ {
		s, err := readSymbol(r)
		if err != nil {
			return nil, fmt.Errorf("symbol %d: %w", i, err)
		}
		gotCandles += uint64(len(s.Bars))
		out = append(out, s)
	}
	if gotCandles != nCandlesTotal {
		return nil, fmt.Errorf("candle count mismatch: header %d got %d", nCandlesTotal, gotCandles)
	}
	if r.Len() != 0 {
		return nil, fmt.Errorf("trailing %d bytes", r.Len())
	}
	return out, nil
}

func readSymbol(r *bytes.Reader) (SymbolSeries, error) {
	var nameLen uint16
	if err := binary.Read(r, binary.LittleEndian, &nameLen); err != nil {
		return SymbolSeries{}, err
	}
	nameBuf := make([]byte, nameLen)
	if _, err := io.ReadFull(r, nameBuf); err != nil {
		return SymbolSeries{}, err
	}

	var nAttrs uint16
	if err := binary.Read(r, binary.LittleEndian, &nAttrs); err != nil {
		return SymbolSeries{}, err
	}
	attrs := make(map[string]float64, nAttrs)
	for i := uint16(0); i < nAttrs; i++ {
		var alen uint16
		if err := binary.Read(r, binary.LittleEndian, &alen); err != nil {
			return SymbolSeries{}, err
		}
		abuf := make([]byte, alen)
		if _, err := io.ReadFull(r, abuf); err != nil {
			return SymbolSeries{}, err
		}
		var v float64
		if err := binary.Read(r, binary.LittleEndian, &v); err != nil {
			return SymbolSeries{}, err
		}
		attrs[string(abuf)] = v
	}

	var n uint32
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return SymbolSeries{}, err
	}

	times := make([]int64, n)
	opens := make([]float32, n)
	highs := make([]float32, n)
	lows := make([]float32, n)
	closes := make([]float32, n)
	vols := make([]int64, n)

	for i := uint32(0); i < n; i++ {
		if err := binary.Read(r, binary.LittleEndian, &times[i]); err != nil {
			return SymbolSeries{}, err
		}
	}
	for i := uint32(0); i < n; i++ {
		if err := binary.Read(r, binary.LittleEndian, &opens[i]); err != nil {
			return SymbolSeries{}, err
		}
	}
	for i := uint32(0); i < n; i++ {
		if err := binary.Read(r, binary.LittleEndian, &highs[i]); err != nil {
			return SymbolSeries{}, err
		}
	}
	for i := uint32(0); i < n; i++ {
		if err := binary.Read(r, binary.LittleEndian, &lows[i]); err != nil {
			return SymbolSeries{}, err
		}
	}
	for i := uint32(0); i < n; i++ {
		if err := binary.Read(r, binary.LittleEndian, &closes[i]); err != nil {
			return SymbolSeries{}, err
		}
	}
	for i := uint32(0); i < n; i++ {
		if err := binary.Read(r, binary.LittleEndian, &vols[i]); err != nil {
			return SymbolSeries{}, err
		}
	}

	bars := make([]core.Candle, n)
	for i := uint32(0); i < n; i++ {
		bars[i] = core.Candle{
			Time:   time.Unix(0, times[i]).UTC(),
			Open:   float64(opens[i]),
			High:   float64(highs[i]),
			Low:    float64(lows[i]),
			Close:  float64(closes[i]),
			Volume: vols[i],
		}
	}

	return SymbolSeries{
		Symbol: string(nameBuf),
		Attrs:  attrs,
		Bars:   bars,
	}, nil
}

func filterSeries(series []SymbolSeries, opts LoadOptions) []SymbolSeries {
	if opts.Start.IsZero() && opts.End.IsZero() {
		return series
	}
	out := make([]SymbolSeries, 0, len(series))
	for _, s := range series {
		bars := make([]core.Candle, 0, len(s.Bars))
		for _, c := range s.Bars {
			if !opts.Start.IsZero() && c.Time.Before(opts.Start) {
				continue
			}
			if !opts.End.IsZero() && c.Time.After(opts.End) {
				continue
			}
			bars = append(bars, c)
		}
		if len(bars) == 0 {
			continue
		}
		out = append(out, SymbolSeries{Symbol: s.Symbol, Attrs: s.Attrs, Bars: bars})
	}
	return out
}
