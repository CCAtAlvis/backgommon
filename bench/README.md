# Framework performance benches

Bench-only prototypes and measurements for backgommon hot-path design.
**Not production APIs.**

## Quick start

```bash
# Writes RESULTS.md with impact table + recommendations
go test ./bench -run TestImpactReport -count=1 -timeout 60m -v

# Or use the helper script
./bench/run_benches.sh

# Individual Go benchmarks
go test ./bench -bench=BenchmarkE3 -benchmem
```

## Layout

| File | Contents |
|------|----------|
| `synth.go` | Synthetic TimeseriesTable + SoA closes |
| `proto.go` | MarketView, FeaturePanel, lazy next, equity dials, map pool |
| `e0_*` … `e11_*` | Per-experiment benchmarks |
| `impact_test.go` | Timed comparison → `RESULTS.md` |
| `RESULTS.md` | Impact report (implement / defer / reject) |

## Real-cache check

See `gp-trade/cmd/fwbench` — loads NSE binary cache and runs the same kernels
without strategy logic.
