package portfolio

import (
	"fmt"
	"sync/atomic"
	"time"
)

// posCounter provides globally unique, monotonically increasing IDs for positions.
var posCounter uint64

// Position tracks the lifecycle of a single instrument holding from entry through
// partial or full exit. It accumulates PnL, records all constituent orders, and
// maintains price extremes for drawdown calculation.
type Position struct {
	ID         string
	Instrument string
	Side       OrderSide
	Quantity   int // Current remaining quantity (decreases on partial exits)
	PeakQty   int // Maximum quantity held before any partial exits

	OpenPrice  float64 // Volume-weighted average entry price (updated on scale-in)
	ClosePrice float64 // Set to exit price when Status becomes Closed
	OpenTime   time.Time
	CloseTime  time.Time
	Status     PositionStatus
	Orders     []Order  // Full order history (entries and exits) for this position
	Leverage   float64  // Leverage applied at open; inherited from the entry order

	// StopLoss and TakeProfit are reserved for per-position risk overrides.
	// They are not currently enforced by the framework's risk manager.
	StopLoss   float64
	TakeProfit float64

	MaxDrawdown   float64 // Worst observed drawdown (0–1 fraction) from price peak/trough
	HighestPrice  float64 // Highest price seen while position is open
	LowestPrice   float64 // Lowest price seen while position is open
	UnrealizedPnL float64 // Mark-to-market PnL on remaining quantity
	RealizedPnL   float64 // Accumulated PnL from partial and full exits

	// TrailingStopHigh is the high-water mark tracked by the risk manager's
	// trailing-stop exit condition. Mutated as a side-effect during risk checks.
	TrailingStopHigh float64
}

// PositionStatus represents the lifecycle state of a position.
type PositionStatus int

const (
	Open          PositionStatus = iota // Position is fully active
	Closed                              // Position fully exited; moved to closed history
	PartiallyOpen                       // Some quantity exited, remainder still active
)

// NewPosition creates a new position from an entry order. It generates a unique
// ID using an atomic counter (safe across goroutines, though Portfolio itself is
// not concurrent). Returns an error if the order is not an Entry type or has
// invalid quantity.
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

	openTime := ord.FilledAt
	if openTime.IsZero() {
		openTime = time.Now()
	}

	seq := atomic.AddUint64(&posCounter, 1)
	pos := &Position{
		ID:               fmt.Sprintf("pos_%d_%d", openTime.Unix(), seq),
		Instrument:       ord.Instrument,
		Side:             ord.Side,
		Quantity:         ord.Quantity,
		PeakQty:          ord.Quantity,
		OpenPrice:        ord.Price,
		OpenTime:         openTime,
		Status:           Open,
		Orders:           []Order{ord},
		Leverage:         ord.Leverage,
		TrailingStopHigh: ord.Price,
	}

	return pos, nil
}

// AddOrder applies an order to the position, updating quantity and PnL.
//
// For Entry orders: computes a new volume-weighted average OpenPrice, increases
// Quantity, and updates PeakQty if the new total exceeds the previous peak.
//
// For Exit orders: reduces Quantity, accumulates RealizedPnL for the exited
// portion (leveraged), and transitions Status to PartiallyOpen or Closed.
//
// Returns an error if the order's instrument doesn't match, quantity is invalid,
// or exit quantity exceeds the current position size.
func (pos *Position) AddOrder(ord Order) error {
	if ord.Instrument != pos.Instrument {
		return fmt.Errorf("order instrument %s does not match position instrument %s",
			ord.Instrument, pos.Instrument)
	}

	if ord.Quantity <= 0 {
		return fmt.Errorf("invalid order quantity: %d", ord.Quantity)
	}

	switch ord.Type {
	case Entry:
		totalValue := pos.OpenPrice*float64(pos.Quantity) + ord.Price*float64(ord.Quantity)
		newQuantity := pos.Quantity + ord.Quantity
		pos.OpenPrice = totalValue / float64(newQuantity)
		pos.Quantity = newQuantity
		if newQuantity > pos.PeakQty {
			pos.PeakQty = newQuantity
		}

	case Exit:
		if ord.Quantity > pos.Quantity {
			return fmt.Errorf("exit quantity %d exceeds position size %d",
				ord.Quantity, pos.Quantity)
		}

		pos.Quantity -= ord.Quantity
		if pos.Quantity == 0 {
			pos.Status = Closed
			pos.ClosePrice = ord.Price
			closeTime := ord.FilledAt
			if closeTime.IsZero() {
				closeTime = time.Now()
			}
			pos.CloseTime = closeTime
		} else {
			pos.Status = PartiallyOpen
		}

		if pos.Side == Long {
			pnl := float64(ord.Quantity) * (ord.Price - pos.OpenPrice) * pos.Leverage
			pos.RealizedPnL += pnl
		} else {
			pnl := float64(ord.Quantity) * (pos.OpenPrice - ord.Price) * pos.Leverage
			pos.RealizedPnL += pnl
		}
	}

	pos.Orders = append(pos.Orders, ord)
	return nil
}

// Value returns the leveraged notional value of the position at the given price.
func (pos *Position) Value(currentPrice float64) float64 {
	return float64(pos.Quantity) * currentPrice * pos.Leverage
}

// UpdatePrice refreshes mark-to-market metrics with the latest price: updates
// HighestPrice/LowestPrice extremes, recalculates UnrealizedPnL (leverage-aware),
// and tracks MaxDrawdown from peak (long) or trough (short).
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

// ROI returns the total return on investment: (realized + unrealized PnL) divided
// by the current notional investment at the average open price.
func (pos *Position) ROI() float64 {
	totalPnL := pos.RealizedPnL + pos.UnrealizedPnL
	investment := pos.OpenPrice * float64(pos.Quantity)
	return totalPnL / investment
}

// Duration returns the elapsed time the position has been held. For closed
// positions it returns CloseTime - OpenTime; for open positions it returns
// time since OpenTime (wall-clock, so meaningful only in live contexts).
func (pos *Position) Duration() time.Duration {
	if pos.Status == Closed {
		return pos.CloseTime.Sub(pos.OpenTime)
	}
	return time.Since(pos.OpenTime)
}
