# `types` Package

**Purpose:**
Provides core data structures for time series, tabular data, and backtest results. These types are used throughout the framework for storing and manipulating market data, indicator values, and analytics.

---

## User Documentation

This section is for developers using the `types` package to manage data within their trading strategies or analysis tools.

### Key Data Structures for Users

*   **`TimeseriesTable[T]`**: This is a primary data structure you'll interact with, especially for market data.
    *   It's generic, meaning it can hold different types of time-series data, but it's commonly used with `core.Candle`.
    *   Use it to store, access, and iterate over time-indexed data like OHLCV candles.
    *   It supports attaching and calculating technical indicators directly on the table.
*   **`Table`**: A more general-purpose table structure for arbitrary tabular data.
    *   Useful if you need to manage non-time-series data with dynamic columns and rows.
*   **`Results`**: A struct that holds summary statistics and analytics from a backtest run.
    *   You'll typically receive this from the `Runner` after a backtest.
    *   Contains fields like `InitialCapital`, `FinalCapital`, `TotalTrades`, `MaxDrawdown`, etc.
*   **`AccountValue`**: Represents a snapshot of your portfolio's value and composition at a specific point in time.
    *   The `Runner` often generates a series of these to create an equity curve.

### Common Usage Examples

1.  **Creating a TimeseriesTable for Candles**:
    ```go
    // Define columns for your candle data
    candleColumns := []string{"Open", "High", "Low", "Close", "Volume"}
    // Create a new TimeseriesTable to hold core.Candle objects
    candleData := types.NewTimeseriesTable[core.Candle](candleColumns)

    // Later, you would populate this table with actual candle data
    // e.g., candleData.AddRow(timestamp, map[string]core.Candle{...})
    ```

2.  **Initializing Backtest Results**:
    ```go
    // When setting up a backtest, you might initialize a Results struct
    backtestResults := types.Results{
        InitialCapital: 10000.00,
        StartTime:      time.Now(), // Placeholder
    }
    // This struct would be populated by the backtesting engine.
    ```

3.  **Working with `TimeseriesTable`**:
    *   **Adding Data**: Rows are added with a timestamp and a map of column names to values.
    *   **Iterating**: You can iterate over rows in chronological order.
    *   **Accessing Data**: Retrieve rows by timestamp or specific values by timestamp and column.

---

## Developer Documentation

This section is for developers looking to understand the internals of the `types` package or extend its data structures.

### Main Files

*   `timeseries_table.go`: Implements the generic `TimeseriesTable[T]` for time-indexed data. Includes logic for managing timestamps, sorting, and applying indicators.
*   `table.go`: Implements the generic `Table` structure for tabular data with dynamic columns and rows.
*   `results.go`: Defines the `Results` struct for storing backtest summary statistics and the `AccountValue` struct for equity curve points.

### Key Types & Design Considerations

*   **`TimeseriesTable[T]`**:
    *   **Generics**: Uses Go generics (`[T any]`) to provide a type-safe way to handle different kinds of time-series records, though `core.Candle` is a primary use case.
    *   **Internal Structure**: Internally, it often wraps or uses a `Table` and adds timestamp mapping (`map[time.Time]int`) for efficient time-based lookups and an array of timestamps (`[]time.Time`) for ordered iteration.
    *   **Indicator Support**: Contains methods to apply `interfaces.Indicator` instances to its data, facilitating the calculation and storage of indicator values alongside the base time-series data (e.g., candles).
*   **`Table`**:
    *   **Flexibility**: Designed as a flexible container for data that doesn't necessarily have a time component or where rows might have heterogeneous types (though column values are typically consistent within a column).
    *   **Dynamic Columns**: May support adding columns dynamically.
*   **`Results`**:
    *   **Aggregation**: Primarily a data holder for aggregated metrics generated at the end of a backtest.
    *   Contains fields for common performance metrics like Sharpe Ratio, Sortino Ratio, drawdown, etc.
*   **`AccountValue`**:
    *   **Snapshot**: Represents a point-in-time snapshot of account metrics. Essential for building equity curves and detailed performance tracking over time.

### Usage Notes & Internal Flow

*   **Primary Data Container**: `TimeseriesTable` is the workhorse for market data and indicator values during a backtest simulation run by the `Runner`.
*   **Data Integrity**: The `TimeseriesTable` usually ensures that data is sorted by time before iteration, often using a dirty flag and re-sorting when new data is added or when an iterator is requested.
*   **Analytics**: `Results` and `AccountValue` are primarily used for post-backtest analysis and reporting. The `Runner` populates these structures.

### Extending the Package

*   **Custom Data Types in `TimeseriesTable`**: While `core.Candle` is common, `TimeseriesTable[T]` can be used with other custom structs `T` if they fit a time-series model.
*   **New Analytics in `Results`**: The `Results` struct could be extended with more advanced financial metrics if needed.
*   **Alternative Table Implementations**: If a different underlying table storage or performance characteristic is needed, one might consider alternative implementations, though the current `Table` aims for general utility. 