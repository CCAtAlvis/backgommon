package runner

import (
	"fmt"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/execution"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

func defaultFillPricer() interfaces.FillPricer {
	return execution.NewStandardFillPricer(execution.FillCurrentClose)
}

// applyFillPrices resolves execution prices for orders that lack an explicit
// price (Price <= 0). It delegates to the configured FillPricer — which
// selects a price from the current or next bar — and then applies the
// slippage model to simulate real-world adverse fill conditions.
// Orders with Price > 0 are left unchanged (explicit limit/stop from the strategy).
func (r *Runner) applyFillPrices(orders []portfolio.Order, current, next map[string]core.Candle) error {
	if r.fillPricer == nil {
		r.fillPricer = defaultFillPricer()
	}

	for i := range orders {
		if orders[i].Price > 0 {
			continue
		}

		cur, ok := current[orders[i].Instrument]
		if !ok {
			return fmt.Errorf("fill price: no bar for %s", orders[i].Instrument)
		}

		var nextBar *core.Candle
		if next != nil {
			if nb, ok := next[orders[i].Instrument]; ok {
				nextBar = &nb
			}
		}

		price, err := r.fillPricer.FillPrice(interfaces.FillContext{
			Order:      orders[i],
			CurrentBar: cur,
			NextBar:    nextBar,
			AllBars:    current,
			NextBars:   next,
		})
		if err != nil {
			return fmt.Errorf("fill price for %s: %w", orders[i].Instrument, err)
		}
		price = r.slippageModel().AdjustPrice(execution.SlippageContext{
			BasePrice: price,
			Order:     orders[i],
			Execution: r.executionSettings(),
		})
		orders[i].Price = price
	}

	return nil
}

func (r *Runner) slippageModel() execution.SlippageModel {
	if r.slippage != nil {
		return r.slippage
	}
	return execution.StandardSlippageModel{}
}

func (r *Runner) executionSettings() portfolio.ExecutionSettings {
	if p, ok := r.Portfolio.(*portfolio.Portfolio); ok {
		return p.Settings().Execution
	}
	return *portfolio.NewDefaultExecutionSettings()
}
