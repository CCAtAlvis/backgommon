# Code Review Observations & Refactoring Notes

**Date:** 2026-05-26
**Scope:** Full audit of all packages in `pkg/`

This document captures design issues, bugs, redundancies, confusing patterns, and refactoring suggestions discovered during the source code documentation effort. Items are categorized by severity and grouped by package.

---

## Critical — Bugs & Data Integrity

### 1. `types/table.go` — `GetColumnValues` returns double-length slice

**File:** `pkg/types/table.go`
**Issue:** The function creates a slice with `make([]interface{}, t.NumRows())` (pre-filling with nils), then *appends* values. The returned slice has `NumRows` nil entries followed by `NumRows` actual values — double the expected length.
**Impact:** Any caller iterating the returned slice gets unexpected nil entries at the front.
**Fix:** Either use `make([]interface{}, 0, t.NumRows())` or index-assign into the pre-allocated slice.

### 2. `portfolio/order.go` & `portfolio/position.go` — `time.Now()` in a backtesting library

**File:** `pkg/portfolio/order.go` (line ~85, `Fill` method), `pkg/portfolio/position.go` (line ~62, `NewPosition`)
**Issue:** Both fall back to `time.Now()` for timestamps. In a backtest, the wall clock is meaningless — all times should come from the simulation clock (`runner.CurrentTime`).
**Impact:** If `FilledAt` is not explicitly set before calling `Fill()`, the order gets a real-world timestamp mixed in with simulated dates. Currently the runner does set `FilledAt` before calling `Fill`, so this is a latent bug (only triggered if someone calls `Fill()` directly without setting `FilledAt`).
**Fix:** Remove the `time.Now()` fallback; require explicit timestamp or accept it as a parameter.

### 3. `types/timeseries_table.go` — Generic `[T any]` is misleading

**File:** `pkg/types/timeseries_table.go`
**Issue:** `TimeseriesTable[T any]` is declared as generic, but `ApplyIndicator*` methods hard-code type assertions to `core.Candle` / `*core.Candle`. The generic parameter is a false promise — indicator functionality only works when `T` is `core.Candle`.
**Impact:** Users might instantiate `TimeseriesTable[MyCustomType]` and discover at runtime that indicators don't work.
**Fix options:**
  1. Constrain the generic: `TimeseriesTable[T core.Candle]` (but Go constraints don't work this way for concrete types)
  2. Move indicator methods to a separate `CandleTable` type alias
  3. Document the limitation prominently (chosen for now)

---

## High — Design & Consistency Issues

### 4. `portfolio/portfolio.go` — `Positions()` returns live mutable map

**File:** `pkg/portfolio/portfolio.go`
**Issue:** `Positions()` returns the internal `map[string]*portfolio.Position` directly. Callers can mutate portfolio internals. Meanwhile, `ClosedPositions()` returns a defensive copy.
**Impact:** Inconsistent safety contract. A strategy could accidentally corrupt portfolio state by modifying the returned map.
**Fix:** Return a copy (like `ClosedPositions`), or document that mutation is intentional and expected.

### 5. `risk/exit_conditions.go` — Hidden mutation in a "check" method

**File:** `pkg/risk/exit_conditions.go` (lines ~41, ~72)
**Issue:** `checkLongExitConditions` and `checkShortExitConditions` mutate `pos.TrailingStopHigh` as a side-effect. This means calling `checkExitConditions` twice with the same position yields different results. The function name ("check") implies a pure read operation.
**Impact:** Confusing semantics; difficult to test; violates principle of least surprise.
**Fix:** Either rename to `updateAndCheckExitConditions` or extract the trailing-stop-high update into a separate explicit step.

### 6. `risk/` package — Hardcoded `fmt.Printf` logging

**Files:** `pkg/risk/manager.go` (lines ~157-158), `pkg/risk/hooks.go` (lines ~129, 134, 138, 143)
**Issue:** Risk evaluation and exit-condition methods print directly to stdout. There is no way to silence, redirect, or format these messages.
**Impact:** Clutters output in production; impossible to capture structured logs; makes testing harder.
**Fix:** Introduce an injectable logger interface (or use `log/slog`) on the Manager, defaulting to a no-op or stdout logger.

### 7. `output/html.go` — Indian number formatting hard-coded in framework

**File:** `pkg/output/html.go` (`fmtNum` function)
**Issue:** The `fmtNum` JavaScript function formats numbers using Crore (10^7) and Lakh (10^5) — Indian numbering conventions. This is baked into framework-level reporting code.
**Impact:** International users get unfamiliar number formatting. The framework should be locale-neutral.
**Fix:** Make the number formatter configurable (accept a locale/format function in ReportData), or use standard international suffixes (M for millions, K for thousands).

### 8. `portfolio/portfolio.go` — Duplicate `Stats()` / `GetPortfolioStats()`

**File:** `pkg/portfolio/portfolio.go`
**Issue:** Both `Stats()` and `GetPortfolioStats()` are exported. `Stats()` simply calls `GetPortfolioStats()`. The `Get` prefix is non-idiomatic in Go.
**Impact:** Confusing API surface; users don't know which to call.
**Fix:** Deprecate `GetPortfolioStats()` in favor of `Stats()`, or remove the duplication.

---

## Medium — Dead Code & Redundancy

### 9. `portfolio/costs.go` — `TradeCosts` struct is unused

**File:** `pkg/portfolio/costs.go`
**Issue:** The `TradeCosts` struct is declared but never referenced anywhere in the codebase.
**Fix:** Remove it, or document its intended future use.

### 10. `portfolio/costs.go` — Deprecated `Compute*` functions still present

**File:** `pkg/portfolio/costs.go`
**Issue:** `ComputeBrokerage`, `ComputeTransactionTax`, `ComputeCapitalGainsTax` are all marked deprecated but still exist. They add clutter and confusion.
**Fix:** Remove in next major version, or move to a `compat` file with clear removal timeline.

### 11. `runner/runner.go` — `preCalculateIndicators` appears dead

**File:** `pkg/runner/runner.go`
**Issue:** The `preCalculateIndicators` method operates on `map[string]*core.Candle` but the main loop operates on `map[string]core.Candle` (value, not pointer). The `Start()` method uses `preRunIndicators` via `Data.ApplyIndicators()` directly. This method appears unreachable.
**Fix:** Verify and remove if dead, or fix the signature mismatch.

### 12. `risk/exit_conditions.go` — Duplicated arithmetic

**File:** `pkg/risk/exit_conditions.go`
**Issue:** The `calculateStopLossPrice`, `calculateTakeProfitPrice`, `calculateTrailingStopPrice` helper functions duplicate the same arithmetic that already exists inline in `checkLongExitConditions` / `checkShortExitConditions`.
**Fix:** Have the check methods call the helper functions instead of duplicating the math.

### 13. `risk/exit_conditions.go` — `createExitOrder` discards exit reason

**File:** `pkg/risk/exit_conditions.go`
**Issue:** `createExitOrder` accepts a `reason` string parameter but the `Order` type has no field to store it. The reason is logged in `CheckPositionExits` but lost from the order itself.
**Fix:** Add a `Reason` or `Metadata` field to `portfolio.Order` so downstream code (reports, callbacks) can access why an exit was triggered.

---

## Low — Naming, Style & Placement

### 14. `runner/results.go` — Magic number `252` (trading days/year)

**File:** `pkg/runner/results.go`
**Issue:** The number 252 appears twice without being a named constant. It represents US equity market trading days per year and is used for annualization.
**Fix:** Extract to `const tradingDaysPerYear = 252` with a doc comment explaining the convention.

### 15. `runner/results.go` — Magic constant `0.05` for risk-free rate

**File:** `pkg/runner/results.go`
**Issue:** `defaultRiskFreeRate = 0.05` is defined but has no doc comment explaining why 5% was chosen or that it's an assumption.
**Fix:** Add a doc comment.

### 16. `portfolio/execution_settings.go` — Arguably misplaced

**File:** `pkg/portfolio/execution_settings.go`
**Issue:** `ExecutionSettings` configures fill pricing and slippage behavior, but the actual fill/slippage logic lives in `pkg/execution/`. Having it in `pkg/portfolio/` is architecturally odd.
**Impact:** Import cycles prevent moving it without refactoring, but it should be documented why it lives here.
**Fix (future):** Consider moving to `pkg/execution/` or a shared `pkg/config/` package if import cycles can be resolved.

### 17. `execution/slippage.go` — Slippage mode strings are magic strings

**File:** `pkg/execution/slippage.go`
**Issue:** Slippage modes (`"fixedpoints"`, `"percentofprice"`, `"None"`) are compared with mixed casing strategies (`EqualFold` for "None", `ToLower` for others) and have no corresponding `const` declarations — unlike `FillMode` which has proper typed constants.
**Fix:** Define `SlippageMode` typed constants (like `FillMode`) and use them consistently.

### 18. `portfolio/hooks.go` — Misleading filename

**File:** `pkg/portfolio/hooks.go`
**Issue:** Named "hooks" but contains context structs and core interfaces (`CostCalculator`, `CashFlowCalculator`, `PeriodicEventsModel`), not hooks in the traditional event-callback sense.
**Fix:** Rename to `interfaces.go` or `contracts.go`, or add a file-level comment explaining the naming choice.

### 19. `strategy/base.go` — Potentially dead lifecycle methods

**File:** `pkg/strategy/base.go`
**Issue:** `OnPositionOpened` and `OnPositionClosed` are defined on `BaseStrategy` but the `interfaces.Strategy` interface includes them. Need to verify the runner actually calls them (it does via type assertion in `processOrder`).
**Clarification:** These ARE called — but only via direct method call after type-asserting the strategy. This is valid but should be documented in the interface definition.

### 20. `position.go` — `StopLoss` and `TakeProfit` fields unused by framework

**File:** `pkg/portfolio/position.go`
**Issue:** `StopLoss` and `TakeProfit` fields exist on `Position` but are never set by any code in the portfolio package. The risk manager uses `Settings.DefaultStopLossRate` instead of `pos.StopLoss`.
**Impact:** These fields suggest per-position stop/TP configuration that doesn't actually work.
**Fix:** Either wire them up (risk manager checks `pos.StopLoss` if non-zero, falling back to default) or remove them and document that stop/TP is always configured at the settings level.

### 21. `types/table.go` — `Iterator()` goroutine leak risk

**File:** `pkg/types/table.go`, `pkg/types/timeseries_table.go`
**Issue:** `Iterator()` spawns a goroutine that sends rows into a channel. If the caller breaks out of a `for range` loop early (doesn't drain the channel), the goroutine hangs forever.
**Fix:** Document this prominently as a usage contract, or redesign using a callback/closure pattern, or use `context.Context` for cancellation.

---

## Items That May Belong at User/Strategy Level (Not Framework)

1. **Profit pocketing settings** (`IsPocketEnabled`, `MinProfitForPocket`, `PocketPercent`) in `portfolio.Settings` — This is a trading strategy policy, not a portfolio accounting concern.

2. **`GetPositionRisk` and `calculate*` helpers** in `risk/exit_conditions.go` — Convenience utilities that duplicate logic already in exit-check methods. Could live in a user-facing `riskutil` package.

3. **SIP settings** (`SIPAmount`, `SIPFrequency`) in `portfolio.Settings` — Systematic investment plans are user-level cash injection policies, not core portfolio mechanics.

4. **Account lock settings** (`IsAccountLockEnabled`, `AccountLockPercent`) — Strategy-specific risk policy that could be implemented via a custom `DrawdownPolicy` rather than being a first-class setting.

---

## Recommended Refactoring Priority

| Priority | Item | Effort |
|----------|------|--------|
| 1 | Fix `GetColumnValues` bug (#1) | Small |
| 2 | Extract `252` and `0.05` to named constants (#14, #15) | Tiny |
| 3 | Add `Reason` field to `Order` (#13) | Small |
| 4 | Define `SlippageMode` typed constants (#17) | Small |
| 5 | Document `Positions()` mutation contract (#4) | Tiny |
| 6 | Injectable logger for risk package (#6) | Medium |
| 7 | Make number formatter configurable (#7) | Medium |
| 8 | Remove dead code (#9, #10, #11) | Small |
| 9 | Fix `time.Now()` fallbacks (#2) | Small |
| 10 | Resolve generic constraint issue (#3) | Medium-Large |
