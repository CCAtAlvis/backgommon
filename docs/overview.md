# Backgommon `pkg` Folder Overview

This document provides a developer-focused overview of the `pkg` directory in Backgommon. It describes the purpose, main files, and key types/interfaces of each subpackage, and explains how they fit together in the framework.

---

## Table of Contents
- [Introduction](#introduction)
- [Package Overviews](#package-overviews)
  - [indicators](#indicators)
  - [types](#types)
  - [core](#core)
  - [interfaces](#interfaces)
  - [portfolio](#portfolio)
  - [runner](#runner)
  - [risk](#risk)
  - [strategy](#strategy)
- [Overall Flow and Integration](#overall-flow-and-integration)

---

## Introduction

The `pkg` directory contains the core building blocks of the Backgommon backtesting framework. Each subpackage is responsible for a specific aspect of the system, and together they provide a modular, extensible foundation for implementing and testing trading strategies in Go.

This documentation is intended for developers extending, maintaining, or integrating with Backgommon.

---

## Package Overviews

### indicators

**Purpose:**
Provides a collection of technical indicators (e.g., SMA, EMA, MACD) and utilities for use in trading strategies. Indicators are composable and can depend on each other. The package also supports user-defined custom indicators.

**Main Files:**
- `sma.go`: Simple Moving Average (SMA) implementation
- `ema.go`: Exponential Moving Average (EMA) implementation
- `macd.go`: Moving Average Convergence Divergence (MACD) implementation
- `custom.go`: Support for user-defined custom indicators
- `value.go`: Defines value types returned by indicators (e.g., `SingleValue`)
- `validator.go`: Utilities for validating indicator dependency graphs (e.g., cycle detection)

**Key Types/Interfaces:**
- `SMA`, `EMA`, `MACD`: Implementations of common technical indicators
- `CustomIndicator`: Allows users to define their own indicator logic
- `SingleValue`, `MACDValue`: Types representing indicator output values
- All indicators implement the `Indicator` interface (from `pkg/interfaces`), which requires `Calculate`, `Name`, and `Dependencies` methods

**Usage Notes:**
- Indicators can be composed: e.g., MACD depends on multiple EMA instances
- The `MACD` indicator, for instance, internally creates and relies on three `EMA` instances (fast, slow, and signal). Its `Dependencies()` method formally lists these EMAs, which is useful for automatic dependency resolution systems. During its `Calculate` method, `MACD` will attempt to use pre-calculated EMA values if they are already present on the input `core.Candle` data (retrieved using the respective EMA's `Name()`); otherwise, it will compute them. The signal line's EMA has special handling, as it's calculated over a synthetic series derived from the MACD line itself.
- Custom indicators can be created by providing a calculation function and dependencies
- The `validator.go` utility helps ensure there are no circular dependencies between indicators
- Example usage:
  ```go
  ema := indicators.NewEMA(20)
  value := ema.Calculate(candles)
  macd := indicators.NewMACD(12, 26, 9)
  custom := indicators.NewCustomIndicator("MyInd", myFunc, []interfaces.Indicator{ema})
  ```

### types

**Purpose:**
Provides core data structures for time series, tabular data, and backtest results. These types are used throughout the framework for storing and manipulating market data, indicator values, and analytics.

**Main Files:**
- `timeseries_table.go`: Generic time-series table for storing and processing time-indexed data (e.g., candles, indicators)
- `table.go`: Generic table structure for tabular data with dynamic columns and rows
- `results.go`: Structures for storing backtest results and analytics (e.g., `Results`, `AccountValue`)

**Key Types/Interfaces:**
- `TimeseriesTable[T]`: Generic time-series table with support for indicators and efficient time-based access
- `Table`: Flexible table for arbitrary columns and rows
- `Results`: Holds summary statistics and analytics for a backtest
- `AccountValue`: Represents account value snapshots over time

**Usage Notes:**
- `TimeseriesTable` is the main structure for storing candles and indicator values during a backtest
- Tables support dynamic columns and can be iterated or queried by column
- Results and AccountValue are used for reporting and analytics after a backtest
- Example usage:
  ```go
  candles := types.NewTimeseriesTable[core.Candle]([]string{"Open", "High", "Low", "Close"})
  results := types.Results{InitialCapital: 10000}
  ```

### core

**Purpose:**
Defines the fundamental data structures and interfaces for market data and indicator values. This package is the foundation for all time-series and indicator operations in the framework.

**Main Files:**
- `candle.go`: Defines the `Candle` type (OHLCV data) and methods for storing/retrieving indicator values per candle
- `indicator.go`: Defines the `Value` interface, which all indicator output types must implement

**Key Types/Interfaces:**
- `Candle`: Represents a single time period's OHLCV data, with support for storing indicator values
- `Value`: Interface for indicator output values (must implement `Value() float64`)

**Usage Notes:**
- `Candle` is the primary unit of market data throughout the framework
- Indicator values can be attached to each candle for efficient access
- All indicator output types must implement the `Value` interface
- Indicator values attached to a `Candle` (via its `SetIndicator` method) are stored in a map. This map is keyed by the string returned by the indicator's `Name()` method, as defined in the `interfaces.Indicator` interface.
- Example usage:
  ```go
  candle := core.NewCandle()
  candle.Open = 100
  candle.SetIndicator("EMA_20", emaValue)
  v, err := candle.GetIndicator("EMA_20")
  ```

### interfaces

**Purpose:**
Defines the main interfaces for strategies, portfolio management, risk management, and indicators. These interfaces enable extensibility and allow users to provide custom implementations for each major component.

**Main Files:**
- `interface.go`: Contains all core interfaces for strategies, portfolio, risk, and indicators

**Key Types/Interfaces:**
- `Strategy`: Interface for trading strategies (main entry point for user logic)
- `PortfolioManager`: Interface for portfolio management (order processing, position tracking)
- `RiskManager`: Interface for risk management (order validation, exit checks)
- `Indicator`: Interface for all technical indicators (calculation, naming, dependencies)

**Usage Notes:**
- Users implement the `Strategy` interface to define custom trading logic
- Custom portfolio or risk management logic can be provided by implementing the respective interfaces
- All indicators must implement the `Indicator` interface
- Example usage:
  ```go
  type MyStrategy struct {}
  func (s *MyStrategy) OnTick(data map[string]core.Candle) []portfolio.Order { /* ... */ }
  ```

### portfolio

**Purpose:**
Manages all aspects of portfolio state, including cash, open/closed positions, order processing, and portfolio-level statistics. Provides types and logic for positions, orders, and portfolio analytics.

**Main Files:**
- `portfolio.go`: Main portfolio logic (cash, open/closed positions, order processing, stats)
- `position.go`: Position management, metrics, and risk tracking
- `order.go`: Order type, creation, and fill logic

**Key Types/Interfaces:**
- `Portfolio`: Manages cash, positions, and order processing
- `Position`: Represents an open or closed trading position, with metrics and risk tracking
- `Order`: Represents a trading order (entry/exit, long/short, leverage, etc.)
- `Settings`: Portfolio configuration (initial capital, max positions, etc.)
- `PositionMetrics`, `PortfolioStats`: Analytics and reporting types

**Usage Notes:**
- The `Portfolio` is the main stateful object for tracking trading activity
- Orders are processed through the portfolio, which updates cash and positions
- Positions track PnL, drawdown, ROI, and other metrics
- Example usage:
  ```go
  settings := &portfolio.Settings{InitialCapital: 10000, AllowShorts: true, MaxPositions: 5}
  pf := portfolio.New(settings)
  order := portfolio.NewOrder("AAPL", portfolio.Long, portfolio.Entry, 10, 1.0)
  pf.ProcessOrder(order)
  pf.UpdatePositions(map[string]float64{"AAPL": 150.0})
  ```

### runner

**Purpose:**
Coordinates the entire backtesting process, managing the strategy, portfolio, risk manager, and data flow. Provides the main entry point for running a backtest and collecting results.

**Main Files:**
- `runner.go`: Main backtest runner logic (core loop, order processing, equity curve, etc.)
- `options.go`: Option functions for configuring the runner (portfolio, risk manager, data, results)

**Key Types/Interfaces:**
- `Runner`: Main struct that ties together strategy, portfolio, risk manager, and data
- `Option`: Functional options for configuring the runner
- `IndicatorConfig`: Configuration for indicator calculation during backtests

**Usage Notes:**
- The `Runner` is the main entry point for running a backtest
- Strategies, portfolio managers, and risk managers are injected via options
- Data is provided as a `TimeseriesTable` of candles
- The runner manages the backtest loop, order processing, and equity curve
- The `Runner` struct includes an `IndicatorConfig` field. If this field is populated (e.g., directly after `Runner` instantiation or via a custom `Option`), it allows specifying a list of `interfaces.Indicator` and a common lookback period. The `Runner` can then leverage this configuration, for example, to pre-calculate these indicators across the entire input `TimeseriesTable[core.Candle]` (potentially by invoking a method like `ApplyIndicators` on the table) before the main event-driven backtest simulation begins. This ensures indicator values are computed and available on `core.Candle` instances when the strategy's `OnTick` is called.
- Example usage:
  ```go
  runner := runner.New(myStrategy,
      runner.WithPortfolio(myPortfolio),
      runner.WithRiskManager(myRiskManager),
      runner.WithData(myData),
  )
  err := runner.Start()
  ```

### risk

**Purpose:**
Provides risk management logic, including order validation, position exit rules, and risk metrics. Allows configuration of risk parameters such as max drawdown, leverage, stop loss, and take profit.

**Main Files:**
- `manager.go`: Main risk manager logic (order validation, exit checks, settings)
- `exit_conditions.go`: Logic for stop loss, take profit, trailing stop, and risk metrics

**Key Types/Interfaces:**
- `Manager`: Main risk manager struct implementing `RiskManager` interface
- `Settings`: Risk management configuration (max drawdown, leverage, stop loss, etc.)
- `PositionRisk`: Struct for risk metrics on a position

**Usage Notes:**
- The risk manager validates orders and checks for exit conditions on positions
- Supports stop loss, take profit, and trailing stop logic
- Risk parameters are configurable via the `Settings` struct
- Example usage:
  ```go
  riskSettings := &risk.Settings{MaxDrawdown: 0.2, MaxLeverage: 2.0, UseStopLoss: true, DefaultStopLoss: 0.05}
  riskMgr := risk.New(riskSettings)
  ```

### strategy

**Purpose:**
Provides the base implementation for trading strategies, including default method implementations and portfolio access. Users embed `BaseStrategy` and override only the methods they need for their custom logic.

**Main Files:**
- `base.go`: Base strategy struct and default method implementations

**Key Types/Interfaces:**
- `BaseStrategy`: Embeddable struct that implements the `interfaces.Strategy` interface with default (often no-op) behavior. It also provides additional lifecycle hook methods (e.g., `OnDayStart`, `OnDayEnd`) not formally in the `Strategy` interface, which users can optionally override.

**Usage Notes:**
- Users embed `BaseStrategy` in their own strategy structs and override only the methods they need (e.g., `OnTick`)
- Provides default no-op implementations for all lifecycle hooks
- Example usage:
  ```go
  type MyStrategy struct {
      strategy.BaseStrategy
      // custom fields
  }
  func (s *MyStrategy) OnTick(data map[string]core.Candle) []portfolio.Order {
      // custom logic
  }
  ```

---

## Overall Flow and Integration

The Backgommon framework is designed for modularity and extensibility. Here's how the main subpackages interact during a typical backtest:

1. **Strategy** (`strategy`, `interfaces`):
   - User implements a custom strategy by embedding `BaseStrategy` and overriding relevant methods (e.g., `OnTick`).
   - The strategy receives market data (candles) and returns orders to be executed.

2. **Runner** (`runner`):
   - The `Runner` orchestrates the backtest, calling the strategy on each tick, processing orders, and updating the portfolio.
   - It injects the portfolio manager, risk manager, and data into the strategy.

3. **Portfolio** (`portfolio`):
   - Manages cash, open/closed positions, and processes orders from the strategy.
   - Updates position metrics and provides portfolio-level analytics.

4. **Risk** (`risk`):
   - Validates orders before execution and checks for exit conditions (e.g., stop loss, take profit).
   - Can be customized with different risk parameters.

5. **Indicators** (`indicators`):
   - Used by strategies to compute technical signals from market data.
   - Can be composed and attached to candles or tables for efficient access.

6. **Types/Core** (`types`, `core`):
   - Provide the foundational data structures (candles, tables, results) used throughout the framework.

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

<!-- To be filled: How the subpackages interact during a typical backtest, with optional diagram or code snippet --> 