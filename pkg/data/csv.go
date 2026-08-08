// Package data provides loaders for market data from JSON, CSV, and synthetic
// sources. Each loader produces []core.Candle slices that can be fed into a
// types.TimeseriesTable for backtesting.
package data

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
)

// LoadCandlesFromCSV reads OHLCV rows from a CSV file and returns a slice of
// Candle values sorted in file order (callers must ensure chronological order).
//
// Expected column headers (case-sensitive): date, open, high, low, close, and
// optionally volume. The date column must use the "2006-01-02" (YYYY-MM-DD)
// layout — other formats are not currently supported.
//
// The symbol parameter is required because CSV files do not embed it; it is
// attached to each returned Candle.
func LoadCandlesFromCSV(path, symbol string) ([]core.Candle, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required for csv load")
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("csv must have header and at least one row")
	}

	header := records[0]
	col := mapCSVHeader(header)

	candles := make([]core.Candle, 0, len(records)-1)
	for _, row := range records[1:] {
		c, err := parseCSVRow(row, col)
		if err != nil {
			return nil, err
		}
		candles = append(candles, c)
	}
	return candles, nil
}

func mapCSVHeader(header []string) map[string]int {
	col := make(map[string]int)
	for i, h := range header {
		col[h] = i
	}
	return col
}

func parseCSVRow(row []string, col map[string]int) (core.Candle, error) {
	get := func(name string) (string, error) {
		i, ok := col[name]
		if !ok || i >= len(row) {
			return "", fmt.Errorf("missing column %q", name)
		}
		return row[i], nil
	}

	dateStr, err := get("date")
	if err != nil {
		return core.Candle{}, err
	}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return core.Candle{}, fmt.Errorf("parse date %q: %w", dateStr, err)
	}

	open, err := strconv.ParseFloat(mustGet(row, col, "open"), 64)
	if err != nil {
		return core.Candle{}, fmt.Errorf("open: %w", err)
	}
	high, err := strconv.ParseFloat(mustGet(row, col, "high"), 64)
	if err != nil {
		return core.Candle{}, fmt.Errorf("high: %w", err)
	}
	low, err := strconv.ParseFloat(mustGet(row, col, "low"), 64)
	if err != nil {
		return core.Candle{}, fmt.Errorf("low: %w", err)
	}
	closePx, err := strconv.ParseFloat(mustGet(row, col, "close"), 64)
	if err != nil {
		return core.Candle{}, fmt.Errorf("close: %w", err)
	}

	var volume int64
	if i, ok := col["volume"]; ok && i < len(row) && row[i] != "" {
		volume, _ = strconv.ParseInt(row[i], 10, 64)
	}

	return core.Candle{
		Time:   t,
		Open:   open,
		High:   high,
		Low:    low,
		Close:  closePx,
		Volume: volume,
	}, nil
}

func mustGet(row []string, col map[string]int, name string) string {
	i, ok := col[name]
	if !ok || i >= len(row) {
		return ""
	}
	return row[i]
}
