# Task List: Backgommon Framework Migration & Integration

**Date Created:** 2026-05-24
**Last Updated:** 2026-05-26

## 1. Objective

Rebuild Backgommon as an open-source, modular Go backtesting framework and integrate it with GP-Trade for proprietary NSE strategy development. Fix the broken build, complete a reference SMA strategy, harden core packages, and migrate gp-trade off the deleted `src/` API.

## 2. Requirements Summary

- Fix compile errors so `go build ./...` passes
- Complete SMA crossover as a reference end-to-end example
- Harden portfolio, risk, and runner for realistic backtesting
- Add test coverage across core packages
- Migrate gp-trade strategies to the new `pkg/` API
- Prepare for open-source release (docs, credential cleanup, migration guide)

## 3. Relevant Files & PRDs

- `docs/prd/backgommon-framework/backgommon-framework-plan.md` — Execution plan
- `docs/prd/backgommon-framework/backgommon-framework-task-list.md` — This file
- `docs/prd/sma-crossover-example-strategy/sma-crossover-example-strategy-prd.md` — SMA reference strategy PRD
- `pkg/risk/exit_conditions.go` — Exit condition logic (Settings field names aligned)
- `pkg/risk/manager.go` — New Settings schema
- `pkg/portfolio/portfolio.go` — Execution realism TODOs
- `pkg/runner/runner.go` — Core loop; Results not populated
- `pkg/types/timeseries_table.go` — ApplyIndicators fixed
- `examples/strategies/sma_crossover/main.go` — Reference strategy example
- `pkg/data/` — NSE JSON + synthetic loaders
- `pkg/runner/` — Results, WithIndicators, order fill
- `docs/indicators-application.md` — Indicator patterns
- `examples/strategies/sma_crossover/` — Runnable example
- `gp-trade/strategies/strategy1/` — Old API, needs migration
- `gp-trade/strategies/momentum_strategy/` — Partial new API

## 4. Implementation Plan (Tasks)

**Phase 0: Unblock the Build**
- [x] 0.1: Fix `pkg/risk/exit_conditions.go` Settings field names
- [x] 0.2: Fix `examples/strategies/sma_crossover/main.go` settings API
- [x] 0.3: Fix `examples/indicator_example/main.go` for `Calculate() []any`
- [x] 0.4: Fix `examples/indicator_strategy/strategy.go`
- [x] 0.5: Fix `ApplyIndicators` empty-column bug in timeseries_table.go
- [x] 0.6: Add smoke integration test
- [x] 0.1–0.6: Complete

**Phase 1: Complete SMA Crossover Reference Strategy**
- [x] 1.1: Add `SMAStrategyConfig` with `CustomField`
- [x] 1.2: Implement true crossover detection (prev vs current bar)
- [x] 1.3: Add real data loader helper (`pkg/data`)
- [x] 1.4: Document indicator application pattern
- [x] 1.5: Populate `types.Results` in runner; print summary in SMA example
- [x] 1.6: Document findings

**Phase 2: Harden Core Framework**
- [x] 2a: Portfolio execution realism (brokerage, slippage, fill price, taxes, SIP, interest)
- [x] 2b: Risk manager completion (drawdown enforcement, position sizing, full exit)
- [x] 2c: Runner improvements (lifecycle hooks, preCalculateIndicators, Run vs Start)
- [x] 2d: Indicator system polish (LastValue helper, docs, custom indicator API)

**Phase 3: Test Coverage**
- [x] 3.1: indicators tests (helpers_test.go, ema_test.go, macd_test.go)
- [x] 3.2: portfolio tests (portfolio_test.go, hooks_test.go — covers brokerage, fees, custom cost overrides)
- [x] 3.3: risk tests (drawdown_test.go, hooks_test.go — covers position sizing, drawdown blocking, custom overrides)
- [x] 3.4: runner tests (results_test.go, runner_test.go — max drawdown calc + full smoke test with SMA)
- [x] 3.5: types tests (timeseries_table_test.go — ApplyIndicators validation)
- [ ] 3.6: Expand coverage — position lifecycle, periodic events, cashflow, exit conditions, order routing, options

**Phase 4: GP-Trade Integration**
- [x] 4.1: NSE data loader in gp-trade (`strategies/nse_momentum/data.go` — loads from SQLite into TimeseriesTable)
- [x] 4.2: Migrate strategy1 to new pkg API (`strategies/nse_momentum/strategy.go` — 478 lines, full implementation)
- [ ] 4.3: Fix or remove legacy momentum_strategy (still uses deprecated `types.Candle` and old Settings fields)
- [x] 4.4: Wire gp-trade main.go to runner (`strategies/nse_momentum/main.go` — full runner with all options)

**Phase 5: Documentation & Open-Source Readiness**
- [ ] 5.1: Remove API credentials from README (CRITICAL — client ID + secret key still on lines 39-40!)
- [x] 5.2: Update docs for renamed Settings fields (comprehensive docs exist for all packages)
- [~] 5.3: README features list and examples/README.md (examples/README.md done; main README needs cleanup)
- [ ] 5.4: Migration guide and CHANGELOG (neither exists)
- [ ] 5.5: Add LICENSE file (missing — required before public release)
- [ ] 5.6: Add CONTRIBUTING.md

**Phase 6: Future Roadmap**
- [ ] 6.1: More indicators (only SMA, EMA, MACD currently)
- [ ] 6.2: Monte Carlo simulation
- [x] 6.3: Performance chart generation (pkg/output/ — full HTML reports with Plotly.js, equity curves, drawdown markers, trade tables)
- [ ] 6.4: Live trading web server
- [~] 6.5: Advanced portfolio features (periodic.go has management fees/SIP/interest/leverage costs; cashflow.go done; account lock + pocketing NOT done)

## 5. Current Status (as of 2026-05-26)

- **Phase 0:** Complete. `go build ./...` and `go test ./...` pass.
- **Phase 1:** Complete. SMA crossover reference strategy fully working with data loader and docs.
- **Phase 2:** Complete — portfolio costs, slippage, drawdown, lifecycle hooks, extensibility interfaces, docs.
- **Phase 3:** Core test coverage exists for all packages. Gaps remain in position lifecycle, periodic events, exit conditions, and order routing.
- **Phase 4:** GP-Trade integration complete (`strategies/nse_momentum/` — full strategy with SQLite loader, runner wiring, reporting, and `WithStartTime` support). Legacy `momentum_strategy/` still uses deprecated API.
- **Phase 5:** Documentation is comprehensive (9 user guides, 12 package docs, examples README). BLOCKERS: API credentials in README, no LICENSE, no CHANGELOG, no migration guide.
- **Phase 6:** Chart generation done (Plotly.js HTML reports). Periodic portfolio features (fees, SIP, interest) done. Remaining: more indicators, Monte Carlo, live trading, account lock/pocketing.
- **Next Steps:** Remove secrets from README, add LICENSE, expand test coverage, clean up or remove legacy `momentum_strategy/`.

## 6. Notes & Challenges

- Old `src/` layout deleted; gp-trade `strategy1` still imports removed paths.
- Settings API was renamed (`UseStopLoss` → `EnableStopLoss`, etc.); Phase 0 call sites updated.
- Indicator `Calculate()` returns `[]any`; examples updated with last-value helpers.
- README contains API credentials — remove before any public release.
- Phase 1 fixed order Price=0 bug via runner `fillOrderPrices` at bar close.
- Classic 50/200 SMA needs years of data; example defaults to 20/50, use `-short 50 -long 200` with NSE JSON.
- gp-trade migration can start; use `data.LoadTableFromNSEJSON` from backgommon.
