package portfolio

import (
	"fmt"
	"time"
)

// OrderSide represents the side of an order (Long/Short)
type OrderSide int

// OrderType represents the type of order (Entry/Exit)
type OrderType int

const (
	Long OrderSide = iota
	Short
)

const (
	Entry OrderType = iota
	Exit
)

// String returns the string representation of OrderSide.
func (s OrderSide) String() string {
	switch s {
	case Long:
		return "Long"
	case Short:
		return "Short"
	default:
		return fmt.Sprintf("OrderSide(%d)", s)
	}
}

// Opposite returns the opposing side of the current OrderSide.
func (s OrderSide) Opposite() OrderSide {
	switch s {
	case Long:
		return Short
	case Short:
		return Long
	default:
		// This case should ideally not be reached if OrderSide is always valid.
		// Consider panicking or returning a defined invalid side if necessary.
		return s // Or handle error appropriately
	}
}

// Order represents a trading order
type Order struct {
	ID         string
	Instrument string
	Side       OrderSide
	Type       OrderType
	Quantity   int
	Price      float64
	Leverage   float64
	FilledAt   time.Time
}

// NewOrder creates a new order
func NewOrder(instrument string, side OrderSide, orderType OrderType, qty int, leverage float64) Order {
	if leverage <= 0 {
		leverage = 1.0
	}

	return Order{
		ID:         fmt.Sprintf("ord_%d", time.Now().UnixNano()),
		Instrument: instrument,
		Side:       side,
		Type:       orderType,
		Quantity:   qty,
		Leverage:   leverage,
	}
}

// Fill marks the order as filled at the given price
func (o *Order) Fill(price float64) {
	o.Price = price
	o.FilledAt = time.Now()
}
