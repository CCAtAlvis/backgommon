# Loading market data

Backgommon does not ship market data. You bring OHLCV files; the framework loads them into a **`TimeseriesTable`** — the structure the runner walks bar-by-bar.

This is **not** tied to NSE, Fyers, or any broker. The JSON shape used in this repo is a **generic single-symbol format**; CSV is supported with a standard header.

---

## The journey: from file to backtest

```
  prices.json / prices.csv
           │
           ▼
    pkg/data loader
           │
           ▼
  TimeseriesTable[core.Candle]
    columns = symbols
    rows    = timestamps
           │
           ▼
    runner.WithData(table)
```

---

## JSON format (single symbol per file)

One file = one symbol. The symbol name is **inside** the file.

```json
{
  "symbol": "EXCHANGE:SYMBOL-EQ",
  "candles": [
    {
      "date": "2024-01-02",
      "open": 100,
      "high": 105,
      "low": 99,
      "close": 102,
      "volume": 1000
    }
  ]
}
```

| Field | Required | Notes |
|-------|----------|--------|
| `symbol` | yes | Any string; becomes the table column name |
| `candles` | yes | Array of bars, sorted by date recommended |
| `date` | yes | `YYYY-MM-DD` |
| `open`, `high`, `low`, `close` | yes | Numbers |
| `volume` | no | Defaults to 0 |

### Load JSON in code

```go
table, symbol, err := data.LoadTableFromJSON("path/to/prices.json")
```

Or only candles:

```go
symbol, candles, err := data.LoadCandlesFromJSON("path/to/prices.json")
table, err := data.LoadSingleSymbolTable(symbol, candles)
```

---

## CSV format (single symbol per file)

You pass the **symbol name** separately — CSV has no symbol column.

**Header (required):**

```text
date,open,high,low,close,volume
```

`volume` is optional.

```go
table, err := data.LoadTableFromCSV("path/to/prices.csv", "MY-SYMBOL")
```

---

## Auto-detect by extension

```go
table, symbol, err := data.LoadTableFromFile("prices.json", "")
table, err := data.LoadTableFromCSV("prices.csv", "MY-SYMBOL")
// LoadTableFromFile for .csv requires non-empty symbol flag
table, symbol, err := data.LoadTableFromFile("prices.csv", "MY-SYMBOL")
```

---

## Multi-symbol tables

For a portfolio backtest, one row = one timestamp, one column per symbol:

```go
table := types.NewTimeseriesTable[core.Candle]([]string{"AAA", "BBB", "CCC"})

table.AddRow(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), map[string]core.Candle{
    "AAA": {Time: t, Open: 10, High: 11, Low: 9, Close: 10.5, Volume: 100},
    "BBB": { ... },
})
```

Load each symbol from its own JSON file and merge rows, or build from your database — `pkg/data` helpers are for the common single-file case.

---

## Synthetic data (examples and tests)

For quick runs without a file:

```go
candles := data.GenerateSyntheticTrend(data.DefaultSyntheticTrendConfig())
table, err := data.LoadSingleSymbolTable("SYNTH", candles)
```

---

## CLI example

```bash
go run ./examples/strategies/sma_crossover/ -data ./my/prices.json
go run ./examples/strategies/sma_crossover/ -data ./my/prices.csv -symbol MY-SYM
```

---

## What’s on each `core.Candle`

Besides OHLCV, candles can store **indicator values** after you call `table.ApplyIndicators(...)`. Your strategy reads them with `candle.GetIndicator("SMA_20")`. See [Indicators](./indicators-application.md).

---

## Next steps

- [Getting started](./getting-started.md) — wire table into runner  
- [Indicators](./indicators-application.md) — attach SMA/MACD before `Start()`  
- [Writing a strategy](./writing-a-strategy.md) — read candles in `OnTick`
