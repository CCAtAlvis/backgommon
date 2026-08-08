# Execution Plan: Backgommon Framework Migration & Integration

**created by:** user + AI
**created on:** 2026-05-24
**last updated:** 2026-05-24

## Background & Motivation

Backgommon is being rebuilt from a private, strategy-coupled codebase into an open-source, modular Go backtesting framework. GP-Trade will consume it as the engine for proprietary NSE strategies and data.

This plan covers: fixing the broken build, completing the reference SMA strategy, hardening core packages, integrating gp-trade, and open-source readiness.

## Current State (as of 2026-05-24)

- `pkg/` architecture exists: indicators, core, types, interfaces, portfolio, risk, runner, strategy
- Old `src/` layout deleted; gp-trade `strategy1` still imports removed paths
- `go build ./...` **fails** — `pkg/risk/exit_conditions.go` uses old Settings field names
- Examples partially written but out of sync with current APIs
- SMA crossover example exists but uses outdated settings + level-check (not crossover)
- Runner core loop works; Results not populated; portfolio realism mostly TODO
- Tests: only `ema_test.go`, `macd_test.go`
- README contains API credentials — must be removed before public release

## Scope

**In scope:**
- Fix build, complete reference strategy, harden framework, gp-trade integration, docs

**Out of scope (Phase 6+):**
- Live trading web server, Monte Carlo, HFT optimizations

---

## Phase 0: Unblock the Build

**Goal:** `go build ./...` and `go test ./...` pass in backgommon.

### 0.1 Fix risk package field name mismatch
- [x] Update `pkg/risk/exit_conditions.go` to use new Settings fields:
  - `UseStopLoss` → `EnableStopLoss`
  - `DefaultStopLoss` → `DefaultStopLossRate`
  - `UseTakeProfit` → `EnableTakeProfit`
  - `DefaultTakeProfit` → `DefaultTakeProfitRate`
  - `UseTrailingStop` → `EnableTrailingStop`
  - `DefaultTrailingStop` → `DefaultTrailingStopRate`
- [x] Update `GetPositionRisk` and helper functions in same file

### 0.2 Fix examples to current API
- [x] `examples/strategies/sma_crossover/main.go` — update portfolio/risk Settings field names
- [x] `examples/indicator_example/main.go` — adapt to `Calculate() []any`
- [x] `examples/indicator_strategy/strategy.go` — fix typos, restore custom indicator API, add OnTick

### 0.3 Fix TimeseriesTable bug
- [x] `ApplyIndicators` passes empty column `""` — should iterate columns like `ApplyIndicator`

### 0.4 Smoke test
- [x] Add minimal integration test: table → SMA → runner → non-zero final value

**Acceptance criteria:** entire backgommon repo builds and tests pass.

---

## Phase 1: Complete SMA Crossover Reference Strategy

**Goal:** Validate backbone end-to-end with a trustworthy reference implementation.

### 1.1 PRD gaps
- [x] Add `SMAStrategyConfig` struct with `CustomField`
- [x] Implement true crossover detection (prev bar vs current bar), not level check
- [x] Long-only, flat otherwise

### 1.2 Real data loading
- [x] Add data loader helper (JSON → `TimeseriesTable[core.Candle]`) in `pkg/data`
- [x] Replace synthetic `createSampleData()` in SMA example

### 1.3 Indicator application pattern
- [x] Decide and document: table-level pre-calc vs strategy-level indicators
- [x] Wire `runner.WithIndicators` (table.ApplyIndicators at Start)

### 1.4 Results & reporting
- [x] Populate `types.Results` at end of `runner.Start()`
- [x] Compute: final capital, total trades, win rate, max drawdown, returns
- [x] SMA example prints full summary

### 1.5 Documentation
- [x] Update task list / mark completed phases
- [x] Document findings in `sma-crossover-example-strategy-findings.md`

**Acceptance criteria:** SMA crossover runs on real-ish data with meaningful results output.

---

## Phase 2: Harden Core Framework

**Goal:** Make the engine production-usable, not just demonstrable.

### 2a. Portfolio & execution realism
- [x] Brokerage: `FixedBrokerageFee`, `PercentBrokerageRate`
- [x] Slippage: `SlippageMode`, `PercentSlippageRate`, `FixedSlippageAmount`
- [x] Order fill assumptions: `OrderFillAssumption` (close vs next open)
- [x] Cash math on entry/exit with leverage
- [x] Basic tax on realized P&L (STCG/LTCG)
- [x] SIP contributions on schedule
- [x] Idle cash interest + leverage cost accrual

**Priority:** brokerage → slippage → fill price → taxes → SIP/interest

### 2b. Risk manager completion
- [x] Enforce `MaxPortfolioDrawdownRate` + `MaxDrawdownMode` in runner
- [x] Position sizing from `RiskPerTradeRate` + stop-loss distance
- [x] Consistent exit order creation via `createExitOrder`
- [x] `Quantity: 0` = full exit convention in portfolio

### 2c. Runner improvements
- [x] Call `OnDayStart` / `OnDayEnd` lifecycle hooks
- [x] Fix `preCalculateIndicators` — store per-index value, not full slice
- [x] Merge or clarify `Run()` vs `Start()`
- [x] Expose equity curve and closed positions after backtest

### 2d. Indicator system polish
- [x] Add `indicators.LastValue(values []any) (float64, ok bool)` helper
- [x] Document dependency graph behavior
- [x] Consolidate custom indicator API

**Acceptance criteria:** multi-month backtest with realistic costs and trustworthy analytics.

---

## Phase 3: Test Coverage

**Goal:** Refactors don't silently break backtests.

- [ ] `indicators` — SMA warm-up, MACD dependency chain
- [ ] `portfolio` — entry/exit, cash math, full exit, shorts
- [ ] `risk` — stop/take-profit/trailing, position limits
- [ ] `runner` — tick loop, equity curve, order rejection
- [ ] `types` — TimeseriesTable ordering, indicator application

---

## Phase 4: GP-Trade Integration

**Goal:** Real NSE strategies run on real data through backgommon.

### 4.1 Data bridge
- [ ] Loader: `gp-trade/db/nse-data-1/*.json` → `TimeseriesTable[core.Candle]`
- [ ] Formalize Fyers historical fetch as reusable loader

### 4.2 Migrate strategy1
- [ ] Replace `backgommon/src/*` imports with `pkg/*`
- [ ] Rewrite `Tick(...)` → `OnTick(data map[string]core.Candle) []portfolio.Order`
- [ ] Port preprocessing to table-level indicator API

### 4.3 Fix momentum_strategy
- [ ] `types.Candle` → `core.Candle`
- [ ] Update settings to new field names
- [ ] Real `calculatePriceIncrease` via table lookback
- [ ] Fix quantity / position sizing

### 4.4 Wire gp-trade main.go
- [ ] Replace commented old backgommon bootstrap with runner-based flow
- [ ] One path: load data → run strategy → print/plot results

**Acceptance criteria:** at least one real gp-trade strategy runs end-to-end on NSE JSON data.

---

## Phase 5: Documentation & Open-Source Readiness

- [ ] Remove API credentials from README
- [ ] Update all docs to match renamed Settings fields
- [ ] README features list (replace TODOs)
- [ ] `examples/README.md`
- [ ] Migration guide: old `src/` API → new `pkg/` API
- [ ] CHANGELOG for breaking API rename

---

## Phase 6: Future Roadmap (defer until Phases 0–5 done)

- [ ] More indicators (RSI, Bollinger, ATR)
- [ ] Monte Carlo simulation
- [ ] Performance chart generation
- [ ] Live trading web server
- [ ] Multi-strategy portfolio allocation
- [ ] Account lock, profit pocketing, management fees

---

## Recommended Execution Order

```
Phase 0 → Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5 → Phase 6
```

Do not start gp-trade migration until Phase 0–1 are green.

## Key Files

- `pkg/risk/exit_conditions.go` — broken, fix first
- `pkg/risk/manager.go` — new Settings schema
- `pkg/portfolio/portfolio.go` — execution TODOs
- `pkg/runner/runner.go` — results, lifecycle hooks
- `pkg/types/timeseries_table.go` — ApplyIndicators bug
- `examples/strategies/sma_crossover/main.go` — reference strategy
- `gp-trade/strategies/strategy1/` — old API, needs migration
- `gp-trade/strategies/momentum_strategy/` — partial new API
