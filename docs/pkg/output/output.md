# pkg/output

The `output` package provides reporting functionality for backtest results. Canonical artifacts are compact **`results.json.zst`** plus a shared vanilla **`viewer.html`** SPA. See [reporting-storage.md](../../reporting-storage.md) for layout, `backgommon serve`, and future compression work.

## Public API

### Types

```go
type Report struct {
    Meta        ReportMeta         `json:"meta"`
    Metrics     ReportMetrics      `json:"metrics"`
    EquityCurve []EquityCurvePoint `json:"equity_curve"`
}

type ReportMeta struct {
    StartTime      string  `json:"start_time"`
    EndTime        string  `json:"end_time"`
    InitialCapital float64 `json:"initial_capital"`
    FinalCapital   float64 `json:"final_capital"`
}

type ReportMetrics struct {
    Returns       float64            `json:"returns"`
    MaxDrawdown   float64            `json:"max_drawdown"`
    SharpeRatio   float64            `json:"sharpe_ratio"`
    SortinoRatio  float64            `json:"sortino_ratio"`
    TotalTrades   int                `json:"total_trades"`
    WinningTrades int                `json:"winning_trades"`
    LosingTrades  int                `json:"losing_trades"`
    WinRate       float64            `json:"win_rate"`
    Custom        map[string]float64 `json:"custom,omitempty"`
}

type EquityCurvePoint struct {
    Time          string  `json:"time"`
    Value         float64 `json:"value"`
    Cash          float64 `json:"cash"`
    OpenPositions int     `json:"open_positions"`
    UnrealizedPnL float64 `json:"unrealized_pnl"`
}
```

### Functions

| Function | Description |
|----------|-------------|
| `BuildReport(results, curve)` | Constructs a `Report` from framework types |
| `ExportResultsJSON(results, curve, path)` | Writes full report as indented JSON |
| `ExportEquityCurveJSON(curve, path)` | Writes only the equity curve array as JSON |
| `ExportHTMLReport(results, curve, path)` | Generates a self-contained HTML report with charts |

## Integration with Runner

The runner calls these functions automatically when `WithOutputDir` is set:

```go
runner.New(strategy,
    runner.WithOutputDir("./output"),
    // ...
)
```

After `Start()` completes, `output/results.json` and `output/report.html` are written.

## HTML Report Details

The generated HTML file:
- Loads Chart.js 4.x from CDN (`cdn.jsdelivr.net`)
- Embeds all data as inline JSON (no external data files needed)
- Renders three charts: equity curve, log equity curve, drawdown
- Displays metrics in a card grid
- Uses a dark theme optimized for readability
- Requires no backend/server — just open in any browser

## Dependencies

This package has zero external Go dependencies. Chart rendering happens client-side in the browser via the embedded `<script>` tag.
