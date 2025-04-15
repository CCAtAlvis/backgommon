# Code Structure

## Directory Layout
```
backgommon/
├── cmd/                    # Future command-line tools (empty for now)
├── examples/              # Example strategies and usage
│   ├── simple/
│   │   └── strategy.go    # Simple moving average strategy example
│   └── custom/
│   │   └── strategy.go    # Example with custom components
│   └── README.md          # Examples documentation
├── pkg/                 # Public API
│   ├── runner/          # Core backtesting engine (public interface)
│   │   ├── runner.go    # Main runner implementation
│   │   └── options.go   # Runner configuration options
│   ├── strategy/        # Strategy interface and base implementation
│   │   ├── strategy.go  # Strategy interface
│   │   └── base.go      # Base strategy implementation
│   ├── risk/            # Risk management
│   │   ├── manager.go   # Risk manager interface and default impl
│   │   └── rules.go     # Risk rules and calculations
│   ├── portfolio/       # Portfolio management
│   │   ├── portfolio.go
│   │   ├── position.go
│   │   └── order.go
│   └── types/          # Common types and interfaces
│       ├── candle.go   # Price data structures
│       └── interfaces.go # Core interfaces
├── docs/               # Documentation
│   ├── approach.md     # Design philosophy and approach
│   ├── examples.md     # Usage examples
│   └── code_structure.md # This file
├── go.mod
└── README.md
```

## Package Organization

### pkg/types
Contains core interfaces and data structures used throughout the framework. This is the foundation that other packages build upon.

### pkg/strategy
Strategy interface and base implementation. Users will primarily interact with this package to implement their trading strategies.

### pkg/portfolio
Portfolio management functionality including position tracking, order processing, and P&L calculations.

### pkg/risk
Risk management functionality including position sizing, stop losses, and portfolio-level risk controls.

### pkg/runner
The main backtesting engine that ties everything together. Users create a runner instance to execute their strategies.

## Key Interfaces

Each major component is defined by an interface, allowing users to provide custom implementations:

```go
// Strategy interface defines how trading strategies should behave
type Strategy interface {
    OnTick(data map[string]Candle) []Order
    OnOrderFilled(order Order)
    OnPositionOpened(position Position)
    OnPositionClosed(position Position)
}

// Portfolio manager interface defines portfolio operations
type PortfolioManager interface {
    ProcessOrder(Order) error
    UpdatePositions(map[string]float64)
    Value() float64
}

// Risk manager interface defines risk management operations
type RiskManager interface {
    ValidateOrder(PortfolioManager, Order) error
    CheckPositionExits(PortfolioManager, map[string]float64) []Order
}
```

## Extension Points

The framework is designed to be extensible at several points:

1. **Strategy Implementation**
   - Implement custom trading logic
   - Override specific lifecycle hooks
   - Add strategy-specific state

2. **Risk Management**
   - Custom position sizing
   - Custom exit rules
   - Portfolio-level risk controls

3. **Portfolio Management**
   - Custom order processing
   - Position tracking
   - P&L calculations

4. **Event Handling**
   - Custom logging
   - Performance tracking
   - External integrations
