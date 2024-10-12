package structs

import (
	"time"

	"github.com/CCAtAlvis/backgommon/src/structs/order"
)

type Frequency int

const (
	Hourly Frequency = iota
	Daily
	Monthly
	Yearly
)

type Settings struct {
	StartingAmount float64
	SipAmount      float64
	SipFrequency   Frequency
	StopLoss       float64
	Leverage       float64

	StartDate time.Time
	EndDate   time.Time

	IsSlippageEnabled bool
	SlippagePercent   float64

	IsBrockrageEnabled bool
	BrockrageFixedCost float64
	BrockragePercent   float64

	IsTaxEnabled       bool
	BuySideTaxPercent  float64
	BuySideTaxFixed    float64
	SellSideTaxPercent float64
	SellSideTaxFixed   float64
	STCGTaxPercent     float64
	LTCGTaxPercent     float64

	IsPocketEnabled    bool
	MinProfitForPocket float64
	PocketPercent      float64

	IsManagementFeeEnabled bool
	ManagementFeePercent   float64
	ManagementFeeFixed     float64
	ManagementFeeFrequency Frequency
}

type AccountData struct {
	AccountValue    float64
	FreeAmount      float64
	OpenPositions   map[string]map[order.OrderSide]*Position
	ClosedPositions []Position
	EquityCurve     []AccountValue
	Orders          []*order.Order

	// for logging
	TotalInvestment  float64
	TotalPnL         float64
	TotalTax         float64
	TotalSTCGTax     float64
	TotalLTCGTax     float64
	TotalBuySideTax  float64
	TotalSellSideTax float64
	TotalPocket      float64
}

type Position struct {
	Instrument string
	Side       order.OrderSide

	Quantity   int
	OpenDate   time.Time
	OpenPrice  float64
	AvgPrice   float64
	CloseDate  time.Time
	ClosePrice float64
	Orders     []*order.Order
	Leverage   float64
}

type AccountValue struct {
	Date            string
	Value           float64
	FreeAmount      float64
	PortfolioAmount float64
	OpenPositions   []Position
	LogValue        float64
}
