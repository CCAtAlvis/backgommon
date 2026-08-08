package main

import (
	"fmt"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/indicators"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
	"github.com/CCAtAlvis/backgommon/pkg/strategy"
)

// SMAStrategyConfig configures the SMA crossover reference strategy.
type SMAStrategyConfig struct {
	Symbol      string `json:"symbol"`
	ShortWindow int    `json:"short_window"` // e.g. 50 — faster SMA
	LongWindow  int    `json:"long_window"`  // e.g. 200 — slower SMA
	Quantity    int    `json:"quantity"`
	CustomField string `json:"custom_field"` // extensibility hook for user metadata / tagging
}

// SMACrossoverStrategy is a long-only SMA crossover: enter on golden cross, exit on death cross.
//
// Indicators must be pre-computed on the TimeseriesTable (see docs/indicators-application.md)
// before the runner starts, or use runner.WithIndicators.
type SMACrossoverStrategy struct {
	strategy.BaseStrategy

	cfg      SMAStrategyConfig
	shortSMA *indicators.SMA
	longSMA  *indicators.SMA

	prevShort float64
	prevLong  float64
	hasPrev   bool
}

// NewSMACrossoverStrategy validates config and returns a ready strategy.
func NewSMACrossoverStrategy(cfg SMAStrategyConfig) (*SMACrossoverStrategy, error) {
	if cfg.Symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	if cfg.ShortWindow <= 0 || cfg.LongWindow <= 0 {
		return nil, fmt.Errorf("short and long windows must be positive")
	}
	if cfg.ShortWindow >= cfg.LongWindow {
		return nil, fmt.Errorf("short window (%d) must be less than long window (%d)", cfg.ShortWindow, cfg.LongWindow)
	}
	if cfg.Quantity <= 0 {
		return nil, fmt.Errorf("quantity must be positive")
	}

	return &SMACrossoverStrategy{
		cfg:      cfg,
		shortSMA: indicators.NewSMA(cfg.ShortWindow),
		longSMA:  indicators.NewSMA(cfg.LongWindow),
	}, nil
}

// Indicators returns indicators that should be applied to the data table before the backtest.
func (s *SMACrossoverStrategy) Indicators() []interfaces.Indicator {
	return []interfaces.Indicator{s.shortSMA, s.longSMA}
}

// OnTick implements golden-cross / death-cross logic using the previous bar's SMAs.
func (s *SMACrossoverStrategy) OnTick(data map[string]core.Candle) []portfolio.Order {
	candle, ok := data[s.cfg.Symbol]
	if !ok {
		return nil
	}

	shortVal, longVal, ok := s.readSMAs(candle)
	if !ok {
		return nil
	}

	defer s.storePrevious(shortVal, longVal)

	if !s.hasPrev {
		return nil
	}

	positions := s.Portfolio.Positions()
	pos, inPosition := positions[s.cfg.Symbol]

	goldenCross := s.prevShort <= s.prevLong && shortVal > longVal
	deathCross := s.prevShort >= s.prevLong && shortVal < longVal

	var orders []portfolio.Order

	if !inPosition && goldenCross {
		orders = append(orders, portfolio.NewOrder(
			s.cfg.Symbol,
			portfolio.Long,
			portfolio.Entry,
			s.cfg.Quantity,
			1.0,
		))
	}

	if inPosition && deathCross {
		qty := s.cfg.Quantity
		if pos != nil && pos.Quantity > 0 {
			qty = pos.Quantity
		}
		orders = append(orders, portfolio.NewOrder(
			s.cfg.Symbol,
			portfolio.Long,
			portfolio.Exit,
			qty,
			1.0,
		))
	}

	return orders
}

func (s *SMACrossoverStrategy) readSMAs(candle core.Candle) (shortVal, longVal float64, ok bool) {
	shortRaw, err1 := candle.GetIndicator(s.shortSMA.Name())
	longRaw, err2 := candle.GetIndicator(s.longSMA.Name())
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	shortVal, ok1 := shortRaw.(float64)
	longVal, ok2 := longRaw.(float64)
	if !ok1 || !ok2 {
		return 0, 0, false
	}
	return shortVal, longVal, true
}

func (s *SMACrossoverStrategy) storePrevious(shortVal, longVal float64) {
	s.prevShort = shortVal
	s.prevLong = longVal
	s.hasPrev = true
}
