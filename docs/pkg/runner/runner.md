# `runner` Package

**Purpose:**
Coordinates the entire backtesting process, managing the strategy, portfolio, risk manager, and data flow. Provides the main entry point for running a backtest and collecting results.

---

## User Documentation

This section is for developers who are setting up and running backtests using the Backgommon framework.

### Key Concepts for Users

*   **`Runner`**: The central orchestrator for a backtest.
    *   You instantiate a `Runner` with your implemented `interfaces.Strategy` and various optional configurations.
*   **`Option`**: Functional options are used to configure the `Runner`.
    *   Examples: `runner.WithPortfolio(...)`, `runner.WithRiskManager(...)`, `runner.WithData(...)`, `runner.WithResults(...)`.
*   **Data Input**: Market data is provided to the `Runner` typically as a `*types.TimeseriesTable[core.Candle]`.
*   **Starting a Backtest**: The `runner.Start()` method kicks off the backtesting simulation.
*   **Results**: After a backtest, results can be accessed from the `types.Results` struct (if configured with `WithResults`) and the `EquityCurve` (a slice of `types.AccountValue`) which is often a field within the `Runner` itself or the `Results` struct.
*   **`IndicatorConfig`**: An optional configuration to specify indicators that should be pre-calculated by the `Runner` or `TimeseriesTable` across the dataset before the strategy's `OnTick` is called.

### Setting Up and Running a Backtest

```go
import (
    // ... other necessary imports
    "github.com/CCAtAlvis/backgommon/pkg/runner"
    "github.com/CCAtAlvis/backgommon/pkg/portfolio"
    "github.com/CCAtAlvis/backgommon/pkg/risk"
    "github.com/CCAtAlvis/backgommon/pkg/types"
    "github.com/CCAtAlvis/backgommon/pkg/core"
    // Assuming MyStrategy is your strategy implementation
)

func main() {
    // 1. Initialize Strategy, Portfolio, Risk Manager
    myStrategy := &MyStrategy{/* ... */}

    portfolioSettings := &portfolio.Settings{InitialCapital: 100000}
    pf := portfolio.New(portfolioSettings)

    riskSettings := &risk.Settings{MaxDrawdown: 0.20}
    rm := risk.New(riskSettings)

    // 2. Prepare Data (e.g., load from CSV into types.TimeseriesTable[core.Candle])
    // candleData := types.NewTimeseriesTable[core.Candle](...)
    // ... populate candleData ...
    var candleData *types.TimeseriesTable[core.Candle] // Assume this is populated

    // 3. Create Runner with Options
    backtestRunner := runner.New(myStrategy,
        runner.WithPortfolio(pf),
        runner.WithRiskManager(rm),
        runner.WithData(candleData),
        // Optional: runner.WithResults(&types.Results{}),
        // Optional: To pre-calculate indicators:
        // runner.WithIndicatorConfig(&runner.IndicatorConfig{
        //     Indicators: []interfaces.Indicator{indicators.NewSMA(20), indicators.NewEMA(50)},
        //     LookbackSize: 50, // Ensure enough data for the longest lookback
        // }),
    )

    // 4. Start the Backtest
    err := backtestRunner.Start()
    if err != nil {
        log.Fatalf("Backtest execution failed: %v", err)
    }

    // 5. Access Results (example)
    fmt.Println("Backtest complete.")
    fmt.Printf("Final Portfolio Value: %.2f\n", pf.Value()) 
    // Access more detailed results if WithResults was used or from runner.EquityCurve
}
```

---

## Developer Documentation

This section is for developers working on the `runner` package itself or needing to understand its internal event loop and component interactions.

### Main Files

*   `runner.go`: Contains the `Runner` struct, its main event loop (`Start`, `processTick`), order processing logic (`processOrders`, `processOrder`), and related helper functions. Also defines `IndicatorConfig`.
*   `options.go`: Defines the functional `Option` type (`func(*Runner)`) and provides constructor functions for these options (e.g., `WithPortfolio`, `WithRiskManager`).

### Key Types & Internal Flow

*   **`Runner` (struct)**:
    *   **Core Components**: Holds references to `interfaces.Strategy`, `interfaces.PortfolioManager`, and `interfaces.RiskManager`.
    *   **Data**: Stores the input `*types.TimeseriesTable[core.Candle]`.
    *   **State**: Manages `CurrentTime` and accumulates `EquityCurve ([]types.AccountValue)`.
    *   **`IndicatorConfig`**: If provided, this struct guides the pre-calculation of specified indicators.
*   **`Option` (func type)**: Enables the functional options pattern for flexible `Runner` configuration.
*   **`New(...)`**: Constructor for `Runner`. Takes the strategy and a variadic number of `Option` functions.
*   **`Start()` Method (Core Loop)**:
    1.  Validates that all essential components (Strategy, Portfolio, RiskManager, Data) are set.
    2.  If `IndicatorConfig` is present and `Run()` method is used (as per a previous version of code, or if `ApplyIndicators` is called on data), indicators might be pre-calculated on the `Data` table.
    3.  Iterates through each row (timestamp) in the `Data` (`TimeseriesTable`).
    4.  For each tick (row):
        *   Calls `processTick(data map[string]core.Candle)`.
        *   Updates the `EquityCurve` with the current portfolio snapshot.
*   **`processTick()` Method**:
    1.  Updates open positions in the `PortfolioManager` with current market prices from the tick's data.
    2.  Calls `RiskManager.CheckPositionExits()` to see if any open positions should be closed based on risk rules (e.g., stop-loss). Processes any resulting exit orders.
    3.  Calls `Strategy.OnTick()` with the current market data.
    4.  Processes any new `portfolio.Order` objects returned by the strategy via `processOrders()`.
*   **`processOrder()` Method**:
    1.  Validates the order against the `RiskManager` (`ValidateOrder`).
    2.  If valid, processes the order via `PortfolioManager.ProcessOrder()`.
    3.  Notifies the `Strategy` by calling `OnOrderFilled()`.
*   **`IndicatorConfig` Usage**: The `Runner` struct includes an `IndicatorConfig` field. If this field is populated, the runner can use this configuration to pre-calculate indicators on the dataset. This might involve calling a method like `ApplyIndicators` on the `TimeseriesTable` before the main event loop, ensuring that `core.Candle` objects have indicator values populated when `Strategy.OnTick` is called.

### Extensibility

*   **Custom Event Handling**: Advanced users might want to modify the event loop for different simulation types (e.g., tick-by-tick simulation instead of candle-based, or incorporating live data feeds).
*   **New Options**: The functional options pattern makes it easy to add new configuration parameters to the `Runner` without breaking existing code.
*   **Lifecycle Hooks**: The `Runner` could be extended to support more lifecycle hooks for strategies or other components (e.g., `OnBacktestStart`, `OnBacktestEnd`). 