package risk

import (
	"math"

	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

// checkExitConditions checks if a position should be exited
func (m *Manager) checkExitConditions(pos *portfolio.Position, currentPrice float64) (bool, string) {
	if !m.settings.UseStopLoss && !m.settings.UseTakeProfit && !m.settings.UseTrailingStop {
		return false, ""
	}

	if pos.Side == portfolio.Long {
		return m.checkLongExitConditions(pos, currentPrice)
	}
	return m.checkShortExitConditions(pos, currentPrice)
}

func (m *Manager) checkLongExitConditions(pos *portfolio.Position, currentPrice float64) (bool, string) {
	// Stop Loss
	if m.settings.UseStopLoss {
		stopPrice := pos.OpenPrice * (1 - m.settings.DefaultStopLoss)
		if currentPrice <= stopPrice {
			return true, "stop_loss"
		}
	}

	// Take Profit
	if m.settings.UseTakeProfit {
		takeProfitPrice := pos.OpenPrice * (1 + m.settings.DefaultTakeProfit)
		if currentPrice >= takeProfitPrice {
			return true, "take_profit"
		}
	}

	// Trailing Stop
	if m.settings.UseTrailingStop {
		if currentPrice > pos.TrailingStopHigh {
			pos.TrailingStopHigh = currentPrice
		}
		stopPrice := pos.TrailingStopHigh * (1 - m.settings.DefaultTrailingStop)
		if currentPrice <= stopPrice {
			return true, "trailing_stop"
		}
	}

	return false, ""
}

func (m *Manager) checkShortExitConditions(pos *portfolio.Position, currentPrice float64) (bool, string) {
	// Stop Loss
	if m.settings.UseStopLoss {
		stopPrice := pos.OpenPrice * (1 + m.settings.DefaultStopLoss)
		if currentPrice >= stopPrice {
			return true, "stop_loss"
		}
	}

	// Take Profit
	if m.settings.UseTakeProfit {
		takeProfitPrice := pos.OpenPrice * (1 - m.settings.DefaultTakeProfit)
		if currentPrice <= takeProfitPrice {
			return true, "take_profit"
		}
	}

	// Trailing Stop
	if m.settings.UseTrailingStop {
		if currentPrice < pos.TrailingStopHigh {
			pos.TrailingStopHigh = currentPrice
		}
		stopPrice := pos.TrailingStopHigh * (1 + m.settings.DefaultTrailingStop)
		if currentPrice >= stopPrice {
			return true, "trailing_stop"
		}
	}

	return false, ""
}

// createExitOrder creates an exit order for a position
func createExitOrder(pos *portfolio.Position, reason string) portfolio.Order {
	return portfolio.NewOrder(
		pos.Instrument,
		pos.Side,
		portfolio.Exit,
		pos.Quantity,
		pos.Leverage,
	)
}

// Additional helper functions for risk management

// GetPositionRisk calculates the risk metrics for a position
func (m *Manager) GetPositionRisk(pos *portfolio.Position, currentPrice float64) PositionRisk {
	return PositionRisk{
		StopLossPrice:     calculateStopLossPrice(pos, m.settings.DefaultStopLoss),
		TakeProfitPrice:   calculateTakeProfitPrice(pos, m.settings.DefaultTakeProfit),
		TrailingStopPrice: calculateTrailingStopPrice(pos, m.settings.DefaultTrailingStop),
		MaxLoss:           calculateMaxLoss(pos, currentPrice, m.settings.DefaultStopLoss),
		RiskRewardRatio:   calculateRiskRewardRatio(pos, currentPrice, m.settings),
	}
}

// PositionRisk holds risk metrics for a position
type PositionRisk struct {
	StopLossPrice     float64
	TakeProfitPrice   float64
	TrailingStopPrice float64
	MaxLoss           float64
	RiskRewardRatio   float64
}

// Helper functions for risk calculations
func calculateStopLossPrice(pos *portfolio.Position, stopLoss float64) float64 {
	if pos.Side == portfolio.Long {
		return pos.OpenPrice * (1 - stopLoss)
	}
	return pos.OpenPrice * (1 + stopLoss)
}

func calculateTakeProfitPrice(pos *portfolio.Position, takeProfit float64) float64 {
	if pos.Side == portfolio.Long {
		return pos.OpenPrice * (1 + takeProfit)
	}
	return pos.OpenPrice * (1 - takeProfit)
}

func calculateTrailingStopPrice(pos *portfolio.Position, trailingStop float64) float64 {
	if pos.Side == portfolio.Long {
		return pos.TrailingStopHigh * (1 - trailingStop)
	}
	return pos.TrailingStopHigh * (1 + trailingStop)
}

func calculateMaxLoss(pos *portfolio.Position, currentPrice, stopLoss float64) float64 {
	stopLossPrice := calculateStopLossPrice(pos, stopLoss)
	return float64(pos.Quantity) * math.Abs(currentPrice-stopLossPrice) * pos.Leverage
}

func calculateRiskRewardRatio(pos *portfolio.Position, currentPrice float64, settings *Settings) float64 {
	if !settings.UseStopLoss || !settings.UseTakeProfit {
		return 0
	}

	stopLossPrice := calculateStopLossPrice(pos, settings.DefaultStopLoss)
	takeProfitPrice := calculateTakeProfitPrice(pos, settings.DefaultTakeProfit)

	risk := math.Abs(currentPrice - stopLossPrice)
	reward := math.Abs(takeProfitPrice - currentPrice)

	if risk == 0 {
		return 0
	}
	return reward / risk
}
