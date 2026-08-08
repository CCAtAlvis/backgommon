# Backgommon Documentation

Welcome to Backgommon — a Go backtesting framework for **portfolio-level** trading strategies (one symbol or many).

This guide is organized as a **journey**: each doc builds on the previous one. You do not need to read everything at once; use the path that matches where you are.

---

## Recommended reading order

| Step | Document | You will learn |
|------|----------|----------------|
| 1 | [Getting started](./getting-started.md) | Run your first backtest end-to-end |
| 2 | [Writing a strategy](./writing-a-strategy.md) | Implement `OnTick`, orders, and `BaseStrategy` |
| 3 | [Loading market data](./loading-data.md) | JSON and CSV formats, `TimeseriesTable` |
| 4 | [Indicators](./indicators-application.md) | Where and how to compute SMA, MACD, etc. |
| 5 | [Order fill pricing](./order-fill.md) | How execution prices and slippage are simulated |
| 6 | [Portfolio settings & costs](./portfolio-and-costs.md) | Brokerage, taxes, SIP, drawdown, leverage |
| 7 | [Reporting and output](./reporting.md) | JSON export, HTML charts, equity curve |
| 8 | [Extending the framework](./extending-the-framework.md) | Override costs, slippage, cash flow, risk hooks |
| 9 | [Strategy design notes](./approach.md) | Why embed `BaseStrategy`, lifecycle hooks |
| 10 | [Parameter search](./parameter-search.md) | Grid vs random vs TPE vs staged; budgeted sweeps |
| 11 | [Target / reverse search](./target-search.md) | Constraints & goals (DD, Sharpe, CAGR) → params |

After that, dig into package reference under [`docs/pkg/`](./pkg/index.md) when you need API-level detail.

---

## Quick mental model

```
  Market data                Your logic              Framework
 ┌─────────────┐         ┌──────────────┐         ┌─────────────┐
 │ Timeseries  │ ──────► │  Strategy    │ ──────► │   Runner    │
 │   Table     │  OnTick │  OnTick()    │ orders  │  Portfolio  │
 │  + candles  │         │  returns []  │         │  Risk mgr   │
 └─────────────┘         │   Order      │         └─────────────┘
       ▲                 └──────────────┘                │
       │                        ▲                        ▼
  pkg/data              examples/ or your app      Results, equity curve
  pkg/indicators
```

**You own:** strategy logic, data files, configuration.  
**The framework owns:** the event loop, order fill simulation, portfolio accounting, risk checks.

---

## Examples in this repo

| Example | Path | What it demonstrates |
|---------|------|----------------------|
| SMA crossover | [`examples/strategies/sma_crossover/`](../examples/strategies/sma_crossover/) | Full backtest with indicators, results, real JSON data |
| Indicator usage | [`examples/indicator_example/`](../examples/indicator_example/) | Direct vs table-based indicator calculation |
| Indicator strategy | [`examples/indicator_strategy/`](../examples/indicator_strategy/) | Custom indicator in a strategy struct |

See [`examples/README.md`](../examples/README.md) for run commands.

---

## Package reference (for contributors)

Developer-focused docs live under [`docs/pkg/`](./pkg/index.md):

- [runner](./pkg/runner/runner.md) · [portfolio](./pkg/portfolio/portfolio.md) · [risk](./pkg/risk/risk.md)
- [indicators](./pkg/indicators/indicators.md) · [types](./pkg/types/types.md) · [interfaces](./pkg/interfaces/interfaces.md)

These describe **what each package does internally**. User guides above describe **how to use them together**.

---

## Project status

Backgommon is under active development. Some features (brokerage, slippage, live trading) are designed but not fully implemented. See the [framework task list](./prd/backgommon-framework/backgommon-framework-task-list.md) for current progress.
