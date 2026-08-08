package portfolio_test

import (
	"testing"

	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

func TestEntryCashOutflow(t *testing.T) {
	calc := &portfolio.StandardCashFlowCalculator{}
	settings := portfolio.Settings{
		FixedBrokerageFee:    10.0,
		PercentBrokerageRate: 0.001,
		EnableTaxes:          true,
		BuyTaxRate:           0.001,
	}

	ord := portfolio.NewOrder("AAPL", portfolio.Long, portfolio.Entry, 100, 2.0)
	ord.Price = 50.0

	outflow := calc.EntryCashOutflow(portfolio.EntryCashContext{
		Order:    ord,
		Settings: settings,
	})

	notional := 100.0 * 50.0       // 5000
	margin := notional / 2.0        // 2500
	brokerage := 10.0 + 5000*0.001  // 15
	tax := 5000.0 * 0.001           // 5
	want := margin + brokerage + tax // 2520

	if outflow != want {
		t.Errorf("EntryCashOutflow = %f, want %f", outflow, want)
	}
}

func TestExitCashInflow(t *testing.T) {
	calc := &portfolio.StandardCashFlowCalculator{}
	settings := portfolio.Settings{
		FixedBrokerageFee:    10.0,
		PercentBrokerageRate: 0.001,
		EnableTaxes:          true,
		SellTaxRate:          0.001,
		STCapitalGainsTaxRate: 0.15,
		ShortTermHoldingPeriod: 365 * 24 * 3600e9, // 365 days as Duration
	}

	ord := portfolio.NewOrder("AAPL", portfolio.Long, portfolio.Exit, 100, 2.0)
	ord.Price = 60.0

	realizedPnL := 100.0 * (60.0 - 50.0) * 2.0 // 2000

	inflow := calc.ExitCashInflow(portfolio.ExitCashContext{
		Order:            ord,
		OpenPrice:        50.0,
		Leverage:         2.0,
		RealizedPnLSlice: realizedPnL,
		HoldingPeriod:    30 * 24 * 3600e9, // 30 days
		Settings:         settings,
	})

	notional := 100.0 * 60.0                        // 6000
	marginReleased := 100.0 * 50.0 / 2.0            // 2500
	brokerage := 10.0 + notional*0.001               // 16
	txnTax := notional * 0.001                       // 6
	cgTax := realizedPnL * 0.15                      // 300
	want := marginReleased + realizedPnL - brokerage - txnTax - cgTax

	if inflow != want {
		t.Errorf("ExitCashInflow = %f, want %f", inflow, want)
	}
}

func TestCustomCashFlowCalculator(t *testing.T) {
	custom := &portfolio.FuncCashFlowCalculator{
		EntryCashOutflowFn: func(ctx portfolio.EntryCashContext) float64 {
			return 42.0
		},
	}

	ord := portfolio.NewOrder("AAPL", portfolio.Long, portfolio.Entry, 10, 1.0)
	ord.Price = 100.0

	outflow := custom.EntryCashOutflow(portfolio.EntryCashContext{
		Order:    ord,
		Settings: portfolio.Settings{},
	})

	if outflow != 42.0 {
		t.Errorf("custom EntryCashOutflow = %f, want 42", outflow)
	}

	inflow := custom.ExitCashInflow(portfolio.ExitCashContext{
		Order:    portfolio.NewOrder("AAPL", portfolio.Long, portfolio.Exit, 10, 1.0),
		Settings: portfolio.Settings{},
	})
	if inflow == 42.0 {
		t.Error("ExitCashInflow should fall back to standard, not use entry override")
	}
}
