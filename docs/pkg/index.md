# Backgommon `pkg` Directory: Core Framework Documentation

This section provides detailed documentation for the sub-packages within the `pkg` directory, which form the core building blocks of the Backgommon backtesting framework.

## Table of Contents

*   [Indicators (`indicators`)](./indicators/indicators.md)
*   [Types (`types`)](./types/types.md)
*   [Core (`core`)](./core/core.md)
*   [Interfaces (`interfaces`)](./interfaces/interfaces.md)
*   [Portfolio (`portfolio`)](./portfolio/portfolio.md)
*   [Runner (`runner`)](./runner/runner.md)
*   [Risk (`risk`)](./risk/risk.md)
*   [Strategy (`strategy`)](./strategy/strategy.md)

---

## Overall Flow and Integration

The Backgommon framework is designed for modularity and extensibility. Here's how the main subpackages interact during a typical backtest:

1.  **Strategy** (`strategy`, `interfaces`):
    *   User implements a custom strategy by embedding `BaseStrategy` and overriding relevant methods (e.g., `OnTick`).
    *   The strategy receives market data (candles) and returns orders to be executed.

2.  **Runner** (`runner`):
    *   The `Runner` orchestrates the backtest, calling the strategy on each tick, processing orders, and updating the portfolio.
    *   It injects the portfolio manager, risk manager, and data into the strategy.

3.  **Portfolio** (`portfolio`):
    *   Manages cash, open/closed positions, and processes orders from the strategy.
    *   Updates position metrics and provides portfolio-level analytics.

4.  **Risk** (`risk`):
    *   Validates orders before execution and checks for exit conditions (e.g., stop loss, take profit).
    *   Can be customized with different risk parameters.

5.  **Indicators** (`indicators`):
    *   Used by strategies to compute technical signals from market data.
    *   Can be composed and attached to candles or tables for efficient access.

6.  **Types/Core** (`types`, `core`):
    *   Provide the foundational data structures (candles, tables, results) used throughout the framework.

### Example Flow (Code Snippet)

```go
// 1. Create portfolio, risk manager, and strategy
settings := &portfolio.Settings{InitialCapital: 10000, AllowShorts: true, MaxPositions: 5}
pf := portfolio.New(settings)
riskSettings := &risk.Settings{MaxDrawdown: 0.2, MaxLeverage: 2.0, UseStopLoss: true, DefaultStopLoss: 0.05}
riskMgr := risk.New(riskSettings)
myStrategy := &MyStrategy{strategy.BaseStrategy{}}

// 2. Prepare data (candles)
data := types.NewTimeseriesTable[core.Candle]([]string{"Open", "High", "Low", "Close"})
// ... load candles into data ...

// 3. Create and run the backtest runner
runner := runner.New(
    myStrategy,
    runner.WithPortfolio(pf),
    runner.WithRiskManager(riskMgr),
    runner.WithData(data),
)
err := runner.Start()
if err != nil {
    log.Fatalf("Backtest failed: %v", err)
}

// 4. Analyze results
fmt.Println("Final portfolio value:", pf.Value())
```

### Diagram (Textual)

```
[Strategy] <-> [Runner] <-> [Portfolio] <-> [Risk]
     |             |             |
 [Indicators]   [Data]      [Types/Core]
```

- The **Strategy** makes decisions using indicators and market data.
- The **Runner** coordinates the flow, calling the strategy and updating the portfolio.
- The **Portfolio** processes orders and tracks positions.
- The **Risk Manager** validates orders and enforces exit rules.
- **Indicators** and **Types/Core** provide reusable building blocks for all components. 