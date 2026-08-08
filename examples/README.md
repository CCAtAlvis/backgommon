# Examples

Runnable examples for Backgommon. **Strategies live here**, not in `pkg/`.

Start with the [documentation hub](../docs/README.md) if you are new.

---

## Strategies

| Example | Command | Docs |
|---------|---------|------|
| **SMA crossover** | `go run ./examples/strategies/sma_crossover/` | [README](./strategies/sma_crossover/README.md) |

Flags for SMA crossover:

```bash
go run ./examples/strategies/sma_crossover/ -data prices.json
go run ./examples/strategies/sma_crossover/ -data prices.csv -symbol MY-SYM
go run ./examples/strategies/sma_crossover/ -short 20 -long 50 -qty 10
```

---

## Indicators

| Example | Command |
|---------|---------|
| Indicator math & table | `go run ./examples/indicator_example/` |
| Custom indicator in strategy | See `examples/indicator_strategy/strategy.go` (library-style package) |

---

## Tests

SMA crossover includes tests in the same folder:

```bash
go test ./examples/strategies/sma_crossover/...
```

- Unit tests for crossover logic  
- Integration test for runner + strategy end-to-end  

Framework tests live under `pkg/...` (e.g. `pkg/runner`, `pkg/execution`).
