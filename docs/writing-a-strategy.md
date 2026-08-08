# Writing a strategy

A **strategy** is your trading logic: given the current market snapshot, it returns **orders**. Backgommon runs that logic once per bar inside the **runner**.

Strategies live in **your code** (`examples/…` or your own repo), not in `pkg/`. The framework only provides **`strategy.BaseStrategy`** to reduce boilerplate.

---

## The journey: one bar at a time

```
Runner loads row T from TimeseriesTable
        │
        ▼
  map[string]core.Candle   ←── one entry per symbol
        │
        ▼
  YourStrategy.OnTick(data)
        │
        ▼
  []portfolio.Order         ←── may be empty
        │
        ▼
  Runner fills prices, validates risk, updates portfolio
```

You think in **signals → orders**. The runner thinks in **orders → fills → positions**.

---

## Minimal strategy shape

```go
package mystrat

import (
    "github.com/CCAtAlvis/backgommon/pkg/core"
    "github.com/CCAtAlvis/backgommon/pkg/portfolio"
    "github.com/CCAtAlvis/backgommon/pkg/strategy"
)

type MyStrategy struct {
    strategy.BaseStrategy
    symbol string
}

func NewMyStrategy(symbol string) *MyStrategy {
    return &MyStrategy{symbol: symbol}
}

func (s *MyStrategy) OnTick(data map[string]core.Candle) []portfolio.Order {
    candle, ok := data[s.symbol]
    if !ok {
        return nil
    }

    _ = candle.Close // your logic here

    return nil // or []portfolio.Order{ ... }
}
```

Embed **`BaseStrategy`** so you only override what you need. The runner injects the portfolio via `SetPortfolio` before the loop — use `s.Portfolio` inside `OnTick`.

---

## Creating orders

```go
portfolio.NewOrder(
    instrument,           // must match a column in your table
    portfolio.Long,       // or portfolio.Short (if EnableShorts)
    portfolio.Entry,      // or portfolio.Exit
    quantity,             // shares/contracts
    leverage,             // 1.0 = no leverage
)
```

**Price:** Leave at **0** to let the runner apply [fill pricing](./order-fill.md). Set `Price > 0` only for explicit limits.

**Exits:** Use `portfolio.Exit` with the same side as the open position (long position → `portfolio.Long` + `portfolio.Exit`). Exit quantity can match position size; see SMA example for full exit.

---

## Accessing portfolio state

```go
positions := s.Portfolio.Positions()
pos, inPosition := positions[s.symbol]

if !inPosition {
    // entry logic
}
if inPosition {
    // exit logic; pos.Quantity, pos.OpenPrice, etc.
}
```

Do not call `ProcessOrder` from the strategy — return orders and let the runner handle execution.

---

## Indicators in `OnTick`

Indicators should already be on the candle (pre-computed on the table). See [Indicators](./indicators-application.md).

```go
val, err := candle.GetIndicator("SMA_50")
if err != nil {
    return nil // warm-up period, not ready yet
}
sma, ok := val.(float64)
if !ok {
    return nil
}
```

Expose which indicators you need via a helper (convention):

```go
func (s *MyStrategy) Indicators() []interfaces.Indicator {
    return []interfaces.Indicator{s.sma50, s.sma200}
}
```

Pass to runner: `runner.WithIndicators(strat.Indicators())`.

---

## Reference example: SMA crossover

Full implementation: [`examples/strategies/sma_crossover/sma_crossover.go`](../examples/strategies/sma_crossover/sma_crossover.go).

Patterns to copy:

- **Config struct** (`SMAStrategyConfig`) for windows, quantity, custom metadata  
- **True crossover** — compare previous bar vs current bar, not just `short > long`  
- **`Indicators()`** for runner pre-calculation  
- **Exit quantity** — use open position size when closing  

---

## Lifecycle hooks (optional)

`BaseStrategy` provides no-op defaults you can override:

| Method | When it runs |
|--------|----------------|
| `OnTick` | Each bar — **required** for logic |
| `OnOrderFilled` | After an order is processed |
| `OnPositionOpened` | New position opened |
| `OnPositionClosed` | Position fully closed |

`OnDayStart` / `OnDayEnd` exist on `BaseStrategy` but are **not** called by the runner yet — planned for a later phase.

---

## Where to put your code

| Use case | Location |
|----------|----------|
| Learning / open-source examples | `examples/strategies/your_strategy/` |
| Private trading system | Your app (e.g. gp-trade), import Backgommon as a module |

Keep **`pkg/`** for framework code only.

---

## Checklist before `runner.Start()`

- [ ] `TimeseriesTable` has enough history for your indicators  
- [ ] `Indicators()` applied (`WithIndicators` or `table.ApplyIndicators`)  
- [ ] `symbol` in strategy matches table column names  
- [ ] Portfolio `InitialCapital`, `EnableShorts` set  
- [ ] Risk settings reasonable for your order sizes  
- [ ] Fill mode chosen ([order fill doc](./order-fill.md))  

---

## Related docs

- [Getting started](./getting-started.md)  
- [Loading market data](./loading-data.md)  
- [Order fill pricing](./order-fill.md)  
- [Strategy design philosophy](./approach.md)  
