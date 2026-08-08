package data

import (
	"fmt"
	"strings"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

// LoadSingleSymbolTable builds a one-column TimeseriesTable from a slice of candles.
func LoadSingleSymbolTable(symbol string, candles []core.Candle) (*types.TimeseriesTable[core.Candle], error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	if len(candles) == 0 {
		return nil, fmt.Errorf("no candles provided")
	}

	table := types.NewTimeseriesTable[core.Candle]([]string{symbol})
	for _, c := range candles {
		if err := table.AddRow(c.Time, map[string]core.Candle{symbol: c}); err != nil {
			return nil, fmt.Errorf("add row at %v: %w", c.Time, err)
		}
	}
	return table, nil
}

// LoadTableFromJSON loads a single-symbol JSON OHLCV file into a TimeseriesTable.
func LoadTableFromJSON(path string) (*types.TimeseriesTable[core.Candle], string, error) {
	symbol, candles, err := LoadCandlesFromJSON(path)
	if err != nil {
		return nil, "", err
	}
	table, err := LoadSingleSymbolTable(symbol, candles)
	if err != nil {
		return nil, "", err
	}
	return table, symbol, nil
}

// LoadTableFromCSV loads a CSV OHLCV file into a TimeseriesTable.
func LoadTableFromCSV(path, symbol string) (*types.TimeseriesTable[core.Candle], error) {
	candles, err := LoadCandlesFromCSV(path, symbol)
	if err != nil {
		return nil, err
	}
	return LoadSingleSymbolTable(symbol, candles)
}

// LoadTableFromFile loads JSON or CSV based on file extension.
// For CSV, symbol must be provided (non-empty).
func LoadTableFromFile(path, symbol string) (*types.TimeseriesTable[core.Candle], string, error) {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".json"):
		return LoadTableFromJSON(path)
	case strings.HasSuffix(lower, ".csv"):
		if symbol == "" {
			return nil, "", fmt.Errorf("symbol is required for csv files")
		}
		table, err := LoadTableFromCSV(path, symbol)
		if err != nil {
			return nil, "", err
		}
		return table, symbol, nil
	default:
		return nil, "", fmt.Errorf("unsupported file type %q (use .json or .csv)", path)
	}
}
