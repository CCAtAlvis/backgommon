package core

import (
	"fmt"
	"time"
)

// Candle represents OHLCV data for a single time period
type Candle struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64

	indicators map[string]any
}

// NewCandle creates a new Candle instance
func NewCandle() *Candle {
	return &Candle{
		indicators: make(map[string]any),
	}
}

// SetIndicator sets an indicator value
func (c *Candle) SetIndicator(name string, value any) {
	if c.indicators == nil {
		c.indicators = make(map[string]any)
	}
	c.indicators[name] = value
}

// GetIndicator retrieves an indicator value
func (c *Candle) GetIndicator(name string) (any, error) {
	if value, exists := c.indicators[name]; exists {
		return value, nil
	}
	return nil, fmt.Errorf("indicator %s not found", name)
}

// HasIndicator checks if an indicator exists
func (c *Candle) HasIndicator(name string) bool {
	_, exists := c.indicators[name]
	return exists
}

// GetAllIndicators returns all indicator values
func (c *Candle) GetAllIndicators() map[string]any {
	return c.indicators
}
