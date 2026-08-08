# Reporting Storage & Compression

How backgommon stores backtest / sweep artifacts, and what comes next.

## Current layout (Phase 1)

### Single run (`runner.WithOutputDir`)

```
output/run_<timestamp>/
  results.json.zst   # canonical compact JSON, zstd-compressed
  config.json        # strategy/settings (uncompressed)
  viewer.html        # vanilla JS SPA (loads data.js or fetches results.json)
  data.js            # window.__REPORT__ = … (file:// offline open)
  report.html        # thin redirect → viewer.html
```

### Sweep (`sweep.Run`)

```
output/sweep_<timestamp>/
  viewer.html          # one shared SPA for all trials
  dashboard.html
  summary.json
  summary.csv
  trials/trial_NNN/
    results.json.zst   # only — no per-trial fat HTML
    config.json
```

Open a sweep with:

```bash
go run ./cmd/backgommon serve ./output/sweep_<timestamp>
# then open http://127.0.0.1:8743/dashboard.html
# trial drill-down: viewer.html?trial=000
```

The serve handler maps `…/results.json` → decompress `results.json.zst` on the fly.

### Schema notes

- JSON is **compact** (no indent).
- **`all_orders`** is the order source of truth.
- **`all_trades`** no longer nests `orders[]`; the viewer joins by `trade_id` and recomputes running qty / avg / PnL in JS.
- Per-bar **holdings** on the equity curve are kept (needed for composition insights).

---

## Future work (not implemented)

### Phase 2 — Retention (top-N × metrics)

Keep full `results.json.zst` only for the union of top-N trials per user-chosen metrics (e.g. Sharpe, Sortino, CAGR, returns). Everyone else stays metrics-only (or lite). Streaming min-heaps during the sweep cap peak disk.

```go
type Retention struct {
    TopN    int
    Metrics []string // e.g. sharpe_ratio, sortino_ratio, cagr, returns
}
```

### Sparse holdings (B3)

Today every bar stores the full holdings array even when positions are unchanged. Sparse export would write holdings only on bars where `(instrument, qty)` changes; the viewer forward-fills for tooltips. Same insights, much less disk.

### Downsample / export profiles (B5)

Profiles such as `none | metrics | lite | full`:

- **lite** — downsampled equity (e.g. every Nth bar), no trade timelines
- **full** — current detail

Useful as the default for non-keeper trials once retention lands.

### Insights platform (on top of `serve`)

- `backgommon serve` with no directory argument starts the **catalog daemon** (SQLite under `--data-dir`, default `~/.backgommon`):

```bash
backgommon serve --addr 127.0.0.1:8743   # web UI only — optional for CLI
backgommon register ./bin/sma_crossover  # works offline against --data-dir
backgommon run sma_crossover --config run.yml
backgommon sweep sma_crossover
backgommon ingest ./output/sweep_<timestamp>
backgommon status
backgommon open                          # needs serve running
```

CLI commands read/write the local catalog directly. `serve` is for the browser UI (and its HTTP API). Strategy workers speak JSON (`describe` / `run` / `sweep`); human configs use YAML. See `pkg/worker` and the SMA example (`examples/strategies/sma_crossover`).

Longer term:

- Retention / top-N compression
- Cross-strategy comparison and insight pages
- Live / paper monitoring dashboards

### Post-hoc tools

- Recompress / prune existing legacy sweeps (indented JSON + fat HTML → zstd + shared viewer) without re-running backtests.
