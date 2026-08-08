package risk

import (
	"math"

	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
)

// checkExitConditions evaluates stop-loss, take-profit, and trailing-stop
// rules against currentPrice for a single position. It returns whether the
// position should be closed and a reason string for logging.
//
// IMPORTANT: this method has a side-effect — when a trailing stop is enabled
// it updates pos.TrailingStopHigh in place (raising it for longs, lowering it
// for shorts) to track the high-water mark. Callers must be aware that the
// Position is mutated even when no exit is triggered.
func (m *Manager) checkExitConditions(pos *portfolio.Position, currentPrice float64) (bool, string) {
	if !m.settings.EnableStopLoss && !m.settings.EnableTakeProfit && !m.settings.EnableTrailingStop {
		return false, ""
	}

	if pos.Side == portfolio.Long {
		return m.checkLongExitConditions(pos, currentPrice)
	}
	return m.checkShortExitConditions(pos, currentPrice)
}

func (m *Manager) checkLongExitConditions(pos *portfolio.Position, currentPrice float64) (bool, string) {
	if m.settings.EnableStopLoss {
		if currentPrice <= calculateStopLossPrice(pos, m.settings.DefaultStopLossRate) {
			return true, "stop_loss"
		}
	}

	if m.settings.EnableTakeProfit {
		if currentPrice >= calculateTakeProfitPrice(pos, m.settings.DefaultTakeProfitRate) {
			return true, "take_profit"
		}
	}

	if m.settings.EnableTrailingStop {
		if currentPrice > pos.TrailingStopHigh {
			pos.TrailingStopHigh = currentPrice
		}
		if currentPrice <= calculateTrailingStopPrice(pos, m.settings.DefaultTrailingStopRate) {
			return true, "trailing_stop"
		}
	}

	return false, ""
}

func (m *Manager) checkShortExitConditions(pos *portfolio.Position, currentPrice float64) (bool, string) {
	if m.settings.EnableStopLoss {
		if currentPrice >= calculateStopLossPrice(pos, m.settings.DefaultStopLossRate) {
			return true, "stop_loss"
		}
	}

	if m.settings.EnableTakeProfit {
		if currentPrice <= calculateTakeProfitPrice(pos, m.settings.DefaultTakeProfitRate) {
			return true, "take_profit"
		}
	}

	if m.settings.EnableTrailingStop {
		if currentPrice < pos.TrailingStopHigh {
			pos.TrailingStopHigh = currentPrice
		}
		if currentPrice >= calculateTrailingStopPrice(pos, m.settings.DefaultTrailingStopRate) {
			return true, "trailing_stop"
		}
	}

	return false, ""
}

// createExitOrder builds an exit order that fully closes the given position,
// preserving the original side and leverage. The reason string is stored on the
// returned Order so downstream code can inspect why the exit was triggered.
func createExitOrder(pos *portfolio.Position, reason string) portfolio.Order {
	ord := portfolio.NewOrder(
		pos.Instrument,
		pos.Side,
		portfolio.Exit,
		pos.Quantity,
		pos.Leverage,
	)
	ord.Reason = reason
	return ord
}

// GetPositionRisk computes a snapshot of the current risk levels for a
// position: where the stop-loss, take-profit, and trailing-stop would
// trigger, the maximum monetary loss if the stop-loss fires, and the
// risk-reward ratio. Useful for dashboards, logging, and strategy
// introspection. Does not mutate any state.
func (m *Manager) GetPositionRisk(pos *portfolio.Position, currentPrice float64) PositionRisk {
	return PositionRisk{
		StopLossPrice:     calculateStopLossPrice(pos, m.settings.DefaultStopLossRate),
		TakeProfitPrice:   calculateTakeProfitPrice(pos, m.settings.DefaultTakeProfitRate),
		TrailingStopPrice: calculateTrailingStopPrice(pos, m.settings.DefaultTrailingStopRate),
		MaxLoss:           calculateMaxLoss(pos, currentPrice, m.settings.DefaultStopLossRate),
		RiskRewardRatio:   calculateRiskRewardRatio(pos, currentPrice, m.settings),
	}
}

// PositionRisk holds a point-in-time snapshot of the risk metrics for a
// single open position. All prices are absolute (not percentages).
type PositionRisk struct {
	StopLossPrice     float64
	TakeProfitPrice   float64
	TrailingStopPrice float64
	MaxLoss           float64
	RiskRewardRatio   float64
}

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
	if !settings.EnableStopLoss || !settings.EnableTakeProfit {
		return 0
	}

	stopLossPrice := calculateStopLossPrice(pos, settings.DefaultStopLossRate)
	takeProfitPrice := calculateTakeProfitPrice(pos, settings.DefaultTakeProfitRate)

	risk := math.Abs(currentPrice - stopLossPrice)
	reward := math.Abs(takeProfitPrice - currentPrice)

	if risk == 0 {
		return 0
	}
	return reward / risk
}
