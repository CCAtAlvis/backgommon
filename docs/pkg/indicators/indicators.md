# `indicators` Package

**Purpose:**
Provides a collection of technical indicators (e.g., SMA, EMA, MACD) and utilities for use in trading strategies. Indicators are composable and can depend on each other. The package also supports user-defined custom indicators.

---

## User Documentation

This section is for developers using the `indicators` package to implement or analyze trading strategies.

### Key Concepts for Users

*   **Available Indicators**: The package provides standard indicators like `SMA` (Simple Moving Average), `EMA` (Exponential Moving Average), and `MACD` (Moving Average Convergence Divergence).
*   **Custom Indicators**: You can define your own indicators using `NewCustomIndicator`, providing a name, a calculation function (`func([]core.Candle) []any`), and a list of any other `interfaces.Indicator` it depends on.
*   **Indicator Values**: Indicators return a slice of values (`[]any`), one for each input candle. If the indicator cannot provide a value for a candle (e.g., insufficient data), the value at that index will be `nil`. If the indicator can provide a value (including zero), it will be present at the corresponding index.

### Using Indicators in Your Strategy

1.  **Instantiation**: Create instances of the indicators you need, providing necessary parameters like the period.
    ```go
    sma20 := indicators.NewSMA(20)
    ema50 := indicators.NewEMA(50)
    macd := indicators.NewMACD(12, 26, 9)
    ```
2.  **Calculation**: In your strategy (e.g., within `OnTick`), you would typically calculate indicator values using a series of `core.Candle` objects. The `Calculate` method on an indicator takes `[]core.Candle` and returns a `[]any` of values, one per input candle.
    ```go
    // Assuming 'currentCandles' is a slice of core.Candle relevant for the calculation
    smaValues := sma20.Calculate(currentCandles)
    fmt.Println("SMA(20) Values:", smaValues) // Each entry is either nil or a float64

    macdResults := macd.Calculate(currentCandles)
    for i, v := range macdResults {
        if v != nil {
            mv := v.(indicators.MACDValue)
            fmt.Printf("Candle %d: MACD=%.2f, Signal=%.2f\n", i, mv.Value(), mv.Signal())
        }
    }
    ```
3.  **Attaching to Candles**: While indicators can be calculated on the fly, for efficiency, their values can be pre-calculated and attached to `core.Candle` objects (using `candle.SetIndicator(name, value)`). The `name` should be the string from the indicator's `Name()` method (e.g., `"SMA_20"`). Your strategy can then retrieve these using `candle.GetIndicator(name)`.

### Example: Custom Indicator

```go
// myFunc is your custom calculation logic
// e.g., func myFunc(candles []core.Candle) []any { ... return ... }

// Suppose your custom indicator uses an EMA(10)
ema10 := indicators.NewEMA(10)
customInd := indicators.NewCustomIndicator("MyCustomEMAIndicator", myFunc, []interfaces.Indicator{ema10})

// Later, calculate it
// customValues := customInd.Calculate(candles)
```

---

## Developer Documentation

This section is for developers looking to extend the `indicators` package or understand its internals.

### Main Files

*   `sma.go`: Simple Moving Average (SMA) implementation.
*   `ema.go`: Exponential Moving Average (EMA) implementation.
*   `macd.go`: Moving Average Convergence Divergence (MACD) implementation.
*   `custom.go`: Support for user-defined custom indicators.
*   `validator.go`: Utilities for validating indicator dependency graphs (e.g., cycle detection).

### Key Types/Interfaces & Design

*   **`interfaces.Indicator`**: All indicators (e.g., `SMA`, `EMA`, `MACD`, `CustomIndicator`) must implement the `interfaces.Indicator` interface. This requires:
    *   `Calculate(candles []core.Candle) []any`: Computes the indicator's values, returning a slice of length equal to the input candles. Each entry is either the computed value or `nil` if not available.
    *   `Name() string`: Returns a unique identifier for the indicator instance (e.g., `"SMA_20"`). This name is crucial for storing and retrieving indicator values from `core.Candle` objects.
    *   `Dependencies() []interfaces.Indicator`: Returns a slice of other `Indicator` instances that this indicator depends on. For base indicators like `SMA` or `EMA` (when calculated directly, not as part of another like MACD), this might be `nil` or empty.
*   **`SMA`, `EMA`, `MACD`**: Structs implementing common technical indicators.
*   **`CustomIndicator`**: A struct that wraps a user-provided calculation function and its dependencies, allowing easy creation of new indicators without needing to define a new struct for each.
*   **Composition**: Indicators can be composed. For example, `MACD` internally creates and uses `EMA` instances. Its `Dependencies()` method lists these EMAs. This allows for automated dependency resolution if, for example, a system wants to pre-calculate all necessary indicators.
*   **MACD Dependency Calculation**: The `MACD` indicator, for instance, internally creates and relies on three `EMA` instances (fast, slow, and signal). Its `Dependencies()` method formally lists these EMAs, which is useful for automatic dependency resolution systems. During its `Calculate` method, `MACD` will attempt to use pre-calculated EMA values if they are already present on the input `core.Candle` data (retrieved using the respective EMA's `Name()`); otherwise, it will compute them. The signal line's EMA has special handling, as it's calculated over a synthetic series derived from the MACD line itself.
*   **Validation**: `validator.go` provides utilities to help ensure there are no circular dependencies between indicators, which is important for systems that automatically resolve and calculate indicator chains.

### Extending the Package

*   **Adding New Standard Indicators**: To add a new standard indicator (e.g., RSI, Bollinger Bands):
    1.  Create a new Go file (e.g., `rsi.go`).
    2.  Define a struct for the indicator (e.g., `type RSI struct { period int }`).
    3.  Implement the `interfaces.Indicator` interface for this struct.
    4.  Return a slice of values (`[]any`), with `nil` for insufficient data.
*   **Enhancing `validator.go`**: If more complex dependency validation rules are needed. 