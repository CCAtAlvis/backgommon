# `portfolio` Package

**Purpose:**
Manages all aspects of portfolio state, including cash, open/closed positions, order processing, and portfolio-level statistics. Provides types and logic for positions, orders, and portfolio analytics.

---

## User Documentation

This section is for developers who need to understand how the portfolio system works from the perspective of a strategy implementer, or for those who might interact with portfolio objects for analysis.

### Key Concepts for Users

*   **`Order`**: Represents a trading order that your strategy generates.
    *   Key fields: `Instrument` (e.g., "AAPL"), `Side` (`Long` or `Short`), `Type` (`Entry` or `Exit`), `Quantity`, `Leverage`.
    *   Created using `portfolio.NewOrder(...)`.
    *   Your strategy returns `[]portfolio.Order` from `OnTick`.
*   **`Position`**: Represents an open or closed trading position in a specific instrument.
    *   Tracks details like `OpenPrice`, `Quantity`, `Side`, `RealizedPnL`, `UnrealizedPnL`.
    *   You generally don't create `Position` objects directly; they are managed by the `Portfolio`.
*   **`Portfolio`**: The main component that manages your simulated trading account.
    *   It processes `Order` objects from your strategy.
    *   Tracks `cash`, `openPositions`, and `closedPositions`.
    *   While your strategy is given a `PortfolioManager` interface (which the `Portfolio` implements), direct manipulation from the strategy is usually limited. The primary interaction is submitting orders and receiving updates via `OnOrderFilled`, `OnPositionOpened`, `OnPositionClosed` callbacks.
*   **`Settings`**: Configuration for the portfolio, such as `InitialCapital`, `AllowShorts`, `MaxPositions`.

### Interaction from a Strategy

1.  **Generating Orders**: Your main interaction is creating `Order` objects.
    ```go
    // In your strategy's OnTick method
    orders := make([]portfolio.Order, 0)
    if entryConditionMet {
        buyOrder := portfolio.NewOrder("AAPL", portfolio.Long, portfolio.Entry, 100, 1.0) // 1.0 leverage
        // Set order.Price if it's a limit order, or it might be filled at current market by portfolio
        orders = append(orders, buyOrder)
    }
    return orders
    ```
2.  **Receiving Updates**: Implement the `Strategy` interface callbacks:
    *   `OnOrderFilled(order portfolio.Order)`: To know when an order is executed.
    *   `OnPositionOpened(position portfolio.Position)`: To get details of a newly opened position.
    *   `OnPositionClosed(position portfolio.Position)`: To get details of a closed position and its final PnL.
3.  **Querying Portfolio State (Limited)**: While the `Strategy` interface receives a `PortfolioManager`, and you can call methods like `Value()`, `Cash()`, or `Positions()`, it's often better practice to rely on the state passed through `OnTick` (candles) and the callbacks for a cleaner separation of concerns.

### Portfolio Analytics

*   The `Portfolio` object itself (or through the `PortfolioManager` interface) exposes methods to get its current `Value()`, `Cash()`, list of `Positions()`, etc.
*   The `portfolio.go` file also defines `PositionMetrics` and `PortfolioStats` structs. These are more for post-backtest analysis or detailed reporting, usually populated by the `Portfolio` or `Runner`.

---

## Developer Documentation

This section is for developers working on the `portfolio` package itself or needing to understand its internal mechanics.

### Main Files

*   `portfolio.go`: Contains the main `Portfolio` struct and its logic for managing cash, positions, processing orders, and calculating overall portfolio statistics. Defines `Settings`, `PositionMetrics`, `PortfolioStats`.
*   `position.go`: Defines the `Position` struct, `PositionStatus` enum, and methods for managing individual positions (e.g., adding orders to an existing position, updating PnL, tracking metrics like ROI, duration).
*   `order.go`: Defines the `Order` struct, `OrderSide` and `OrderType` enums, and helper functions like `NewOrder` and `Fill`.

### Key Types & Internal Flow

*   **`Portfolio` (struct)**:
    *   **State**: Holds `cash`, `openPositions (map[string]*Position)`, `closedPositions ([]*Position)`, `orderHistory ([]Order)`, and `settings (*Settings)`.
    *   **Order Processing (`ProcessOrder`)**: This is a core method.
        1.  Validates the order (e.g., sufficient cash, allowed shorts, max positions).
        2.  If an entry order for a new instrument, creates a new `Position`.
        3.  If an entry order for an existing position, updates the `Position` (e.g., average price, quantity).
        4.  If an exit order, updates the `Position`, calculates realized PnL, and potentially moves the position from open to closed.
        5.  Adjusts `cash` based on the order.
    *   **Position Updates (`UpdatePositions`)**: Takes a map of current market prices and updates the `UnrealizedPnL` and other price-dependent metrics for all open positions.
*   **`Position` (struct)**:
    *   **Lifecycle**: Created by an entry `Order`. Modified by subsequent entry/exit `Order`s for the same instrument. Status changes from `Open` to `PartiallyOpen` (if applicable) to `Closed`.
    *   **Metrics**: Tracks `OpenPrice`, `ClosePrice`, `Quantity`, `Leverage`, `RealizedPnL`, `UnrealizedPnL`, `MaxDrawdown`, `HighestPrice`, `LowestPrice`, etc.
    *   `UpdatePrice(currentPrice float64)` is called to refresh unrealized PnL and other market-dependent metrics.
*   **`Order` (struct)**:
    *   Primarily a data carrier. `NewOrder` helps in its creation. The `Fill` method (if used) marks when an order is considered executed at a specific price and time, though much of the fill logic might reside within the `Portfolio.ProcessOrder`.
*   **`Settings` (struct)**: Simple configuration holder for portfolio behavior.
*   **`PositionMetrics`, `PortfolioStats`**: Structs for holding aggregated analytics, typically populated by methods within `Portfolio`.

### Extensibility

*   **Custom `PortfolioManager`**: If the default `Portfolio` logic is insufficient (e.g., for specific brokerage simulations, margin rules, or fee models), one can implement the `interfaces.PortfolioManager` with custom logic.
*   **Fee Models/Slippage**: The current `Portfolio.ProcessOrder` is a good place to integrate more complex fee models or slippage simulation if desired.
*   **Advanced Order Types**: If more order types beyond simple entry/exit are needed (e.g., limit, stop-limit), the `Order` type and `Portfolio.ProcessOrder` logic would need extension. 