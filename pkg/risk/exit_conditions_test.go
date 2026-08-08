package risk

import (
	"math"
	"testing"

	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

func approxEqual(a, b, eps float64) bool {
	return math.Abs(a-b) < eps
}

func longPos(price float64, qty int) *portfolio.Position {
	ord := portfolio.Order{
		Instrument: "AAPL", Side: portfolio.Long, Type: portfolio.Entry,
		Quantity: qty, Price: price, Leverage: 1.0,
	}
	pos, _ := portfolio.NewPosition(ord)
	return pos
}

func shortPos(price float64, qty int) *portfolio.Position {
	ord := portfolio.Order{
		Instrument: "AAPL", Side: portfolio.Short, Type: portfolio.Entry,
		Quantity: qty, Price: price, Leverage: 1.0,
	}
	pos, _ := portfolio.NewPosition(ord)
	return pos
}

func TestStopLossLong(t *testing.T) {
	m := New(&Settings{EnableStopLoss: true, DefaultStopLossRate: 0.05})
	pos := longPos(100, 10)

	exit, reason := m.checkExitConditions(pos, 95.0)
	if !exit || reason != "stop_loss" {
		t.Errorf("expected stop_loss exit at 95, got exit=%v reason=%q", exit, reason)
	}

	exit, _ = m.checkExitConditions(pos, 96.0)
	if exit {
		t.Error("should not exit at 96 (above stop)")
	}
}

func TestStopLossShort(t *testing.T) {
	m := New(&Settings{EnableStopLoss: true, DefaultStopLossRate: 0.05})
	pos := shortPos(100, 10)

	exit, reason := m.checkExitConditions(pos, 105.0)
	if !exit || reason != "stop_loss" {
		t.Errorf("expected stop_loss exit at 105, got exit=%v reason=%q", exit, reason)
	}

	exit, _ = m.checkExitConditions(pos, 104.0)
	if exit {
		t.Error("should not exit at 104 (below stop)")
	}
}

func TestTakeProfitLong(t *testing.T) {
	m := New(&Settings{EnableTakeProfit: true, DefaultTakeProfitRate: 0.10})
	pos := longPos(100, 10)

	exit, reason := m.checkExitConditions(pos, 110.01)
	if !exit || reason != "take_profit" {
		t.Errorf("expected take_profit at 110.01, got exit=%v reason=%q", exit, reason)
	}

	exit, _ = m.checkExitConditions(pos, 109.0)
	if exit {
		t.Error("should not exit at 109")
	}
}

func TestTakeProfitShort(t *testing.T) {
	m := New(&Settings{EnableTakeProfit: true, DefaultTakeProfitRate: 0.10})
	pos := shortPos(100, 10)

	exit, reason := m.checkExitConditions(pos, 90.0)
	if !exit || reason != "take_profit" {
		t.Errorf("expected take_profit at 90, got exit=%v reason=%q", exit, reason)
	}

	exit, _ = m.checkExitConditions(pos, 91.0)
	if exit {
		t.Error("should not exit at 91")
	}
}

func TestTrailingStopLong(t *testing.T) {
	m := New(&Settings{EnableTrailingStop: true, DefaultTrailingStopRate: 0.10})
	pos := longPos(100, 10)

	m.checkExitConditions(pos, 120.0)
	if pos.TrailingStopHigh != 120.0 {
		t.Fatalf("TrailingStopHigh = %f, want 120", pos.TrailingStopHigh)
	}

	exit, _ := m.checkExitConditions(pos, 109.0)
	if exit {
		t.Error("should not exit at 109 (stop is at 108)")
	}

	exit, reason := m.checkExitConditions(pos, 108.0)
	if !exit || reason != "trailing_stop" {
		t.Errorf("expected trailing_stop at 108, got exit=%v reason=%q", exit, reason)
	}
}

func TestTrailingStopUpdatesHigh(t *testing.T) {
	m := New(&Settings{EnableTrailingStop: true, DefaultTrailingStopRate: 0.10})
	pos := longPos(100, 10)

	m.checkExitConditions(pos, 110.0)
	if pos.TrailingStopHigh != 110.0 {
		t.Errorf("after 110: TrailingStopHigh = %f, want 110", pos.TrailingStopHigh)
	}

	m.checkExitConditions(pos, 130.0)
	if pos.TrailingStopHigh != 130.0 {
		t.Errorf("after 130: TrailingStopHigh = %f, want 130", pos.TrailingStopHigh)
	}

	m.checkExitConditions(pos, 125.0)
	if pos.TrailingStopHigh != 130.0 {
		t.Errorf("after pullback to 125: TrailingStopHigh = %f, want 130", pos.TrailingStopHigh)
	}
}

func TestNoExitWhenDisabled(t *testing.T) {
	m := New(&Settings{})
	pos := longPos(100, 10)

	prices := []float64{50, 200, 0.01, 10000}
	for _, p := range prices {
		exit, _ := m.checkExitConditions(pos, p)
		if exit {
			t.Errorf("no exit should fire when all flags disabled, triggered at price %f", p)
		}
	}
}

func TestGetPositionRisk(t *testing.T) {
	m := New(&Settings{
		EnableStopLoss:          true,
		DefaultStopLossRate:     0.05,
		EnableTakeProfit:        true,
		DefaultTakeProfitRate:   0.10,
		EnableTrailingStop:      true,
		DefaultTrailingStopRate: 0.08,
	})
	pos := longPos(100, 10)
	pos.TrailingStopHigh = 105.0

	pr := m.GetPositionRisk(pos, 102.0)

	const eps = 1e-9
	wantSL := 100.0 * 0.95
	if !approxEqual(pr.StopLossPrice, wantSL, eps) {
		t.Errorf("StopLossPrice = %f, want %f", pr.StopLossPrice, wantSL)
	}
	wantTP := 100.0 * 1.10
	if !approxEqual(pr.TakeProfitPrice, wantTP, eps) {
		t.Errorf("TakeProfitPrice = %f, want %f", pr.TakeProfitPrice, wantTP)
	}
	wantTS := 105.0 * 0.92
	if !approxEqual(pr.TrailingStopPrice, wantTS, eps) {
		t.Errorf("TrailingStopPrice = %f, want %f", pr.TrailingStopPrice, wantTS)
	}
	if pr.RiskRewardRatio <= 0 {
		t.Errorf("RiskRewardRatio should be positive, got %f", pr.RiskRewardRatio)
	}
}
