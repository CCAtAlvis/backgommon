package portfolio

import (
	"testing"
	"time"
)

func makeEntryOrder(instrument string, side OrderSide, qty int, price float64) Order {
	return Order{
		ID:         "test",
		Instrument: instrument,
		Side:       side,
		Type:       Entry,
		Quantity:   qty,
		Price:      price,
		Leverage:   1.0,
		FilledAt:   time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
	}
}

func makeExitOrder(instrument string, side OrderSide, qty int, price float64) Order {
	return Order{
		ID:         "test-exit",
		Instrument: instrument,
		Side:       side,
		Type:       Exit,
		Quantity:   qty,
		Price:      price,
		Leverage:   1.0,
		FilledAt:   time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC),
	}
}

func TestNewPositionUniqueIDs(t *testing.T) {
	ts := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		ord := Order{
			ID: "ord", Instrument: "AAPL", Side: Long, Type: Entry,
			Quantity: 10, Price: 100, Leverage: 1, FilledAt: ts,
		}
		pos, err := NewPosition(ord)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ids[pos.ID] {
			t.Fatalf("duplicate ID: %s", pos.ID)
		}
		ids[pos.ID] = true
	}
}

func TestAddOrderEntry(t *testing.T) {
	ord := makeEntryOrder("AAPL", Long, 10, 100.0)
	pos, err := NewPosition(ord)
	if err != nil {
		t.Fatalf("NewPosition: %v", err)
	}

	if pos.Quantity != 10 {
		t.Errorf("quantity = %d, want 10", pos.Quantity)
	}
	if pos.OpenPrice != 100.0 {
		t.Errorf("OpenPrice = %f, want 100", pos.OpenPrice)
	}
	if pos.PeakQty != 10 {
		t.Errorf("PeakQty = %d, want 10", pos.PeakQty)
	}
	if pos.Status != Open {
		t.Errorf("Status = %d, want Open", pos.Status)
	}
}

func TestAddOrderMultipleEntries(t *testing.T) {
	pos, _ := NewPosition(makeEntryOrder("AAPL", Long, 10, 100.0))

	err := pos.AddOrder(Order{
		Instrument: "AAPL", Side: Long, Type: Entry,
		Quantity: 10, Price: 200.0, Leverage: 1.0,
	})
	if err != nil {
		t.Fatalf("AddOrder: %v", err)
	}

	if pos.Quantity != 20 {
		t.Errorf("quantity = %d, want 20", pos.Quantity)
	}
	wantAvg := (100.0*10 + 200.0*10) / 20
	if pos.OpenPrice != wantAvg {
		t.Errorf("OpenPrice = %f, want %f", pos.OpenPrice, wantAvg)
	}
	if pos.PeakQty != 20 {
		t.Errorf("PeakQty = %d, want 20", pos.PeakQty)
	}
}

func TestAddOrderPartialExit(t *testing.T) {
	pos, _ := NewPosition(makeEntryOrder("AAPL", Long, 10, 100.0))

	exit := Order{
		Instrument: "AAPL", Side: Long, Type: Exit,
		Quantity: 5, Price: 120.0, Leverage: 1.0,
		FilledAt: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC),
	}
	if err := pos.AddOrder(exit); err != nil {
		t.Fatalf("AddOrder exit: %v", err)
	}

	if pos.Quantity != 5 {
		t.Errorf("quantity = %d, want 5", pos.Quantity)
	}
	if pos.Status != PartiallyOpen {
		t.Errorf("Status = %d, want PartiallyOpen", pos.Status)
	}
	wantPnL := 5.0 * (120.0 - 100.0) * 1.0
	if pos.RealizedPnL != wantPnL {
		t.Errorf("RealizedPnL = %f, want %f", pos.RealizedPnL, wantPnL)
	}
}

func TestAddOrderFullExit(t *testing.T) {
	pos, _ := NewPosition(makeEntryOrder("AAPL", Long, 10, 100.0))

	exit := makeExitOrder("AAPL", Long, 10, 110.0)
	if err := pos.AddOrder(exit); err != nil {
		t.Fatalf("AddOrder exit: %v", err)
	}

	if pos.Quantity != 0 {
		t.Errorf("quantity = %d, want 0", pos.Quantity)
	}
	if pos.Status != Closed {
		t.Errorf("Status = %d, want Closed", pos.Status)
	}
	if pos.CloseTime.IsZero() {
		t.Error("CloseTime should be set")
	}
	wantPnL := 10.0 * (110.0 - 100.0)
	if pos.RealizedPnL != wantPnL {
		t.Errorf("RealizedPnL = %f, want %f", pos.RealizedPnL, wantPnL)
	}
}

func TestPeakQtyTracking(t *testing.T) {
	pos, _ := NewPosition(makeEntryOrder("AAPL", Long, 10, 100.0))
	pos.AddOrder(Order{Instrument: "AAPL", Side: Long, Type: Entry, Quantity: 5, Price: 105.0, Leverage: 1.0})

	if pos.PeakQty != 15 {
		t.Errorf("PeakQty after second entry = %d, want 15", pos.PeakQty)
	}

	pos.AddOrder(Order{
		Instrument: "AAPL", Side: Long, Type: Exit, Quantity: 3, Price: 110.0, Leverage: 1.0,
		FilledAt: time.Date(2025, 1, 3, 10, 0, 0, 0, time.UTC),
	})

	if pos.Quantity != 12 {
		t.Errorf("quantity after partial exit = %d, want 12", pos.Quantity)
	}
	if pos.PeakQty != 15 {
		t.Errorf("PeakQty should remain 15 after exit, got %d", pos.PeakQty)
	}
}

func TestROI(t *testing.T) {
	pos, _ := NewPosition(makeEntryOrder("AAPL", Long, 10, 100.0))
	pos.UpdatePrice(120.0)

	roi := pos.ROI()
	wantROI := (pos.RealizedPnL + pos.UnrealizedPnL) / (100.0 * 10)
	if roi != wantROI {
		t.Errorf("ROI = %f, want %f", roi, wantROI)
	}

	posLev, _ := NewPosition(Order{
		Instrument: "AAPL", Side: Long, Type: Entry,
		Quantity: 10, Price: 100.0, Leverage: 2.0,
		FilledAt: time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
	})
	posLev.UpdatePrice(120.0)

	roiLev := posLev.ROI()
	if roiLev <= roi {
		t.Errorf("leveraged ROI (%f) should exceed unleveraged (%f)", roiLev, roi)
	}
}

func TestDuration(t *testing.T) {
	pos, _ := NewPosition(makeEntryOrder("AAPL", Long, 10, 100.0))
	exit := makeExitOrder("AAPL", Long, 10, 110.0)
	pos.AddOrder(exit)

	want := exit.FilledAt.Sub(pos.OpenTime)
	got := pos.Duration()
	if got != want {
		t.Errorf("Duration = %v, want %v", got, want)
	}
}
