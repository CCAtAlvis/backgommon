# SMA Crossover — Implementation Findings (Phase 1)

**Date:** 2026-05-24

## Completed

- Strategy moved to `pkg/strategy/sma_crossover.go` with `SMAStrategyConfig` and true crossover logic.
- Example thinned to `examples/strategies/sma_crossover/main.go` with `-data` for NSE JSON.
- `pkg/data` loads gp-trade/Fyers JSON format.
- `runner.WithIndicators`, `finalizeResults`, `PrintResults`, order fill at bar close.
- `Portfolio.Value()` fixed to include cost basis + unrealized PnL.
- Documented indicator patterns in `docs/indicators-application.md`.

## Issues found (defer to later phases)

| Issue | Severity | Phase |
|-------|----------|-------|
| Orders without price were silently using 0 before runner fill fix | Fixed in Phase 1 | — |
| `Runner.Run()` still incomplete | Low | 2c |
| Sharpe/Sortino not computed | Low | 2 |
| Portfolio cash/PnL on entry/exit simplified (no brokerage) | Medium | 2a |
| Synthetic demo data rarely produces crossovers with 50/200 | Doc only | — |
| `PortfolioManager` interface grew; gp-trade momentum needs same methods | Medium | 4 |

## Validation

- Unit: golden cross detection (`pkg/strategy/sma_crossover_test.go`)
- Integration: 7-bar forced crossover (`pkg/runner/sma_integration_test.go`)
- Manual: ABB-EQ NSE JSON, 20/50 windows → 73 trades, ~50% return over sample period (not validated for correctness)
