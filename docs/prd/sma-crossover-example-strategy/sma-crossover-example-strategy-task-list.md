# Task List: SMA Crossover Example Strategy Implementation

**Date Created:** 2024-06-09
**Last Updated:** 2026-05-24

## 1. Objective

Implement and test the backbone package by creating a simple SMA crossover strategy as an example, integrating risk and portfolio management, and providing a runnable example to validate the package and identify any missing or broken functionality.

## 2. Requirements Summary

- [x] Calculate SMA(short) and SMA(long) on input price data
- [x] Buy on golden cross, sell on death cross (true crossover)
- [x] Only long or flat positions
- [x] Integrate risk and portfolio management via runner
- [x] Custom field on config
- [x] Runnable example with real JSON data support

## 3. Relevant Files

- `pkg/strategy/sma_crossover.go` — Strategy implementation
- `examples/strategies/sma_crossover/main.go` — Example entrypoint
- `pkg/data/` — Data loading
- `docs/indicators-application.md` — Indicator patterns
- `sma-crossover-example-strategy-findings.md` — Findings log

## 4. Implementation Plan (Tasks)

**Phase 1–5 from original plan:** Complete (see findings doc).

## 5. Current Status

- **Complete.** Strategy lives in `pkg/strategy`; example runs on synthetic or NSE JSON.

## 6. Notes

- Implementation uses `pkg/strategy/sma_crossover.go`, not a standalone example-only file.
- See `sma-crossover-example-strategy-findings.md` for known limitations.
