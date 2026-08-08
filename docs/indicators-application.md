# Indicators: where and how to compute them

Indicators (SMA, EMA, MACD, custom) can be computed in two places. **For backtests, pre-compute on the table.** For live/streaming (future), compute inside the strategy.

This guide walks through both paths and when to use each.

---

## The journey: indicator reaches `OnTick`

```
Historical candles in TimeseriesTable
           │
           ▼
  ApplyIndicators([SMA, MACD, ...])     ←── once before backtest
           │
           ▼
  Each core.Candle stores values by name ("SMA_50", ...)
           │
           ▼
  Strategy.OnTick reads candle.GetIndicator("SMA_50")
```

You do **not** recompute the full series inside `OnTick` unless you have a good reason (streaming, experimental).

---

## Path 1: Table-level (recommended for backtests)

### Option A — runner applies them

```go
r := runner.New(
    strat,
    runner.WithData(table),
    runner.WithIndicators(strat.Indicators()),
    // ...
)
// Start() calls table.ApplyIndicators internally
```

### Option B — you apply manually

```go
err := table.ApplyIndicators([]interfaces.Indicator{
    indicators.NewSMA(20),
    indicators.NewMACD(12, 26, 9),
})
r := runner.New(strat, runner.WithData(table), ...)
```

MACD automatically pulls in its EMA dependencies via `Dependencies()`.

### Reading values in the strategy

```go
raw, err := candle.GetIndicator(sma20.Name()) // e.g. "SMA_20"
if err != nil {
    return nil // not enough history on this bar yet
}
value, ok := raw.(float64)
```

`Calculate()` returns `[]any` — one value per input candle, with `nil` during warm-up. The table stores the value **for that bar** on the candle.

---

## Path 2: Strategy-level (streaming / advanced)

```go
func (s *MyStrategy) OnTick(data map[string]core.Candle) []portfolio.Order {
    history := s.window // you maintain []core.Candle
    values := s.sma.Calculate(history)
    last := values[len(values)-1]
    // ...
}
```

**Pros:** Works without full history in memory.  
**Cons:** You manage window size, dependency order, and performance.

Use for live adapters later; for research backtests prefer Path 1.

---

## Custom indicators

```go
custom := indicators.NewCustomIndicator("Momentum", func(candles []core.Candle) []any {
    out := make([]any, len(candles))
    for i := range candles {
        if i < 10 {
            continue
        }
        out[i] = candles[i].Close - candles[i-10].Close
    }
    return out
}, nil)
```

See [`examples/indicator_strategy/`](../examples/indicator_strategy/) for embedding custom indicators in a strategy struct.

---

## Direct calculation (no backtest)

To explore indicator math without a runner:

```bash
go run ./examples/indicator_example/
```

That example shows `indicator.Calculate(candles)` vs `table.ApplyIndicatorsToColumn`.

---

## SMA crossover example

[`examples/strategies/sma_crossover/`](../examples/strategies/sma_crossover/):

- `Indicators()` returns short and long SMA  
- Golden/death cross uses **previous** vs **current** bar (state on strategy struct)  
- Not the same as “short SMA above long SMA every day” (that would be a level rule, not a crossover)

---

## Pitfalls

| Pitfall | What to do |
|---------|------------|
| Forgetting to apply indicators | Use `WithIndicators` or `ApplyIndicators` before `Start()` |
| Wrong type assertion | SMA/EMA return `float64`; MACD returns `indicators.MACDValue` |
| Not enough bars | First `period-1` bars have `nil` indicators — return no orders |
| `Runner.Run()` vs `Start()` | Prefer `Start()`; `Run()` delegates to `Start()` for compatibility |

## Dependency graph

Indicators declare dependencies via `Dependencies()`. When you call `table.ApplyIndicators`, the framework:

1. Collects all indicators (including transitive dependencies)
2. Topologically sorts them so dependencies compute first
3. Applies each indicator to every column in the table

Example: `MACD` depends on two EMAs — those EMAs are computed and stored on candles before MACD runs. You only pass `[macd]` to `ApplyIndicators`; you do not need to list EMAs separately.

---

## Related

- [Writing a strategy](./writing-a-strategy.md)  
- [Loading market data](./loading-data.md)  
- [pkg/indicators](./pkg/indicators/indicators.md) (API reference)
