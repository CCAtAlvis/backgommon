# `strategy` Package

**Purpose:**
Provides the base implementation for trading strategies, including default method implementations for the `interfaces.Strategy` interface. Users typically embed `BaseStrategy` and override only the methods they need for their custom logic.

---

## User Documentation

This section is for developers implementing their trading strategies using the Backgommon framework.

### Key Concepts for Users

*   **`BaseStrategy`**: This is an embeddable struct that provides default (often no-op) implementations for all methods required by the `interfaces.Strategy` interface, plus a few additional common lifecycle hooks.
    *   You will embed this in your own strategy struct.
*   **Implementing `interfaces.Strategy`**: By embedding `BaseStrategy`, your struct automatically satisfies the `interfaces.Strategy` interface. You then override methods like `OnTick` to implement your specific trading rules.

### Implementing Your Trading Strategy

1.  **Define Your Strategy Struct**: Embed `strategy.BaseStrategy`.
    ```go
    package mystrategy // Your package name

    import (
        "fmt"
        "github.com/CCAtAlvis/backgommon/pkg/core"
        "github.com/CCAtAlvis/backgommon/pkg/interfaces"
        "github.com/CCAtAlvis/backgommon/pkg/portfolio"
        "github.com/CCAtAlvis/backgommon/pkg/strategy"
        // Import any indicators you need
        "github.com/CCAtAlvis/backgommon/pkg/indicators"
    )

    type MyAwesomeStrategy struct {
        strategy.BaseStrategy
        // Custom fields for your strategy
        smaPeriodShort int
        smaPeriodLong  int
        shortSMA       interfaces.Indicator // Store indicator instances
        longSMA        interfaces.Indicator
    }
    ```

2.  **Create a Constructor (Optional but Recommended)**:
    ```go
    func NewMyAwesomeStrategy(shortPeriod, longPeriod int) *MyAwesomeStrategy {
        s := &MyAwesomeStrategy{
            smaPeriodShort: shortPeriod,
            smaPeriodLong:  longPeriod,
            shortSMA:       indicators.NewSMA(shortPeriod), // Initialize indicators
            longSMA:        indicators.NewSMA(longPeriod),
        }
        // s.BaseStrategy.Portfolio will be set by the Runner
        return s
    }
    ```

3.  **Override `OnTick` for Trading Logic**:
    ```go
    func (s *MyAwesomeStrategy) OnTick(data map[string]core.Candle) []portfolio.Order {
        orders := make([]portfolio.Order, 0)

        for instrument, candle := range data {
            // Calculate indicator values using the candle data for this instrument
            // Note: For efficiency with multiple candles, indicators.Calculate might need full history or use pre-calculated values on candles
            // This example assumes indicators can calculate from a single candle or recent history if managed internally or via TimeseriesTable

            // A more robust way for indicators requiring history:
            // 1. Ensure TimeseriesTable is populated with historical data.
            // 2. Use Runner.WithIndicatorConfig to have indicators pre-calculated on candles.
            // 3. Retrieve values: shortSMAVal, err := candle.GetIndicator(s.shortSMA.Name())

            // Simplified example (assuming direct calculation is feasible here or values are on candle)
            // For this example, let's assume we're fetching from candle after runner pre-calculation
            shortVal, shortErr := candle.GetIndicator(s.shortSMA.Name())
            longVal, longErr := candle.GetIndicator(s.longSMA.Name())

            if shortErr != nil || longErr != nil {
                // fmt.Printf("Indicator not ready for %s\n", instrument)
                continue // Or handle error
            }

            fmt.Printf("Instrument: %s, Close: %.2f, Short SMA: %.2f, Long SMA: %.2f\n", 
                instrument, candle.Close, shortVal.Value(), longVal.Value())

            // Example Crossover Logic
            // (This is a naive example; proper state management for crossover is needed)
            if shortVal.Value() > longVal.Value() { // And wasn't already crossed over
                // Check if we already have a position for this instrument
                currentPosition, hasPosition := s.Portfolio.Positions()[instrument]
                if !hasPosition || currentPosition.Side != portfolio.Long {
                    fmt.Printf("BUY signal for %s\n", instrument)
                    // orders = append(orders, portfolio.NewOrder(instrument, portfolio.Long, portfolio.Entry, 100, 1.0 /* price, leverage */))
                }
            } else if shortVal.Value() < longVal.Value() { // And wasn't already crossed under
                currentPosition, hasPosition := s.Portfolio.Positions()[instrument]
                if hasPosition && currentPosition.Side == portfolio.Long {
                     fmt.Printf("SELL signal for %s\n", instrument)
                    // orders = append(orders, portfolio.NewOrder(instrument, portfolio.Short, portfolio.Entry, 100, 1.0 /* price, leverage */)) // If shorting allowed
                    // Or exit long: orders = append(orders, portfolio.NewOrder(instrument, portfolio.Long, portfolio.Exit, currentPosition.Quantity, 1.0))
                }
            }
        }
        return orders
    }
    ```

4.  **Override Other Callbacks (Optional)**: Implement `OnOrderFilled`, `OnPositionOpened`, `OnPositionClosed` if you need to react to these events or manage state based on them.
    ```go
    func (s *MyAwesomeStrategy) OnOrderFilled(order portfolio.Order) {
        fmt.Printf("Strategy: Order filled for %s: %v qty at %.2f\n", order.Instrument, order.Quantity, order.Price)
    }
    ```

---

## Developer Documentation

This section is for developers working on the `strategy` package itself or understanding its role in the framework's architecture.

### Main Files

*   `base.go`: Defines the `BaseStrategy` struct and its default method implementations for the `interfaces.Strategy` interface and other common lifecycle hooks.

### Key Types & Design

*   **`BaseStrategy` (struct)**:
    *   **Embedding**: Designed to be embedded into user-defined strategy structs.
    *   **Default Implementations**: Provides default, often no-op (no operation), implementations for all methods in `interfaces.Strategy` (`OnTick`, `SetPortfolio`, `OnOrderFilled`, `OnPositionOpened`, `OnPositionClosed`).
    *   **Additional Hooks**: It also includes default implementations for other potentially useful lifecycle methods not strictly part of the `interfaces.Strategy` contract, such as `OnDayStart` and `OnDayEnd` (though their invocation by the `Runner` would need to be explicitly implemented if desired).
    *   **Portfolio Access**: Contains a field `Portfolio interfaces.PortfolioManager`, which is set by the `Runner` via the `SetPortfolio` method. This allows the strategy to (cautiously) query portfolio state.

### Role in the Framework

*   **User Convenience**: `BaseStrategy` significantly simplifies the creation of new strategies. Users only need to override the specific methods relevant to their logic, rather than implementing every method of the `interfaces.Strategy` from scratch.
*   **Decoupling Strategy Logic**: It helps in keeping the core strategy logic (e.g., in `OnTick`) separate from the boilerplate of satisfying the full interface contract.
*   **Promoting Convention**: Encourages a common starting point for all strategies within the Backgommon ecosystem.

### Extending the Package

*   **More Default Behaviors**: `BaseStrategy` could be enhanced with more sophisticated default behaviors if a common pattern emerges across many strategies (e.g., basic logging, default position sizing helpers accessible to overriding methods).
*   **Additional Lifecycle Hooks**: If new standard lifecycle events become important for strategies (e.g., `OnWarmupComplete`, `OnParameterChange`), they could be added to `BaseStrategy` (and potentially to the `interfaces.Strategy` if they are fundamental). 