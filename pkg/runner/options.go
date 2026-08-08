package runner

import (
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/execution"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
	"github.com/CCAtAlvis/backgommon/pkg/output"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
	"github.com/CCAtAlvis/backgommon/pkg/risk"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

// =============================================================================
// Core Components
// =============================================================================

// WithPortfolio sets the portfolio manager.
func WithPortfolio(p interfaces.PortfolioManager) Option {
	return func(r *Runner) {
		r.Portfolio = p
		if r.fillPricer == nil {
			if cp, ok := p.(*portfolio.Portfolio); ok {
				mode := execution.FillModeFromString(cp.Settings().Execution.OrderFillAssumption)
				r.fillPricer = execution.NewStandardFillPricer(mode)
			}
		}
	}
}

// WithRiskManager sets the risk manager.
func WithRiskManager(rm interfaces.RiskManager) Option {
	return func(r *Runner) {
		r.RiskManager = rm
	}
}

// WithData sets the OHLCV data table for backtesting.
func WithData(data *types.TimeseriesTable[core.Candle]) Option {
	return func(r *Runner) {
		r.Data = data
	}
}

// WithResults provides a pre-allocated results container.
func WithResults(results *types.Results) Option {
	return func(r *Runner) {
		r.Results = results
	}
}

// =============================================================================
// Indicators
// =============================================================================

// WithIndicators pre-computes indicators on the data table at the start of Start().
// Prefer this or TimeseriesTable.ApplyIndicators before Start — not both unless you
// intend to overwrite.
func WithIndicators(indicators []interfaces.Indicator) Option {
	return func(r *Runner) {
		r.preRunIndicators = indicators
	}
}

// =============================================================================
// Order Execution & Fill Pricing
// =============================================================================

// WithFillPricer sets a custom fill pricer that resolves execution prices from bar data.
// Use this for full control over fill logic (e.g., VWAP, custom auction model).
func WithFillPricer(fp interfaces.FillPricer) Option {
	return func(r *Runner) {
		r.fillPricer = fp
	}
}

// WithFillMode sets a built-in fill assumption by name.
// Available modes: FillCurrentClose, FillCurrentOpen, FillNextBarOpen, FillWorstCase, etc.
// See pkg/execution for the full list.
func WithFillMode(mode execution.FillMode) Option {
	return func(r *Runner) {
		r.fillPricer = execution.NewStandardFillPricer(mode)
	}
}

// WithSlippageModel sets how adverse slippage is applied after fill pricing.
// Use execution.FuncSlippageModel{AdjustFn: ...} for a quick function-based override,
// or implement the execution.SlippageModel interface for full control.
func WithSlippageModel(m execution.SlippageModel) Option {
	return func(r *Runner) {
		r.slippage = m
	}
}

// =============================================================================
// Portfolio Customization (pass-through to portfolio.Portfolio)
// =============================================================================

// WithCostCalculator sets a custom brokerage/tax calculator on the portfolio.
// Use portfolio.FuncCostCalculator{BrokerageFn: ..., TransactionTaxFn: ..., CapitalGainsTaxFn: ...}
// to override individual cost methods while keeping defaults for the rest.
//
// Requires WithPortfolio to be called with a *portfolio.Portfolio (the default).
func WithCostCalculator(c portfolio.CostCalculator) Option {
	return func(r *Runner) {
		if p, ok := r.Portfolio.(*portfolio.Portfolio); ok {
			portfolio.WithCostCalculator(c)(p)
		}
	}
}

// WithCashFlowCalculator sets a custom cash flow model that controls how cash moves
// on entry, exit, and for equity contribution calculations.
// Use portfolio.FuncCashFlowCalculator{EntryCashOutflowFn: ..., ExitCashInflowFn: ...}
// for partial overrides.
//
// Requires WithPortfolio to be called with a *portfolio.Portfolio (the default).
func WithCashFlowCalculator(c portfolio.CashFlowCalculator) Option {
	return func(r *Runner) {
		if p, ok := r.Portfolio.(*portfolio.Portfolio); ok {
			portfolio.WithCashFlowCalculator(c)(p)
		}
	}
}

// WithPeriodicEventsModel sets a custom model for scheduled cash flows
// (SIP contributions, idle cash interest, leverage costs, management fees).
// Use portfolio.FuncPeriodicEventsModel{ApplyFn: ...} for a function-based override.
//
// Requires WithPortfolio to be called with a *portfolio.Portfolio (the default).
func WithPeriodicEventsModel(m portfolio.PeriodicEventsModel) Option {
	return func(r *Runner) {
		if p, ok := r.Portfolio.(*portfolio.Portfolio); ok {
			portfolio.WithPeriodicEventsModel(m)(p)
		}
	}
}

// =============================================================================
// Risk Management Customization (pass-through to risk.Manager)
// =============================================================================

// WithPositionSizer sets a custom position sizing model on the risk manager.
// Use risk.FuncPositionSizer{SuggestFn: ...} for a function-based override.
//
// Requires WithRiskManager to be called with a *risk.Manager (the default).
func WithPositionSizer(s risk.PositionSizer) Option {
	return func(r *Runner) {
		if rm, ok := r.RiskManager.(*risk.Manager); ok {
			risk.WithPositionSizer(s)(rm)
		}
	}
}

// WithDrawdownPolicy sets a custom drawdown evaluation policy on the risk manager.
// Use risk.FuncDrawdownPolicy{EvaluateFn: ...} for a function-based override.
//
// Requires WithRiskManager to be called with a *risk.Manager (the default).
func WithDrawdownPolicy(p risk.DrawdownPolicy) Option {
	return func(r *Runner) {
		if rm, ok := r.RiskManager.(*risk.Manager); ok {
			risk.WithDrawdownPolicy(p)(rm)
		}
	}
}

// =============================================================================
// Time Range
// =============================================================================

// WithStartTime sets the earliest date for backtest processing.
// Rows before this date are skipped (no orders, no equity curve snapshots),
// but data remains available for indicator lookbacks and strategy queries.
func WithStartTime(t time.Time) Option {
	return func(r *Runner) {
		r.startTime = t
	}
}

// WithEndTime sets the latest date for backtest processing.
// Rows after this date are skipped. Useful for testing strategies over specific date ranges.
func WithEndTime(t time.Time) Option {
	return func(r *Runner) {
		r.endTime = t
	}
}

// WithStrictOrderProcessing controls whether a rejected order aborts the backtest.
// Default is false (lenient): failed risk checks / ProcessOrder errors are logged,
// the order is skipped, and the run continues. Set true for fail-fast / CI runs.
func WithStrictOrderProcessing(strict bool) Option {
	return func(r *Runner) {
		r.strictOrderProcessing = strict
	}
}

// =============================================================================
// Output & Reporting
// =============================================================================

// WithOutputDir sets the directory for automatic report export after the backtest completes.
// When set, the runner writes results.json.zst (and optional offline HTML) into a timestamped subfolder.
func WithOutputDir(dir string) Option {
	return func(r *Runner) {
		r.outputDir = dir
		r.fixedOutputDir = false
	}
}

// WithReportDir writes report artifacts directly into dir (no run_<timestamp> subfolder).
// Use this for parameter sweeps where each trial already has a unique output folder.
func WithReportDir(dir string) Option {
	return func(r *Runner) {
		r.outputDir = dir
		r.fixedOutputDir = true
	}
}

// WithReportSettings attaches a JSON-serializable settings struct to be exported
// as config.json alongside the report, and rendered in the settings section of the HTML.
func WithReportSettings(settings interface{}) Option {
	return func(r *Runner) {
		r.reportSettings = settings
	}
}

// WithExportOptions overrides default report compression / offline HTML behaviour.
func WithExportOptions(opts output.ExportOptions) Option {
	return func(r *Runner) {
		r.exportOpts = opts
	}
}

// WithOfflineDataJS enables writing data.js + thin report.html for file:// viewing.
func WithOfflineDataJS(enabled bool) Option {
	return func(r *Runner) {
		r.exportOpts.OfflineDataJS = enabled
		if enabled {
			r.exportOpts.WriteViewer = true
		}
		if r.exportOpts.Compress == "" {
			r.exportOpts.Compress = output.CompressZstd
		}
	}
}
