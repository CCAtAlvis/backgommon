package data

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
)

// JSONCandleFile is the standard single-symbol OHLCV JSON format.
// {"symbol":"SYM","candles":[{"date":"2006-01-02","open":1,"high":2,"low":0.5,"close":1.5,"volume":100}]}
type JSONCandleFile struct {
	Symbol  string         `json:"symbol"`
	Candles []JSONCandleRow `json:"candles"`
}

// JSONCandleRow is one OHLCV row in a JSONCandleFile.
type JSONCandleRow struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int64   `json:"volume"`
}

// LoadCandlesFromJSON reads one symbol's candles from a JSON file.
func LoadCandlesFromJSON(path string) (symbol string, candles []core.Candle, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("read file: %w", err)
	}

	var file JSONCandleFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return "", nil, fmt.Errorf("parse json: %w", err)
	}
	if file.Symbol == "" {
		return "", nil, fmt.Errorf("missing symbol in %s", path)
	}
	if len(file.Candles) == 0 {
		return "", nil, fmt.Errorf("no candles in %s", path)
	}

	candles = make([]core.Candle, 0, len(file.Candles))
	for _, c := range file.Candles {
		t, err := time.Parse("2006-01-02", c.Date)
		if err != nil {
			return "", nil, fmt.Errorf("parse date %q: %w", c.Date, err)
		}
		candles = append(candles, core.Candle{
			Time:   t,
			Open:   c.Open,
			High:   c.High,
			Low:    c.Low,
			Close:  c.Close,
			Volume: c.Volume,
		})
	}

	return file.Symbol, candles, nil
}
