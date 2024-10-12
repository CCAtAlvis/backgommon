package runner

import (
	"github.com/CCAtAlvis/backgommon/src/structs"
	"github.com/CCAtAlvis/backgommon/src/structs/order"
)

func executeOrders(r *Runner, orders []order.Order) {
	for _, ord := range orders {
		var position *structs.Position
		pos, ok := r.AccountData.OpenPositions[ord.Instrument]
		if ok {
			position, ok = pos[ord.OrderSide]
		}

		if !ok {
			position = &structs.Position{
				Instrument: ord.Instrument,
				Side:       ord.OrderSide,
				Quantity:   0,
				OpenPrice:  0,
				AvgPrice:   0,
				OpenDate:   r.CurrentTime,
				Leverage:   ord.Leverage,
			}
		}

		if ord.OrderSide == order.Long {
			if ord.OrderType == order.Entry {
				r.Functions.HandleLongEntry(r, position, ord)
			} else if ord.OrderType == order.Exit {
			}

			r.AccountData.OpenPositions[ord.Instrument][ord.OrderSide] = position
		} else if ord.OrderSide == order.Short {
			if ord.OrderType == order.Entry {
			} else if ord.OrderType == order.Exit {
			}
		}

		r.AccountData.Orders = append(r.AccountData.Orders, &ord)
	}
}

// TODO: handle leverage
func handleLongEntry(r *Runner, position *structs.Position, ord order.Order) {
	newQuantity := position.Quantity + ord.Quantity
	avgPrice := (position.AvgPrice*float64(position.Quantity) + ord.Price*float64(ord.Quantity)) / float64(newQuantity)

	if position.OpenPrice == 0 {
		position.OpenDate = r.CurrentTime
		position.OpenPrice = avgPrice
	}

	position.AvgPrice = avgPrice
	position.Quantity = newQuantity

	position.Orders = append(position.Orders, &ord)
}

// TODO: implement cost of leverage
func handleLongExit(r *Runner, position *structs.Position, ord order.Order) {
	cost := position.AvgPrice * float64(ord.Quantity) // cost of accquistion
	profit := ord.Price*float64(ord.Quantity) - cost
	position.Quantity -= ord.Quantity

	if position.Quantity == 0 {
		position.CloseDate = r.CurrentTime
		position.ClosePrice = ord.Price
		r.AccountData.ClosedPositions = append(r.AccountData.ClosedPositions, *position)
		delete(r.AccountData.OpenPositions[position.Instrument], ord.OrderSide)
	}

	if r.Settings.IsTaxEnabled {
		r.Functions.HandleTax(r, position, ord)
	}

	r.AccountData.TotalPnL += profit
	r.AccountData.FreeAmount += (profit + cost)

	pocketAmount := 0.0
	if r.Settings.IsPocketEnabled && profit > r.Settings.MinProfitForPocket {
		pocketAmount = profit * r.Settings.PocketPercent
		r.AccountData.TotalPocket += pocketAmount
		r.AccountData.FreeAmount -= profit * r.Settings.PocketPercent
	}
}

func handleShortEntry(r *Runner, position *structs.Position, ord order.Order) {
}

func handleShortExit(r *Runner, position *structs.Position, ord order.Order) {
}

func handleTax(r *Runner, position *structs.Position, ord order.Order) {
	tax := 0.0

	if ord.OrderType == order.Entry {
		buyValue := ord.Price * float64(ord.Quantity)
		buyTax := buyValue * r.Settings.BuySideTaxPercent

		tax = buyTax
		r.AccountData.TotalBuySideTax += buyTax
	} else if ord.OrderType == order.Exit {
		sellValue := ord.Price * float64(ord.Quantity)
		sellTax := sellValue * r.Settings.SellSideTaxPercent

		cost := position.AvgPrice * float64(ord.Quantity) // cost of accquistion
		profit := sellValue - cost
		stcg := 0.0
		if profit > 0 {
			stcg = profit * r.Settings.STCGTaxPercent
		}

		tax = sellTax + stcg
		r.AccountData.TotalSTCGTax += stcg
		r.AccountData.TotalSellSideTax += sellTax
	}

	r.AccountData.TotalTax += tax
	r.AccountData.FreeAmount -= tax
}
