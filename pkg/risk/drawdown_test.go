package risk

import (
	"testing"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

func TestSuggestedQuantity(t *testing.T) {
	m := New(&Settings{
		RiskPerTradeRate:    0.01,
		EnableStopLoss:      true,
		DefaultStopLossRate: 0.05,
	})
	qty := m.SuggestedQuantity(100000, 100)
	if qty != 200 {
		t.Fatalf("qty: got %d want 200", qty)
	}
}

func TestDrawdownBreachedBlocksEntries(t *testing.T) {
	ps := &portfolio.Settings{InitialCapital: 10000}
	p := portfolio.New(ps)
	m := New(&Settings{
		MaxPortfolioDrawdownRate: 0.05,
		MaxDrawdownMode:          StopNewTrades,
	})

	now := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	entry := portfolio.NewOrder("X", portfolio.Long, portfolio.Entry, 100, 1)
	entry.Price = 100
	if err := p.ProcessOrder(entry); err != nil {
		t.Fatalf("entry: %v", err)
	}
	m.checkDrawdown(p, now) // establish peak at ~10000

	p.UpdatePositions(map[string]float64{"X": 80})
	action := m.checkDrawdown(p, now)
	if !action.Breached {
		t.Fatalf("expected breach, drawdown rate %.2f", action.CurrentRate)
	}
	if !action.Breached {
		t.Fatalf("expected breach, drawdown rate %.2f", action.CurrentRate)
	}
	if !action.BlockNewEntries {
		t.Fatal("expected block new entries")
	}

	entry2 := portfolio.NewOrder("X", portfolio.Long, portfolio.Entry, 1, 1)
	entry2.Price = 100
	if err := m.ValidateOrder(p, entry2); err == nil {
		t.Fatal("expected blocked entry")
	}
}

func TestCreateExitOrderUsesPositionSide(t *testing.T) {
	pos := &portfolio.Position{
		Instrument: "ABC",
		Side:       portfolio.Long,
		Quantity:   5,
		Leverage:   2,
	}
	ord := createExitOrder(pos, "stop_loss")
	if ord.Side != portfolio.Long || ord.Type != portfolio.Exit {
		t.Fatalf("unexpected order: side=%v type=%v", ord.Side, ord.Type)
	}
	if ord.Quantity != 5 {
		t.Fatalf("qty: got %d", ord.Quantity)
	}
}
