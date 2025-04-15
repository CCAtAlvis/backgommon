# `interfaces` Package

**Purpose:**
Defines the main interfaces for strategies, portfolio management, risk management, and indicators. These interfaces enable extensibility and allow users to provide custom implementations for each major component of the Backgommon framework.

---

## User Documentation

This section is for developers who will be implementing these interfaces, particularly the `Strategy` and `Indicator` interfaces, or those who need to understand the contracts for interacting with core framework components.

### Key Interfaces for Users

*   **`Strategy`**: This is the most important interface for users of the framework. You implement this to define your custom trading logic.
    *   `OnTick(data map[string]core.Candle) []portfolio.Order`: The core method called by the `Runner` on each new data tick (candle). You receive current market data and return a slice of `portfolio.Order` objects to be executed.
    *   `SetPortfolio(portfolio PortfolioManager)`: Called by the `Runner` to inject the portfolio manager instance, allowing your strategy to query portfolio state if needed (though direct portfolio manipulation is usually discouraged in favor of returning Orders).
    *   `OnOrderFilled(order portfolio.Order)`: Callback triggered after an order you issued has been successfully processed (filled) by the portfolio.
    *   `OnPositionOpened(position portfolio.Position)`: Callback triggered when a new position is opened as a result of an order.
    *   `OnPositionClosed(position portfolio.Position)`: Callback triggered when an existing position is closed.
*   **`Indicator`**: Implement this interface if you are creating a custom technical indicator that is not covered by the standard ones or `indicators.CustomIndicator`.
    *   `Calculate(candles []core.Candle) core.Value`: Your logic to compute the indicator's value from a series of candles.
    *   `Name() string`: A unique name for your indicator instance (e.g., "MyRSI_14"). This is used for storing/retrieving from `core.Candle` and for dependency management.
    *   `Dependencies() []Indicator`: A list of other `Indicator` instances that your custom indicator relies upon.

### Implementing Core Components

*   **Strategy Implementation**: Typically, you'll create a struct and embed `strategy.BaseStrategy` (which provides default no-op implementations for all `Strategy` methods), then override the methods you need, especially `OnTick`.
    ```go
    import (
        "github.com/CCAtAlvis/backgommon/pkg/core"
        "github.com/CCAtAlvis/backgommon/pkg/interfaces"
        "github.com/CCAtAlvis/backgommon/pkg/portfolio"
        "github.com/CCAtAlvis/backgommon/pkg/strategy" // For BaseStrategy
    )

    type MyCustomStrategy struct {
        strategy.BaseStrategy
        // ... your custom fields, e.g., parameters, other indicators
    }

    func (s *MyCustomStrategy) OnTick(data map[string]core.Candle) []portfolio.Order {
        // Your trading logic here...
        orders := make([]portfolio.Order, 0)
        // if conditionMet {
        //     orders = append(orders, portfolio.NewOrder(...))
        // }
        return orders
    }
    ```
*   **Custom Portfolio/Risk Managers**: While less common for typical strategy writers, advanced users can provide their own implementations of `PortfolioManager` or `RiskManager` if the default behavior (or the one provided by `pkg/portfolio` and `pkg/risk`) is insufficient. This allows for complete control over order processing, position tracking, and risk validation logic.

---

## Developer Documentation

This section is for developers contributing to the Backgommon framework itself or needing a deeper understanding of its architectural contracts.

### Main Files

*   `interface.go`: Contains all core interface definitions: `Strategy`, `PortfolioManager`, `RiskManager`, and `Indicator`.

### Key Interfaces & Design Philosophy

*   **`Strategy`**: Defines the contract for trading algorithms.
    *   Methods: `OnTick`, `SetPortfolio`, `OnOrderFilled`, `OnPositionOpened`, `OnPositionClosed`.
    *   This interface is the primary extension point for users defining trading logic.
*   **`PortfolioManager`**: Defines the contract for portfolio operations.
    *   Methods: `ProcessOrder`, `UpdatePositions`, `Value`, `Cash`, `Positions`.
    *   Allows for different portfolio management implementations (e.g., handling different asset classes, margin, etc.).
*   **`RiskManager`**: Defines the contract for risk management logic.
    *   Methods: `ValidateOrder`, `CheckPositionExits`.
    *   Enables custom risk validation rules and position exit criteria.
*   **`Indicator`**: Defines the contract for all technical indicators.
    *   Methods: `Calculate`, `Name`, `Dependencies`.
    *   Ensures all indicators can be treated uniformly by the system, especially for calculation, storage on candles, and dependency resolution.

### Role in the Framework

*   **Decoupling**: These interfaces are fundamental to the modularity of Backgommon. They decouple the `Runner` (which orchestrates the backtest) from the concrete implementations of strategies, portfolio managers, risk managers, and indicators.
*   **Extensibility**: They provide clear contracts for users and developers to extend the framework. New strategies, indicators, or even core portfolio/risk logic can be introduced by implementing these interfaces.
*   **Testability**: Interfaces make it easier to test components in isolation by allowing mock implementations.

### Considerations for Implementers (Framework Developers)

*   **Clarity of Contract**: Ensure method signatures and expected behaviors are well-documented (as is done here and in code comments).
*   **Minimality**: Interfaces should generally be minimal, only exposing what is necessary for the interaction. The `Strategy` interface, for example, focuses on the essential lifecycle events.
*   **Dependencies**: Note how interfaces like `Strategy` depend on types from other packages (e.g., `core.Candle`, `portfolio.Order`), establishing clear data flow contracts. 