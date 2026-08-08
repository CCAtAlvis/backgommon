# Reporting and Output

After a backtest completes, you typically want to inspect results: metrics, equity curve, charts. Backgommon provides a built-in `pkg/output` package that handles this.

For compressed sweep storage, shared viewer, and `backgommon serve`, see **[reporting-storage.md](./reporting-storage.md)**.

---

## Automatic export via Runner

The simplest approach: pass `runner.WithOutputDir("./output")` when creating the runner. After `Start()` finishes, artifacts are written automatically:

```go
r := runner.New(myStrategy,
    runner.WithPortfolio(p),
    runner.WithRiskManager(rm),
    runner.WithData(table),
    runner.WithOutputDir("./output"),  // <-- enables automatic export
)
r.Start()
// ./output/run_<timestamp>/results.json.zst, viewer.html, data.js, …
```

| File | Contents |
|------|----------|
| `results.json.zst` | Compact JSON report (metrics, equity curve, trades, orders), zstd-compressed |
| `viewer.html` | Vanilla JS SPA (Plotly); loads `data.js` or fetches `results.json` |
| `data.js` | Offline `window.__REPORT__` bootstrap for `file://` open |
| `config.json` | Optional settings from `WithReportSettings` |

---

## Manual export

If you prefer more control (custom paths, exporting only the equity curve, etc.), use the `output` package directly:

```go
import "github.com/CCAtAlvis/backgommon/pkg/output"

// Export full report (meta + metrics + equity curve)
err := output.ExportResultsJSON(r.Results, r.EquityCurve, "my_report.json")

// Export only the equity curve
err = output.ExportEquityCurveJSON(r.EquityCurve, "equity_curve.json")

// Export interactive HTML report
err = output.ExportHTMLReport(r.Results, r.EquityCurve, "report.html")
```

---

## JSON schema

The `results.json` file has this structure:

```json
{
  "meta": {
    "start_time": "1999-01-04",
    "end_time": "2022-12-30",
    "initial_capital": 50000.0,
    "final_capital": 234567.89
  },
  "metrics": {
    "returns": 3.69,
    "max_drawdown": 0.42,
    "sharpe_ratio": 0.0,
    "sortino_ratio": 0.0,
    "total_trades": 312,
    "winning_trades": 187,
    "losing_trades": 125,
    "win_rate": 0.5993,
    "custom": {}
  },
  "equity_curve": [
    {
      "time": "1999-01-04",
      "value": 50000.0,
      "cash": 50000.0,
      "open_positions": 0,
      "unrealized_pnl": 0.0
    }
  ]
}
```

---

## HTML report

The `report.html` file is a single self-contained page that loads Chart.js from a CDN. It contains:

1. **Metrics cards** — Initial/final capital, return %, max drawdown, total trades, win rate
2. **Equity curve chart** — Value over time
3. **Log equity curve** — Log10(value) over time, useful for long-running backtests with exponential growth
4. **Drawdown chart** — Percentage drawdown from peak, filled area chart

Open the file in any browser. No server needed.

---

## Extending

To add custom metrics to the JSON report, set them on `types.Results.Metrics` before or after the run:

```go
r.Results.Metrics["calmar_ratio"] = calmarRatio
r.Results.Metrics["annual_return"] = annualReturn
```

These appear in `metrics.custom` in the JSON output.

---

## Next steps

- [Getting started](./getting-started.md) — Run your first backtest
- [Writing a strategy](./writing-a-strategy.md) — Strategy lifecycle
- [Extending the framework](./extending-the-framework.md) — Override framework behavior
