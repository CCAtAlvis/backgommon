# Backgommon

Backgommon is a backtesting framework for trading strategies, written in Go. It supports **portfolio-level** backtests: one symbol or many, long-only or long/short, with modular risk and execution simulation.

> A portfolio can have a single asset — you do not need multiple symbols.

---

## Documentation

**Start here:** [docs/README.md](docs/README.md) — guided path from first backtest to custom fill logic.

| Guide | Description |
|-------|-------------|
| [Getting started](docs/getting-started.md) | Run your first backtest |
| [Writing a strategy](docs/writing-a-strategy.md) | `OnTick`, orders, `BaseStrategy` |
| [Loading market data](docs/loading-data.md) | JSON & CSV OHLCV formats |
| [Indicators](docs/indicators-application.md) | Pre-compute vs in-strategy |
| [Order fill pricing](docs/order-fill.md) | `FillContext`, `FillPricer`, fill modes |

Examples: [examples/README.md](examples/README.md) · Package reference: [docs/pkg/](docs/pkg/index.md)

---

## Quick run

```bash
go run ./examples/strategies/sma_crossover/
go run ./examples/strategies/sma_crossover/ -data /path/to/prices.json
```

---

## Why Backgommon?

Most backtesters focus on a **single ticker**. Backgommon is built around a **time × symbols** table: each row is a date, each column is an instrument. That matches how portfolio strategies actually work.

---

## Project layout

```
backgommon/
├── pkg/           # Framework only (runner, portfolio, risk, indicators, …)
├── examples/      # Example strategies & demos (your templates live here)
├── docs/          # User guides + package reference
└── cmd/           # CLI tools (future)
```

**Your strategies do not go in `pkg/`.** Put them in `examples/` or your own application that imports this module.

---

## Features (current)

- Bar-by-bar backtest **runner** with equity curve and results summary
- **Portfolio** and **risk** managers (configurable settings)
- Technical **indicators** (SMA, EMA, MACD, custom) on `TimeseriesTable`
- Pluggable **order fill** simulation (`FillPricer`, multiple bar-price modes)
- **JSON / CSV** data loaders (generic OHLCV, not exchange-specific)
- Reference **SMA crossover** example

Planned: brokerage/slippage in portfolio, live trading adapter, more indicators. See [framework task list](docs/prd/backgommon-framework/backgommon-framework-task-list.md).

---

## What Backgommon is not

- Not an HFT engine
- Not a broker API or data vendor
- Not a guarantee of profitable strategies

---

## Status

Work in progress — APIs may change. Contributions welcome.

**Note:** This repo was migrated from a private codebase; documentation and APIs are stabilizing.
