# SMA Crossover Example Strategy Requirements (PRD)

**created by:** user
**created on:** 2024-06-09
**last updated:** 2024-06-09

## Background & Motivation

The purpose of this feature is to implement and test the backbone package by creating a simple, well-known trading strategy: the SMA (Simple Moving Average) Crossover. This will serve as a real-world example to validate the package's core functionality, identify missing or broken components, and provide a reference implementation for future strategies.

## Scope

-   **Functionality:**
    - Implement an SMA crossover strategy using two moving averages (lengths 50 and 200).
    - Buy when SMA(50) crosses above SMA(200).
    - Sell (exit position) when SMA(50) crosses below SMA(200).
    - No shorting; only long or flat positions.
    - Integrate risk management and portfolio management modules.
    - Add a custom field to test extensibility.
-   **Chains/Platforms:**
    - Backgommon backbone package (local test environment).
-   **Providers/Integrations:**
    - Internal backbone package modules for data, execution, risk, and portfolio management.
-   **Out of Scope:**
    - Live trading, external broker integration, or advanced order types.

## Feature Overview / Requirements

1. Calculate SMA(50) and SMA(200) on input price data.
2. Generate a buy signal when SMA(50) crosses above SMA(200).
3. Generate a sell signal (exit) when SMA(50) crosses below SMA(200).
4. Maintain only long or flat positions (no shorting).
5. Integrate risk management (e.g., position sizing, stop-loss).
6. Integrate portfolio management (e.g., capital allocation).
7. Add a custom field to the strategy for demonstration.
8. Provide example code to run the full flow and validate the package.

## Data Structures / Models

```go
// Example (to be refined during implementation)
type SMAStrategyConfig struct {
    ShortWindow int // e.g., 50
    LongWindow  int // e.g., 200
    CustomField string
}
```

## Example Flow / Use Case

- User configures the strategy with SMA windows 50 and 200.
- The strategy processes historical price data.
- When SMA(50) crosses above SMA(200), the strategy issues a buy order.
- When SMA(50) crosses below SMA(200), the strategy issues a sell order (exit).
- Risk and portfolio management modules are invoked as part of order generation.
- The user can observe the full flow and validate the backbone package.

## UI/UX Considerations (Optional)

- Not applicable for this example (CLI or code-based usage).

## API Changes Required (If Applicable)

- None for this example; all internal.

## Implementation Details (Optional Initial Sketch)

- Implement the strategy as a new module or example in the codebase.
- Use existing backbone package interfaces for data, execution, risk, and portfolio management.
- Add a custom field to the strategy struct/config.
- Provide a runnable example to test the end-to-end flow.

### Relevant Files (Initial List)
- `pkg/strategy/sma_crossover.go`
- `examples/sma_crossover_example.go`
- `pkg/risk/`
- `pkg/portfolio/`

### Implementation Gist
- The strategy will be implemented as a new example and/or module, using the backbone package's interfaces. The example will demonstrate the full workflow, including risk and portfolio management, and will help identify any missing or broken functionality in the package. 
