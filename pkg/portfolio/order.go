package portfolio

import (
	"fmt"
	"time"
)

// OrderSide indicates whether a position is directionally long or short.
type OrderSide int

// OrderType distinguishes between orders that open (Entry) or close (Exit) a position.
type OrderType int

const (
	Long  OrderSide = iota // Buy to open / buy to close
	Short                  // Sell to open / sell to close
)

const (
	Entry OrderType = iota // Opens or adds to a position
	Exit                   // Reduces or closes a position
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

// Order represents a trading instruction submitted by a strategy.
//
// Lifecycle: a strategy creates an Order via [NewOrder] or [ExitOrderForPosition]
// with Price and FilledAt left at zero values. The runner's fill-pricing logic
// then sets Price (from bar data + slippage) and FilledAt (bar timestamp) before
// passing the filled order to [Portfolio.ProcessOrder].
type Order struct {
	ID         string
	Instrument string
	Side       OrderSide
	Type       OrderType
	Quantity   int
	Price      float64   // Execution price; zero until filled by the runner
	Leverage   float64   // Leverage for this order; 0 means use Settings.DefaultLeverage
	FilledAt   time.Time // Timestamp of fill; zero until filled by the runner
	Reason     string    // Why exit was triggered (e.g. "stop_loss", "trailing_stop", "take_profit")
}

// ExitOrderForPosition is a convenience constructor that builds a full-exit order
// matching the position's instrument, side, quantity, and leverage. Price is left
// at zero for the runner to fill.
func ExitOrderForPosition(pos *Position) Order {
	return NewOrder(pos.Instrument, pos.Side, Exit, pos.Quantity, pos.Leverage)
}

// NewOrder constructs an order with a unique ID. Price and FilledAt are left at
// zero values; the runner fills them from bar data before execution. If leverage
// is <= 0, it defaults to 1.0 (no leverage).
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

// Fill marks the order as executed at the given price and sets FilledAt to
// time.Now(). In backtesting the runner always sets FilledAt explicitly to the
// bar timestamp before calling ProcessOrder, so this method is primarily useful
// for testing or live adapters.
func (o *Order) Fill(price float64) {
	o.Price = price
	o.FilledAt = time.Now()
}
