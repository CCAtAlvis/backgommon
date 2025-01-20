package runner

import (
	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

// WithPortfolio sets the portfolio manager
func WithPortfolio(p interfaces.PortfolioManager) Option {
	return func(r *Runner) {
		r.Portfolio = p
	}
}

// WithRiskManager sets the risk manager
func WithRiskManager(rm interfaces.RiskManager) Option {
	return func(r *Runner) {
		r.RiskManager = rm
	}
}

// WithData sets the data for backtesting
func WithData(data *types.TimeseriesTable[core.Candle]) Option {
	return func(r *Runner) {
		r.Data = data
	}
}

// WithResults sets the results container
func WithResults(results *types.Results) Option {
	return func(r *Runner) {
		r.Results = results
	}
}
