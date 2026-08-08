# Order fill pricing

When your strategy returns an order, the runner must decide **at what price** it filled. Real brokers fill at market, limit, next open, etc. Backgommon simulates that with a **`FillPricer`**.

This doc explains the model, the `FillContext` struct, and how to use or extend fill logic — without reading the runner source.

---

## The journey: from signal to fill

Here is what happens on each bar when your strategy fires:

```
Bar T (current row in the table)
│
├─ Strategy.OnTick(data)  →  []Order  (often Price = 0)
│
├─ Runner: for each order with Price == 0
│     FillPricer.FillPrice(FillContext)  →  base price
│     execution.ApplySlippage(...)       →  adverse adjustment (from portfolio.Execution)
│     order.Price = final price
│
├─ RiskManager.ValidateOrder(...)
│
└─ Portfolio.ProcessOrder(...)  →  cash & positions updated (margin, brokerage, taxes)
```

**Key rule:** If `Order.Price > 0`, the runner **does not** call `FillPricer`. You set an explicit price (limit/stop). If `Price == 0`, the framework fills it for you.

---

## As a runner user (most people)

You **do not** build `FillContext` yourself. Pick a built-in mode or pass a custom pricer when creating the runner:

```go
import (
    "github.com/CCAtAlvis/backgommon/pkg/execution"
    "github.com/CCAtAlvis/backgommon/pkg/runner"
)

r := runner.New(
    strat,
    runner.WithPortfolio(pf),
    runner.WithRiskManager(rm),
    runner.WithData(table),

    // Option A: built-in mode (recommended to start)
    runner.WithFillMode(execution.FillCurrentClose),

    // Option B: another built-in
    runner.WithFillMode(execution.FillNextBarOpen),

    // Option C: fully custom (see below)
    runner.WithFillPricer(myPricer),
)
```

If you omit `WithFillMode` / `WithFillPricer`, the default is **`CurrentBarClose`** — fill at the close of the bar where the signal occurred.

### Choosing a fill mode

| Mode | When to use | Price used |
|------|-------------|------------|
| `CurrentBarClose` | Signal at end of day, fill at close (common default) | Current bar close |
| `CurrentBarOpen` | Signal assumed at open of same bar | Current bar open |
| `NextBarOpen` | Signal at close, fill at **next** session open (more realistic for daily systems) | Next bar open |
| `MidPrice` | Simple intrabar proxy | (high + low) / 2 |
| `OpenCloseAvg` | Average of open and close | (open + close) / 2 |
| `CurrentBarHigh` / `CurrentBarLow` | Stress / optimistic scenarios | High or low |
| `WorstCaseWithinBar` | Conservative backtest (buy at high, sell at low for longs) | Adverse side of bar |

**Timing mental model (single symbol):**

```
        Bar T              Bar T+1
      ┌─────────┐        ┌─────────┐
      │ signal  │        │         │
      │ OnTick  │        │         │
      └─────────┘        └─────────┘
           │                  │
   CurrentBarClose      NextBarOpen
   fills here           fills here
```

On the **last bar** of the backtest, `NextBarOpen` **returns an error** — there is no next bar. That order fails rather than filling at price 0.

---

## What is `FillContext`?

`FillContext` is a **snapshot of market data** passed to your fill logic. It exists so `FillPricer` implementations know which bar(s) to read.

```go
type FillContext struct {
    Order      portfolio.Order
    CurrentBar core.Candle
    NextBar    *core.Candle              // nil on last row or if symbol missing next row
    AllBars    map[string]core.Candle    // full row at time T (multi-asset)
    NextBars   map[string]core.Candle    // full row at time T+1
}
```

### Field guide

| Field | Meaning |
|-------|---------|
| **`Order`** | The order being priced: instrument, side, entry/exit, quantity. Used for worst-case logic and to know which symbol’s bars matter. |
| **`CurrentBar`** | OHLCV for `Order.Instrument` on the **same timestamp** as `OnTick` — the bar where the signal fired. |
| **`NextBar`** | That instrument’s candle on the **next timestamp**, or `nil` if there is no next row (last bar) or the symbol is missing on the next row. |
| **`AllBars`** | Every symbol’s candle at time T (one row of a multi-column table). |
| **`NextBars`** | Every symbol’s candle at time T+1. |

When the runner builds the context:

- `CurrentBar` == `AllBars[order.Instrument]` (when that symbol exists on the row)
- `NextBar` == pointer to `NextBars[order.Instrument]` when present

Built-in pricers use **`Order`**, **`CurrentBar`**, and **`NextBar`** only.  
**`AllBars` / `NextBars`** are for **custom** pricers that need other symbols (e.g. fill off a benchmark index).

### Who constructs `FillContext`?

**Only the runner** (in `applyFillPrices`). As a strategy author or runner user, you configure `WithFillMode` or `WithFillPricer` — you never assemble `FillContext` manually unless you are **unit testing** a custom pricer.

---

## As a custom `FillPricer` author

Implement `interfaces.FillPricer`:

```go
type FillPricer interface {
    FillPrice(ctx FillContext) (float64, error)
}
```

### Built-in slippage (recommended)

Configure slippage on `portfolio.Settings.Execution` — the runner applies it automatically after the fill pricer:

```go
Execution: portfolio.ExecutionSettings{
    SlippageMode:        "PercentOfPrice",
    PercentSlippageRate: 0.001, // 0.1% adverse
}
```

| Slippage mode | Long entry | Long exit |
|---------------|------------|-----------|
| `PercentOfPrice` | price × (1 + rate) | price × (1 − rate) |
| `FixedPoints` | price + points | price − points |

Default: `StandardSlippageModel` reads `portfolio.Settings.Execution`. Override with `runner.WithSlippageModel` — see [Extending the framework](./extending-the-framework.md).

### Example: custom slippage on close

```go
pricer := execution.NewFuncFillPricer(func(ctx interfaces.FillContext) (float64, error) {
    slip := 0.001 // 0.1%
    if ctx.Order.Side == portfolio.Long && ctx.Order.Type == portfolio.Entry {
        return ctx.CurrentBar.Close * (1 + slip), nil
    }
    if ctx.Order.Side == portfolio.Long && ctx.Order.Type == portfolio.Exit {
        return ctx.CurrentBar.Close * (1 - slip), nil
    }
    return ctx.CurrentBar.Close, nil
})

runner.New(strat, runner.WithFillPricer(pricer), ...)
```

### Example: multi-symbol (use `AllBars`)

```go
pricer := execution.NewFuncFillPricer(func(ctx interfaces.FillContext) (float64, error) {
    bench, ok := ctx.AllBars["SPY"]
    if !ok {
        return ctx.CurrentBar.Close, nil
    }
    return (ctx.CurrentBar.Close + bench.Close) / 2, nil
})
```

### Example: wrap a built-in mode

```go
inner := execution.NewStandardFillPricer(execution.FillMidPrice)
pricer := execution.NewFuncFillPricer(func(ctx interfaces.FillContext) (float64, error) {
    base, err := inner.FillPrice(ctx)
    if err != nil {
        return 0, err
    }
    return base + 0.05, nil
})
```

Built-in implementations live in **`pkg/execution`**. The interface lives in **`pkg/interfaces/fill.go`**.

---

## Interaction with risk exits

The risk manager can emit exit orders (e.g. stop-loss). Those orders go through the **same** `FillPricer` and **same** `FillContext` on the **same bar** as strategy orders.

If you need “entries at next open, exits at current close”, that is **not** supported out of the box yet — you would use a custom pricer that branches on `ctx.Order.Type`.

---

## Limitations (today)

| Limitation | Notes |
|------------|--------|
| No historical window in context | Custom VWAP-over-N-bars needs future API extension |
| Different fill mode per order type | Use a custom `FillPricer` that branches on `ctx.Order.Type` |
| Sparse tables | If symbol missing on T+1, `NextBar` is nil even if other symbols have next data |
| Short selling | Slippage directions are defined; full short cash model is incomplete |

---

## Summary

| Role | What you do |
|------|-------------|
| **Strategy author** | Return orders with `Price == 0`; ignore `FillContext` |
| **Runner user** | `runner.WithFillMode(...)` or `WithFillPricer(...)` |
| **Fill author** | Implement `FillPricer`; read `FillContext` fields as needed |

Default assumption: **signal and fill on the same bar**, price from **close**, unless you choose otherwise.
