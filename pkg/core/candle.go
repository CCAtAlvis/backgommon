// Package core provides the foundational data types for the backgommon backtesting framework.
// The primary type is Candle, which represents a single OHLCV bar and serves as the
// fundamental data atom flowing through the entire backtesting pipeline — from data loading
// through indicator computation to strategy signal generation.
package core

import (
	"fmt"
	"time"
)

// Candle is the fundamental market data unit in the backtesting pipeline.
// It carries OHLCV price data for a single time period plus an extensible indicator
// storage map. Indicators (SMA, RSI, etc.) are computed over candle slices and their
// results are attached via SetIndicator, making each Candle self-contained with both
// raw price data and derived analytics.
//
// The indicators map is lazily initialized on first SetIndicator call, so zero-value
// Candles are safe to use without calling NewCandle. Values in the map are typed as
// any — callers should type-assert to the expected indicator output type (typically
// float64 or nil for insufficient-lookback periods).
type Candle struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64

	indicators map[string]any
}

// NewCandle creates a Candle with a pre-allocated indicator map.
// Use this when you know indicators will be attached; otherwise a zero-value Candle
// is equally valid since SetIndicator initializes the map lazily.
func NewCandle() *Candle {
	return &Candle{
		indicators: make(map[string]any),
	}
}

// SetIndicator stores a computed indicator value on this candle. The map is created
// lazily on first call, so it is safe to call on zero-value Candles. Overwrites any
// existing value for the same name without error.
func (c *Candle) SetIndicator(name string, value any) {
	if c.indicators == nil {
		c.indicators = make(map[string]any)
	}
	c.indicators[name] = value
}

// GetIndicator retrieves a previously stored indicator value by name.
// Returns an error if the indicator has not been set on this candle — use HasIndicator
// for a non-error check, or handle the error to distinguish "not computed" from "nil value".
func (c *Candle) GetIndicator(name string) (any, error) {
	if value, exists := c.indicators[name]; exists {
		return value, nil
	}
	return nil, fmt.Errorf("indicator %s not found", name)
}

// HasIndicator reports whether an indicator with the given name has been stored,
// regardless of whether its value is nil (e.g., insufficient lookback period).
func (c *Candle) HasIndicator(name string) bool {
	_, exists := c.indicators[name]
	return exists
}

// GetAllIndicators returns the internal indicator map directly (not a copy).
// Callers may read freely but should not mutate the map concurrently with
// SetIndicator calls. Returns nil if no indicators have been set.
func (c *Candle) GetAllIndicators() map[string]any {
	return c.indicators
}

// Clone returns a deep copy of the candle. OHLCV fields are copied by value and
// the indicators map is duplicated so SetIndicator on the clone does not affect
// the original. Indicator values themselves are shallow-copied (typical float64
// / nil payloads are immutable in practice).
func (c Candle) Clone() Candle {
	out := Candle{
		Time:   c.Time,
		Open:   c.Open,
		High:   c.High,
		Low:    c.Low,
		Close:  c.Close,
		Volume: c.Volume,
	}
	if c.indicators != nil {
		out.indicators = make(map[string]any, len(c.indicators))
		for k, v := range c.indicators {
			out.indicators[k] = v
		}
	}
	return out
}
