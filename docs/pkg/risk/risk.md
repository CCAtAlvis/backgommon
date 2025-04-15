# `risk` Package

**Purpose:**
Provides risk management logic, including order validation, position exit rules, and risk metrics. Allows configuration of risk parameters such as max drawdown, leverage, stop loss, and take profit.

---

## User Documentation

This section is for developers who need to configure risk management for their backtests or understand how risk rules are applied.

### Key Concepts for Users

*   **`Manager`**: The main risk management component. An instance of `risk.Manager` implements the `interfaces.RiskManager`.
    *   It's configured with `risk.Settings`.
    *   The `Runner` uses it to validate orders and check for position exits.
*   **`Settings`**: A struct to define various risk parameters.
    *   Examples: `MaxDrawdown` (portfolio level), `MaxLeverage` (per order/position), `UseStopLoss`, `DefaultStopLoss` (percentage), `UseTakeProfit`, `DefaultTakeProfit`, `UseTrailingStop`, `DefaultTrailingStop`.
*   **Order Validation**: Before an order from your strategy is processed by the `Portfolio`, the `RiskManager` validates it (e.g., against max leverage, position size limits defined in `risk.Settings`).
*   **Position Exit Checks**: On each tick, the `RiskManager` checks open positions against exit conditions like stop-loss, take-profit, or trailing stops, if enabled in `Settings`.

### Configuring Risk Management

Typically, you create `risk.Settings`, then a `risk.Manager` with these settings, and provide it to the `Runner`.

```go
import (
    "github.com/CCAtAlvis/backgommon/pkg/risk"
    "github.com/CCAtAlvis/backgommon/pkg/runner"
    // ... other imports
)

// In your backtest setup:
riskSettings := &risk.Settings{
    MaxDrawdown:         0.20, // 20% max portfolio drawdown (Note: actual enforcement might be portfolio-level or require more logic)
    MaxLeverage:         10.0, // Max leverage per trade
    UseStopLoss:         true,
    DefaultStopLoss:     0.05, // 5% stop loss from entry price
    UseTakeProfit:       true,
    DefaultTakeProfit:   0.10, // 10% take profit from entry price
    UseTrailingStop:     true,
    DefaultTrailingStop: 0.03, // 3% trailing stop from the peak
    // MaxPositions: 5, // Could be a risk setting or portfolio setting
    // MaxPositionSize: 0.1, // e.g., max 10% of portfolio value in one position
    // MinPositionSize: 100.0, // e.g., min position value in currency
}

riskManager := risk.New(riskSettings)

// Provide to runner
// backtestRunner := runner.New(myStrategy, runner.WithRiskManager(riskManager), ...)
```

Your strategy generally doesn't interact directly with the `RiskManager`. It submits orders, and the `Runner` uses the `RiskManager` to enforce the configured rules.

---

## Developer Documentation

This section is for developers working on the `risk` package itself or needing to understand its internal logic for validation and exit conditions.

### Main Files

*   `manager.go`: Defines the `Manager` struct (which implements `interfaces.RiskManager`) and the `Settings` struct. Contains the primary logic for `ValidateOrder` and `CheckPositionExits`.
*   `exit_conditions.go`: Contains the detailed logic for checking specific exit conditions like stop-loss, take-profit, and trailing-stop (e.g., `checkLongExitConditions`, `checkShortExitConditions`). Defines `PositionRisk` and helper functions for risk calculations.

### Key Types & Internal Flow

*   **`Manager` (struct)**:
    *   Holds a pointer to `Settings`.
    *   `ValidateOrder(pf interfaces.PortfolioManager, ord portfolio.Order) error`: Checks order parameters (e.g., leverage, potential position size relative to portfolio value based on `Settings`) against configured limits.
    *   `CheckPositionExits(pf interfaces.PortfolioManager, prices map[string]float64) []portfolio.Order`: Iterates through open positions in the portfolio. For each position, it uses current market `prices` to evaluate exit conditions defined in `exit_conditions.go` (stop-loss, take-profit, trailing-stop) based on `Settings`. If an exit condition is met, it generates an appropriate exit `portfolio.Order`.
*   **`Settings` (struct)**:
    *   A data struct holding various configurable risk parameters. These parameters drive the behavior of `ValidateOrder` and `CheckPositionExits`.
*   **`PositionRisk` (struct)**:
    *   Defined in `exit_conditions.go`. Holds calculated risk metrics for a specific position, such as `StopLossPrice`, `TakeProfitPrice`, `MaxLoss`, `RiskRewardRatio`.
    *   Can be used for more detailed risk analysis of individual positions, though not directly part of the core exit check loop in the same way `Settings` are.
*   **Exit Condition Logic (`exit_conditions.go`)**:
    *   `checkExitConditions` (method on `Manager`, but logic might be largely in this file) is the main entry point called by `CheckPositionExits`.
    *   It dispatches to side-specific checks (e.g., `checkLongExitConditions`, `checkShortExitConditions`).
    *   These functions compare the `currentPrice` of an asset against calculated stop-loss, take-profit, and trailing stop levels derived from the position's `OpenPrice` (or `TrailingStopHigh`) and the percentages in `Settings`.

### Extensibility

*   **Custom Risk Rules**: To add new types of order validation or position exit rules:
    *   Extend the `Settings` struct with new parameters if needed.
    *   Modify `ValidateOrder` in `manager.go` to include new validation checks.
    *   Add new functions in `exit_conditions.go` for new exit criteria and call them from `checkExitConditions` or its sub-functions.
*   **Portfolio-Level Risk**: Current `MaxDrawdown` in `Settings` is a parameter. Actual enforcement of portfolio-level drawdown might require interaction with the `PortfolioManager` from within the `RiskManager` or even from the `Runner` by periodically checking portfolio value against a drawdown limit.
*   **Dynamic Risk Settings**: The framework could be extended to allow risk settings to be adjusted dynamically during a backtest, though this would add complexity.
*   **More Sophisticated Risk Metrics**: `PositionRisk` could be expanded, or new structs added, for more advanced risk calculations (e.g., VaR, CVaR for positions or the portfolio). 