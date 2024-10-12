package order

type OrderSide int

const (
	Long OrderSide = iota
	Short
)

type OrderType int

const (
	Entry OrderType = iota
	Exit
)

type Price int

const (
	Open = iota
	Close
)

type Order struct {
	OrderId    int
	Instrument string
	OrderSide  OrderSide
	OrderType  OrderType
	Quantity   int
	Price      float64

	Leverage             float64
	StopLoss             float64
	TrailingStopLoss     float64
	TrailingStopLossStep float64
}
