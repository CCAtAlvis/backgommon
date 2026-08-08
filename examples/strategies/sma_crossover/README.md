# SMA Crossover Example

Long-only SMA crossover reference strategy using the Backgommon runner.

Part of the [Backgommon documentation journey](../../../docs/README.md).

## Run

```bash
# Synthetic data (default, 300 bars)
go run ./examples/strategies/sma_crossover/

# JSON OHLCV file (symbol embedded in file)
go run ./examples/strategies/sma_crossover/ -data /path/to/symbol.json

# CSV OHLCV (symbol passed via flag)
go run ./examples/strategies/sma_crossover/ -data /path/to/prices.csv -symbol MY-SYM

# Custom windows
go run ./examples/strategies/sma_crossover/ -short 20 -long 50 -qty 5

# Parameter sweep (Clone data mode + parallel workers + per-trial HTML reports)
go run ./examples/strategies/sma_crossover/ -sweep -output ./output
```

## Strategy code

Implementation: [`sma_crossover.go`](./sma_crossover.go) (`SMACrossoverStrategy`, `SMAStrategyConfig`).  
Wiring: [`main.go`](./main.go).  
Sweep demo: [`sweep.go`](./sweep.go) (uses `pkg/sweep` with Clone mode).

## Indicators

Indicators are pre-computed via `runner.WithIndicators`. See [indicators-application.md](../../../docs/indicators-application.md).
