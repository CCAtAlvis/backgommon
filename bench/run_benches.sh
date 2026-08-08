#!/usr/bin/env bash
# Run framework performance campaign benches and regenerate RESULTS.md
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "==> Impact report (writes bench/RESULTS.generated.md)"
go test ./bench -run TestImpactReport -count=1 -timeout 60m -v

echo "==> Go benchmarks (subset; full suite is long)"
go test ./bench -bench='BenchmarkE0_|BenchmarkE1_|BenchmarkE2_|BenchmarkE3_SoA_SumRow$|BenchmarkE4_|BenchmarkE5_|BenchmarkE6_|BenchmarkE7_|BenchmarkE9a_|BenchmarkE9c_|BenchmarkE10_|BenchmarkE11_' \
  -benchmem -count=1 -timeout 60m

echo "Done. Curated report: bench/RESULTS.md  |  auto table: bench/RESULTS.generated.md"
