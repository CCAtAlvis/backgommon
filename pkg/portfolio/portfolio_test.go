package portfolio

import (
	"math"
	"testing"
	"time"
)

func TestEntryExitCashWithFees(t *testing.T) {
	s := &Settings{
		InitialCapital:       10000,
		FixedBrokerageFee:    10,
		PercentBrokerageRate: 0,
		DefaultLeverage:      1,
	}
	p := New(s)

	entry := NewOrder("AAPL", Long, Entry, 10, 1)
	entry.Price = 100
	if err := p.ProcessOrder(entry); err != nil {
		t.Fatalf("entry: %v", err)
	}

	// 10*100 margin + 10 brokerage
	wantCash := 10000.0 - 1010.0
	if p.Cash() != wantCash {
		t.Fatalf("after entry cash: got %.2f want %.2f", p.Cash(), wantCash)
	}

	exit := NewOrder("AAPL", Long, Exit, 10, 1)
	exit.Price = 110
	if err := p.ProcessOrder(exit); err != nil {
		t.Fatalf("exit: %v", err)
	}

	// margin 1000 + pnl 100 - brokerage 10
	wantCash = wantCash + 1000.0 + 100.0 - 10.0
	if p.Cash() != wantCash {
		t.Fatalf("after exit cash: got %.2f want %.2f", p.Cash(), wantCash)
	}
}

func TestExitQuantityZeroMeansFullClose(t *testing.T) {
	s := &Settings{InitialCapital: 5000, DefaultLeverage: 1}
	p := New(s)

	entry := NewOrder("X", Long, Entry, 5, 1)
	entry.Price = 50
	if err := p.ProcessOrder(entry); err != nil {
		t.Fatalf("entry: %v", err)
	}

	exit := NewOrder("X", Long, Exit, 0, 1)
	exit.Price = 55
	if err := p.ProcessOrder(exit); err != nil {
		t.Fatalf("exit: %v", err)
	}
	if len(p.Positions()) != 0 {
		t.Fatal("expected flat after full exit")
	}
}

func TestProcessSIP(t *testing.T) {
	s := &Settings{
		InitialCapital: 1000,
		SIPAmount:      500,
		SIPFrequency:   24 * time.Hour,
	}
	p := New(s)
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	p.ProcessPeriodicEvents(t0)
	if p.Cash() != 1500 {
		t.Fatalf("SIP cash: got %.2f", p.Cash())
	}
}

func TestEstimateEntryCashMatchesProcessOrder(t *testing.T) {
	s := &Settings{
		InitialCapital:       100_000,
		DefaultLeverage:      1,
		FixedBrokerageFee:    20,
		PercentBrokerageRate: 0.0003,
		EnableTaxes:          true,
		BuyTaxRate:           0.002,
	}
	p := New(s)
	ord := NewOrder("AAPL", Long, Entry, 10, 1)
	ord.Price = 100
	est := p.EstimateEntryCash(ord)
	// margin 1000 + brokerage 20+0.3 + tax 2 = 1022.3
	want := 1000.0 + 20.0 + 0.3 + 2.0
	if math.Abs(est-want) > 1e-9 {
		t.Fatalf("EstimateEntryCash = %f, want %f", est, want)
	}
	before := p.Cash()
	if err := p.ProcessOrder(ord); err != nil {
		t.Fatalf("ProcessOrder: %v", err)
	}
	gotDelta := before - p.Cash()
	if math.Abs(gotDelta-est) > 1e-9 {
		t.Fatalf("cash delta %.6f != EstimateEntryCash %.6f", gotDelta, est)
	}
}

func TestShortEntryCashValidation(t *testing.T) {
	s := &Settings{
		InitialCapital:            500,
		DefaultLeverage:           1,
		EnableShorts:              true,
		AllowNegativeCashFromFees: false,
		FixedBrokerageFee:         10,
		EnableTaxes:               true,
		SellTaxRate:               0.002,
	}
	p := New(s)

	// Notional 1000 alone exceeds cash — must reject
	tooBig := NewOrder("XYZ", Short, Entry, 10, 1)
	tooBig.Price = 100
	if err := p.ProcessOrder(tooBig); err == nil {
		t.Fatal("expected insufficient cash for oversized short entry")
	}

	// Fits margin but fees push over when AllowNegativeCashFromFees=false
	p2 := New(&Settings{
		InitialCapital:            1005,
		DefaultLeverage:           1,
		EnableShorts:              true,
		AllowNegativeCashFromFees: false,
		FixedBrokerageFee:         20,
	})
	ord := NewOrder("XYZ", Short, Entry, 10, 1)
	ord.Price = 100 // margin 1000; +20 fee = 1020 > 1005
	if err := p2.ProcessOrder(ord); err == nil {
		t.Fatal("expected reject when margin+fees exceed cash")
	}

	// AllowNegativeCashFromFees: margin-only check passes
	p3 := New(&Settings{
		InitialCapital:            1005,
		DefaultLeverage:           1,
		EnableShorts:              true,
		AllowNegativeCashFromFees: true,
		FixedBrokerageFee:         20,
	})
	ord3 := NewOrder("XYZ", Short, Entry, 10, 1)
	ord3.Price = 100
	if err := p3.ProcessOrder(ord3); err != nil {
		t.Fatalf("short entry should succeed with AllowNegativeCashFromFees: %v", err)
	}
	if p3.Cash() >= 0 {
		t.Fatalf("expected fees to push cash negative, got %.2f", p3.Cash())
	}
}
