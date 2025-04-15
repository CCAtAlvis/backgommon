# Design Approach

## Strategy Implementation

Backgommon uses an embedded struct-based approach for implementing trading strategies. This decision was made to balance simplicity, flexibility, and maintainability.

### Why Embedded Structs Over Direct Functions?

1. **Inheritance with Flexibility**
   - Users only need to implement the strategy function
   - No need to understand and implement entire interfaces
   - Can start with minimal boilerplate code
   - Base strategy provides default implementations
   - Users can selectively override only what they need
   - Maintains all the benefits of composition
   - No need to implement unused methods

2. **State Management**
   - Can hot-swap individual functions
   - Easy to modify specific behaviors without reimplementing everything
   - Functions can be composed and reused easily
   - Easy to maintain strategy-specific state
   - Clean access to shared resources (Portfolio, Settings)
   - Better encapsulation of strategy logic
   - Type-safe field access

3. **Developer Experience**
   - Users can focus on their trading logic
   - Framework handles all the underlying complexity
   - Clear separation between strategy and framework internals
   - Better IDE support with type hints
   - Clear structure for strategy implementation
   - Easy to extend with new methods
   - Natural organization of related code

4. **Default Implementations**
   - Framework provides sensible defaults for common operations
   - Users can override only what they need
   - Reduces cognitive load on users
   - BaseStrategy provides sensible defaults
   - No boilerplate for unused methods
   - Easy to add new lifecycle hooks
   - Consistent behavior across strategies

### Example Usage

Simple strategy implementation:
```go
// Define your strategy
type MyStrategy struct {
    strategy.BaseStrategy  // Embed base strategy

    // Add strategy-specific fields
    Symbol string
    Period int
}

// Implement only what you need
func (s *MyStrategy) OnTick(data map[string]Candle) []order.Order {
    var orders []order.Order
    
    // Strategy logic here
    if data[s.Symbol].Close > data[s.Symbol].Open {
        orders = append(orders, order.NewOrder(
            s.Symbol,
            order.Long,
            order.Entry,
            100,
            1.0,
        ))
    }
    
    return orders
}

// Use the strategy
strategy := &MyStrategy{
    Symbol: "AAPL",
    Period: 14,
}

// Create runner with strategy
settings := structs.NewDefaultSettings()
runner := NewRunner(strategy, settings)
```

### Framework Design Philosophy

1. **Unix Philosophy**
   - Do one thing well (strategy implementation)
   - Compose with other parts (order management, portfolio tracking)
   - Keep components loosely coupled

1. **Composition Over Inheritance**
   - Use embedding for default implementations
   - Keep components loosely coupled
   - Allow for easy extension
   - Maintain flexibility

2. **Convention Over Configuration**
   - Provide sensible defaults via BaseStrategy
   - Make common operations easy
   - Keep complex operations possible
   - Clear patterns for customization

3. **Progressive Complexity**
   - Start simple with basic strategy
   - Add complexity only when needed
   - Framework shouldn't get in the way
   - Clear path for adding features
   - No penalty for simple strategies

### Available Strategy Methods

1. **Core Methods**
   - `OnTick(data) []Order` - Main strategy logic
   - `OnOrderFilled(order)` - Handle filled orders
   - `OnPositionOpened(position)` - New position opened
   - `OnPositionClosed(position)` - Position closed

2. **Time-Based Methods**
   - `OnDayStart(date)` - Start of trading day
   - `OnDayEnd(date)` - End of trading day

### Future Considerations

While the current embedded struct approach serves well, we may consider:

1. More helper methods in BaseStrategy
2. Additional lifecycle hooks
3. Better debugging and logging support
4. Enhanced documentation and examples

The goal remains to maintain simplicity while providing power users with the tools they need.
