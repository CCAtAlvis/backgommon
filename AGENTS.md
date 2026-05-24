# AGENTS.md

## Cursor Cloud specific instructions

### Overview
Backgommon is a Go library/framework for backtesting trading strategies on multi-asset portfolios. It is **not** a runnable service — it's a library that users embed in their own Go programs. There are no databases, Docker containers, or external services required.

### Codebase structure
- `pkg/` — New, refactored modular public API (active development target)
- `src/` — Legacy codebase being migrated from a private repo
- `examples/` — Example programs demonstrating library usage

### Build & test commands
```bash
go mod tidy            # fetch/sync dependencies
go build ./...         # build all (see known issues below)
go test ./...          # run all tests
go vet ./...           # lint
```

### Known build issues
Two packages have pre-existing compile errors and cannot be built:
- `pkg/risk` — `exit_conditions.go` references stale field names on `Settings` (e.g. `UseStopLoss` vs `EnableStopLoss`, `DefaultStopLoss` vs `DefaultStopLossRate`). The `manager.go` struct was refactored but `exit_conditions.go` was not updated to match.
- `src/runner` — `runner.go` references `structs.Candle` which does not exist in the `src/structs` package.

All other packages (`pkg/indicators`, `pkg/types`, `pkg/core`, `pkg/portfolio`, `pkg/interfaces`, `pkg/strategy`, `pkg/runner`, `src/types`, `src/structs`) compile and vet cleanly.

### Tests
Only `pkg/indicators/` has test files (`ema_test.go`, `macd_test.go` — 3 test cases total). Run with:
```bash
go test ./pkg/indicators/... -v
```

### Go version
The `go.mod` specifies `go 1.19` but `go mod tidy` auto-upgrades the toolchain to `go1.25.10` due to the `gonum.org/v1/plot` dependency requiring `go >= 1.24.0`. The system Go is `1.22.2`; the Go toolchain download mechanism handles this transparently.

### No secrets or external dependencies required
This is a pure Go computational library with zero runtime dependencies on databases, APIs, or secrets.
