package execution

import (
	"testing"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

func TestStandardFillPricer_Modes(t *testing.T) {
	bar := core.Candle{Open: 10, High: 15, Low: 8, Close: 12}
	next := core.Candle{Open: 13, Close: 14}
	ord := portfolio.NewOrder("X", portfolio.Long, portfolio.Entry, 1, 1)

	tests := []struct {
		mode FillMode
		want float64
	}{
		{FillCurrentClose, 12},
		{FillCurrentOpen, 10},
		{FillMidPrice, 11.5},
		{FillOpenCloseAvg, 11},
	}

	for _, tc := range tests {
		p := NewStandardFillPricer(tc.mode)
		got, err := p.FillPrice(interfaces.FillContext{Order: ord, CurrentBar: bar})
		if err != nil {
			t.Fatalf("%s: %v", tc.mode, err)
		}
		if got != tc.want {
			t.Fatalf("%s: got %.2f want %.2f", tc.mode, got, tc.want)
		}
	}

	p := NewStandardFillPricer(FillNextBarOpen)
	got, err := p.FillPrice(interfaces.FillContext{Order: ord, CurrentBar: bar, NextBar: &next})
	if err != nil || got != 13 {
		t.Fatalf("next open: got %.2f err %v", got, err)
	}
}

func TestFuncFillPricer_Custom(t *testing.T) {
	p := NewFuncFillPricer(func(ctx interfaces.FillContext) (float64, error) {
		return ctx.CurrentBar.Close * 1.01, nil
	})
	got, err := p.FillPrice(interfaces.FillContext{
		Order:      portfolio.NewOrder("X", portfolio.Long, portfolio.Entry, 1, 1),
		CurrentBar: core.Candle{Close: 100},
	})
	if err != nil || got != 101 {
		t.Fatalf("custom fill: got %.2f err %v", got, err)
	}
}
