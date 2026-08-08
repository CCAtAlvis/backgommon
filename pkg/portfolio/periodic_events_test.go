package portfolio_test

import (
	"testing"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

func TestSIPContribution(t *testing.T) {
	settings := &portfolio.Settings{
		InitialCapital: 100000,
		SIPAmount:      5000,
		SIPFrequency:   30 * 24 * time.Hour,
	}
	p := portfolio.New(settings)

	initialCash := p.Cash()
	p.ProcessPeriodicEvents(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))

	if p.Cash() != initialCash+5000 {
		t.Errorf("after first SIP: cash = %f, want %f", p.Cash(), initialCash+5000)
	}

	p.ProcessPeriodicEvents(time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC))
	if p.Cash() != initialCash+5000 {
		t.Errorf("SIP should not fire again within frequency window, cash = %f", p.Cash())
	}

	p.ProcessPeriodicEvents(time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC))
	if p.Cash() != initialCash+10000 {
		t.Errorf("after second SIP: cash = %f, want %f", p.Cash(), initialCash+10000)
	}
}

func TestIdleCashInterest(t *testing.T) {
	settings := &portfolio.Settings{
		InitialCapital:            100000,
		IdleCashInterestAnnualRate: 0.0365,
		IdleCashInterestFrequency: 24 * time.Hour,
	}
	p := portfolio.New(settings)

	p.ProcessPeriodicEvents(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))

	if p.Cash() <= 100000 {
		t.Errorf("cash should increase from interest, got %f", p.Cash())
	}
}

func TestLeverageCost(t *testing.T) {
	settings := &portfolio.Settings{
		InitialCapital:         200000,
		DefaultLeverage:        2.0,
		LeverageCostAnnualRate: 0.10,
		LeverageCostFrequency:  24 * time.Hour,
	}
	p := portfolio.New(settings)

	entry := portfolio.NewOrder("AAPL", portfolio.Long, portfolio.Entry, 100, 2.0)
	entry.Price = 100.0
	entry.FilledAt = time.Date(2025, 1, 1, 9, 0, 0, 0, time.UTC)
	if err := p.ProcessOrder(entry); err != nil {
		t.Fatalf("ProcessOrder: %v", err)
	}

	cashBefore := p.Cash()
	p.ProcessPeriodicEvents(time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC))

	if p.Cash() >= cashBefore {
		t.Errorf("cash should decrease from leverage cost, before=%f after=%f", cashBefore, p.Cash())
	}
}

func TestManagementFee(t *testing.T) {
	settings := &portfolio.Settings{
		InitialCapital:         100000,
		EnableManagementFee:    true,
		ManagementFeeAnnualRate: 0.02,
		ManagementFeeFrequency: 30 * 24 * time.Hour,
	}
	p := portfolio.New(settings)

	cashBefore := p.Cash()
	p.ProcessPeriodicEvents(time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC))

	if p.Cash() >= cashBefore {
		t.Errorf("cash should decrease from management fee, before=%f after=%f", cashBefore, p.Cash())
	}
}

func TestCustomPeriodicModel(t *testing.T) {
	called := false
	custom := &portfolio.FuncPeriodicEventsModel{
		ApplyFn: func(ctx portfolio.PeriodicContext) portfolio.PeriodicResult {
			called = true
			return portfolio.PeriodicResult{
				CashDelta: 999,
				State:     ctx.State,
			}
		},
	}

	settings := &portfolio.Settings{
		InitialCapital: 50000,
		SIPAmount:      1000,
		SIPFrequency:   24 * time.Hour,
	}
	p := portfolio.New(settings, portfolio.WithPeriodicEventsModel(custom))

	p.ProcessPeriodicEvents(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))

	if !called {
		t.Error("custom ApplyFn should have been called")
	}
	if p.Cash() != 50999 {
		t.Errorf("cash = %f, want 50999", p.Cash())
	}
}
