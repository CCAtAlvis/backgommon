# Getting started

This guide walks you from zero to a running backtest. When you finish, you will understand how the main pieces connect — even if you never read the source code.

---

## What you need

- Go 1.19+
- This repository cloned locally

---

## Step 1: Run the reference example

The SMA crossover example is the canonical “hello world” backtest:

```bash
cd backgommon

# Synthetic data (300 daily bars, SMA 20/50)
go run ./examples/strategies/sma_crossover/

# Real OHLCV from a JSON file (symbol is inside the file)
go run ./examples/strategies/sma_crossover/ -data /path/to/prices.json

# CSV file (you pass the symbol name)
go run ./examples/strategies/sma_crossover/ -data /path/to/prices.csv -symbol MY-SYM
```

You should see output like:

```
--- Backtest Results ---
Period:        2020-01-01 → 2020-10-26
Initial:       100000.00
Final:         ...
Return:        ...
Total Trades:  ...
```

That output comes from the **runner** after it simulates your strategy bar-by-bar.

---

## Step 2: Understand the five pieces

Every backtest in Backgommon uses the same five components:

| Component | Package | Your job |
|-----------|---------|----------|
| **Data** | `pkg/data`, `pkg/types` | Load OHLCV into a `TimeseriesTable` |
| **Strategy** | Your code (+ `pkg/strategy.BaseStrategy`) | Implement `OnTick`, return orders |
| **Portfolio** | `pkg/portfolio` | Configure capital, shorts, fees (settings) |
| **Risk** | `pkg/risk` | Configure limits, stop-loss behavior (settings) |
| **Runner** | `pkg/runner` | Wire everything and call `Start()` |

You rarely touch the runner loop itself. You **configure** it and **implement** the strategy.

---

## Step 3: The smallest possible backtest (conceptual)

```go
package main

import (
    "log"

    "github.com/CCAtAlvis/backgommon/pkg/data"
    "github.com/CCAtAlvis/backgommon/pkg/portfolio"
    "github.com/CCAtAlvis/backgommon/pkg/risk"
    "github.com/CCAtAlvis/backgommon/pkg/runner"
    // your strategy type here
)

func main() {
    // 1. Load data
    table, symbol, err := data.LoadTableFromJSON("prices.json")
    if err != nil { log.Fatal(err) }

    // 2. Create strategy (you implement this)
    strat := NewMyStrategy(symbol)

    // 3. Configure portfolio & risk
    pf := portfolio.New(&portfolio.Settings{
        InitialCapital: 100_000,
        EnableShorts:   false,
    })
    rm := risk.New(&risk.Settings{
        MaxLeverage:               1.0,
        MaxPositionAllocationRate: 1.0,
    })

    // 4. Create runner and run
    r := runner.New(
        strat,
        runner.WithPortfolio(pf),
        runner.WithRiskManager(rm),
        runner.WithData(table),
        runner.WithIndicators(strat.Indicators()), // if you use indicators
        // runner.WithFillMode(execution.FillCurrentClose), // optional; this is the default
    )

    if err := r.Start(); err != nil {
        log.Fatal(err)
    }

  runner.PrintResults(r.Results)
}
```

**What `Start()` does internally (simplified):**

1. Optionally pre-compute indicators on the full table
2. For each timestamp (row) in the table:
   - Update open positions with current prices
   - Ask the risk manager if any position should exit
   - Call your strategy’s `OnTick`
   - Fill order prices (see [Order fill pricing](./order-fill.md))
   - Process orders through portfolio and risk validation
3. Build equity curve and results summary

---

## Step 4: Where strategies live

**Strategies are your code**, not part of `pkg/`.

- Put example/reference strategies under `examples/strategies/…`
- Put proprietary strategies in your own app (e.g. gp-trade) that imports Backgommon as a module

`pkg/strategy` only provides **`BaseStrategy`** — embed it and override `OnTick`. See [Writing a strategy](./writing-a-strategy.md).

---

## Step 5: What to read next

| If you want to… | Read |
|-----------------|------|
| Implement your own logic | [Writing a strategy](./writing-a-strategy.md) |
| Load CSV / JSON files | [Loading market data](./loading-data.md) |
| Use SMA, MACD, etc. | [Indicators](./indicators-application.md) |
| Control fill prices (close vs next open) | [Order fill pricing](./order-fill.md) |
| Understand package internals | [pkg overview](./overview.md) |

---

## Common first questions

**Q: Do I put my strategy in `pkg/`?**  
No. `pkg/` is the framework. Your strategy lives in `examples/` or your own repository.

**Q: How does the strategy get portfolio state?**  
Embed `BaseStrategy`. The runner calls `SetPortfolio` before the loop. Use `s.Portfolio.Positions()` inside `OnTick`.

**Q: Must I set a price on orders?**  
No. Return `portfolio.NewOrder(...)` with price 0 (default). The runner’s fill pricer sets the execution price. Set `Price > 0` only when you want an explicit limit. See [Order fill pricing](./order-fill.md).

**Q: Single stock or portfolio?**  
Both. A `TimeseriesTable` has **columns = symbols**, **rows = timestamps**. One column is a single-asset backtest.
