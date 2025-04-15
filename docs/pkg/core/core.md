# `core` Package

**Purpose:**
Defines the fundamental data structures and interfaces for market data and indicator values. This package is the foundation for all time-series and indicator operations in the framework.

---

## User Documentation

This section is for developers using the `core` types in their trading strategies or when interacting with other Backgommon packages.

### Key Data Structures for Users

*   **`Candle`**: This is the primary representation of market data for a single time period.
    *   It contains standard OHLCV (Open, High, Low, Close, Volume) data.
    *   Crucially, it can also store calculated indicator values associated with that specific candle (time period).
    *   You'll receive `Candle` objects (often in a `map[string]core.Candle` from the `Runner` via `OnTick`) as input to your strategy.
*   **`Value`**: This is an interface that all indicator output types must implement.
    *   The main method is `Value() float64`, which returns the primary numerical value of an indicator (e.g., the SMA value, the MACD line value).
    *   When you retrieve an indicator from a `Candle` using `GetIndicator()`, or when an `Indicator` calculates its result, you get a `core.Value`.

### Common Usage Examples

1.  **Accessing Candle Data in a Strategy**:
    ```go
    // Inside your strategy's OnTick(data map[string]core.Candle) method
    // Assuming "AAPL" is one of the symbols you're tracking
    if aaplCandle, ok := data["AAPL"]; ok {
        fmt.Printf("AAPL Close: %.2f\n", aaplCandle.Close)
        // aaplCandle is of type core.Candle
    }
    ```

2.  **Working with Indicator Values on a Candle**:
    ```go
    // Assuming ema20Value is a core.Value calculated by an EMA(20) indicator
    // And currentCandle is a *core.Candle or core.Candle

    // Storing an indicator value on a candle:
    // (This is often done by the TimeseriesTable or Runner when applying indicators)
    currentCandle.SetIndicator("EMA_20", ema20Value) // "EMA_20" is from ema20.Name()

    // Retrieving an indicator value from a candle in your strategy:
    retrievedValue, err := currentCandle.GetIndicator("EMA_20")
    if err == nil {
        fmt.Printf("Retrieved EMA(20): %.2f\n", retrievedValue.Value())
    } else {
        // Handle case where indicator is not found on the candle
    }
    ```

---

## Developer Documentation

This section is for developers looking to understand the internal structure of the `core` package or how its types are fundamental to the framework.

### Main Files

*   `candle.go`: Defines the `Candle` type (OHLCV data) and methods for storing/retrieving indicator values per candle.
*   `indicator.go`: Defines the `Value` interface, which all indicator output types (like `indicators.SingleValue` or `indicators.MACDValue`) must implement.

### Key Types & Design Considerations

*   **`Candle`**:
    *   **Structure**: A struct holding `time.Time`, `Open`, `High`, `Low`, `Close` (float64), `Volume` (int64), and an internal `map[string]Value` named `indicators`.
    *   **Indicator Storage**: The `indicators` map uses the string name of an indicator (from `interfaces.Indicator.Name()`) as the key to store its calculated `core.Value` for that candle's specific time period.
    *   **Mutability**: `Candle` instances (or pointers to them) are passed around. The `SetIndicator` method modifies the internal map. `NewCandle()` initializes an empty candle with an initialized indicators map.
*   **`Value` (interface)**:
    *   **Purpose**: Provides a common way to access the primary float64 output of any indicator.
    *   **Implementation**: Implemented by types in the `indicators` package such as `indicators.SingleValue` and `indicators.MACDValue`. This allows `Candle` to store various indicator results generically while still providing a direct way to get the main numerical output.

### Usage Notes & Internal Flow

*   **Fundamental Unit**: `Candle` is the fundamental unit of market data processed by strategies and many other parts of the framework.
*   **Data Enrichment**: The design of `Candle` allowing indicator storage is key to how Backgommon enables strategies to access pre-calculated or on-the-fly indicator data efficiently per time step.
*   **Decoupling**: The `core.Value` interface decouples the `core` package (and `Candle`) from the concrete indicator value types defined in `pkg/indicators`. This is good design, as `core` only needs to know that an indicator produces *some* value that can be represented as a float64, not the specifics of how many sub-values it might have (like MACD).

### Extending the Package

*   **`Candle` Structure**: Modifications to `Candle` would have widespread impact and should be considered carefully. Generally, it holds the standard set of information needed for technical analysis.
*   **`Value` Interface**: This interface is intentionally minimal. If indicators needed to expose more complex, structured data through `Candle` in a generic way, this interface or related patterns might need to be revisited, but the current design favors simplicity for the most common use case (getting a primary float value). 