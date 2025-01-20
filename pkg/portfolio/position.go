package portfolio

import (
	"fmt"
	"time"
)

// Position represents an open or closed trading position
type Position struct {
	ID         string
	Instrument string
	Side       OrderSide
	Quantity   int
	OpenPrice  float64
	ClosePrice float64
	OpenTime   time.Time
	CloseTime  time.Time
	Status     PositionStatus
	Orders     []Order
	Leverage   float64

	// Risk metrics
	StopLoss         float64
	TakeProfit       float64
	MaxDrawdown      float64
	HighestPrice     float64
	LowestPrice      float64
	UnrealizedPnL    float64
	RealizedPnL      float64
	TrailingStopHigh float64
}

// PositionStatus represents the current state of a position
type PositionStatus int

const (
	Open PositionStatus = iota
	Closed
	PartiallyOpen
)

// NewPosition creates a new position from an entry order
func NewPosition(ord Order) (*Position, error) {
	if ord.Type != Entry {
		return nil, fmt.Errorf("cannot create position from non-entry order type")
	}

	if ord.Quantity <= 0 {
		return nil, fmt.Errorf("invalid order quantity: %d", ord.Quantity)
	}

	if ord.Leverage <= 1 {
		ord.Leverage = 1.0
	}

	pos := &Position{
		ID:               fmt.Sprintf("pos_%d", time.Now().UnixNano()),
		Instrument:       ord.Instrument,
		Side:             ord.Side,
		Quantity:         ord.Quantity,
		OpenPrice:        ord.Price,
		OpenTime:         time.Now(),
		Status:           Open,
		Orders:           make([]Order, 1),
		Leverage:         ord.Leverage,
		TrailingStopHigh: ord.Price, // Initialize trailing stop high with entry price
	}

	// Add the entry order to position history
	pos.Orders = append(pos.Orders, ord)
	return pos, nil
}

// AddOrder adds an order to the position and updates position details
func (pos *Position) AddOrder(ord Order) error {
	// Validate order
	if ord.Instrument != pos.Instrument {
		return fmt.Errorf("order instrument %s does not match position instrument %s",
			ord.Instrument, pos.Instrument)
	}

	if ord.Quantity <= 0 {
		return fmt.Errorf("invalid order quantity: %d", ord.Quantity)
	}

	// Handle order based on type
	switch ord.Type {
	case Entry:
		// For entry orders, update average price and increase quantity
		totalValue := pos.OpenPrice*float64(pos.Quantity) + ord.Price*float64(ord.Quantity)
		newQuantity := pos.Quantity + ord.Quantity
		pos.OpenPrice = totalValue / float64(newQuantity)
		pos.Quantity = newQuantity

	case Exit:
		// For exit orders, reduce quantity and handle partial/full closure
		if ord.Quantity > pos.Quantity {
			return fmt.Errorf("exit quantity %d exceeds position size %d",
				ord.Quantity, pos.Quantity)
		}

		pos.Quantity -= ord.Quantity
		if pos.Quantity == 0 {
			pos.Status = Closed
			pos.ClosePrice = ord.Price
			pos.CloseTime = time.Now()
		} else {
			pos.Status = PartiallyOpen
		}

		// Calculate and update realized PnL
		if pos.Side == Long {
			pnl := float64(ord.Quantity) * (ord.Price - pos.OpenPrice) * pos.Leverage
			pos.RealizedPnL += pnl
		} else {
			pnl := float64(ord.Quantity) * (pos.OpenPrice - ord.Price) * pos.Leverage
			pos.RealizedPnL += pnl
		}
	}

	// Append order to history
	pos.Orders = append(pos.Orders, ord)
	return nil
}

// Helper functions for position management

// Value returns the current value of the position
func (pos *Position) Value(currentPrice float64) float64 {
	return float64(pos.Quantity) * currentPrice * pos.Leverage
}

// UpdatePrice updates position metrics with the latest price
func (pos *Position) UpdatePrice(currentPrice float64) {
	if currentPrice > pos.HighestPrice {
		pos.HighestPrice = currentPrice
	}
	if currentPrice < pos.LowestPrice || pos.LowestPrice == 0 {
		pos.LowestPrice = currentPrice
	}

	// Calculate unrealized PnL with leverage
	if pos.Side == Long {
		pos.UnrealizedPnL = float64(pos.Quantity) * (currentPrice - pos.OpenPrice) * pos.Leverage
	} else {
		pos.UnrealizedPnL = float64(pos.Quantity) * (pos.OpenPrice - currentPrice) * pos.Leverage
	}

	// Update max drawdown
	if pos.Side == Long {
		drawdown := (pos.HighestPrice - currentPrice) / pos.HighestPrice
		if drawdown > pos.MaxDrawdown {
			pos.MaxDrawdown = drawdown
		}
	} else {
		drawdown := (currentPrice - pos.LowestPrice) / pos.LowestPrice
		if drawdown > pos.MaxDrawdown {
			pos.MaxDrawdown = drawdown
		}
	}
}

// New helper functions that weren't in the original code

// ROI returns the Return on Investment for the position
func (pos *Position) ROI() float64 {
	totalPnL := pos.RealizedPnL + pos.UnrealizedPnL
	investment := pos.OpenPrice * float64(pos.Quantity)
	return totalPnL / investment
}

// Duration returns how long the position has been open
func (pos *Position) Duration() time.Duration {
	if pos.Status == Closed {
		return pos.CloseTime.Sub(pos.OpenTime)
	}
	return time.Since(pos.OpenTime)
}
