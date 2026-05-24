package main

import (
	"fmt"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/indicators"
	"github.com/CCAtAlvis/backgommon/pkg/portfolio"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

func main() {
	fmt.Println("=== Backgommon Hello World Demo ===")
	fmt.Println()

	prices := []float64{
		100.0, 102.5, 101.0, 103.5, 105.0,
		104.0, 106.5, 108.0, 107.0, 110.0,
		109.5, 112.0, 111.0, 113.5, 115.0,
		114.0, 116.5, 118.0, 117.5, 120.0,
	}

	candles := make([]core.Candle, len(prices))
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, p := range prices {
		candles[i] = core.Candle{
			Time:   baseTime.Add(time.Duration(i) * 24 * time.Hour),
			Open:   p - 0.5,
			High:   p + 1.0,
			Low:    p - 1.0,
			Close:  p,
			Volume: int64(1000 + i*100),
		}
	}

	fmt.Println("--- Technical Indicators ---")

	sma5 := indicators.NewSMA(5)
	smaValues := sma5.Calculate(candles)
	fmt.Printf("\n%s values (last 5 candles):\n", sma5.Name())
	for i := len(candles) - 5; i < len(candles); i++ {
		if smaValues[i] != nil {
			fmt.Printf("  Day %2d | Close: %6.2f | SMA: %6.2f\n", i+1, candles[i].Close, smaValues[i].(float64))
		}
	}

	ema5 := indicators.NewEMA(5)
	emaValues := ema5.Calculate(candles)
	fmt.Printf("\n%s values (last 5 candles):\n", ema5.Name())
	for i := len(candles) - 5; i < len(candles); i++ {
		if emaValues[i] != nil {
			fmt.Printf("  Day %2d | Close: %6.2f | EMA: %6.2f\n", i+1, candles[i].Close, emaValues[i].(float64))
		}
	}

	fmt.Println("\n--- TimeseriesTable ---")

	table := types.NewTimeseriesTable[core.Candle]([]string{"AAPL"})
	for _, c := range candles {
		row := map[string]core.Candle{"AAPL": c}
		if err := table.AddRow(c.Time, row); err != nil {
			fmt.Printf("Error adding row: %v\n", err)
		}
	}
	fmt.Printf("Table created with %d rows and columns: %v\n", len(candles), table.Cols())

	if err := table.ApplyIndicatorToColumn(sma5, "AAPL"); err != nil {
		fmt.Printf("Error applying SMA: %v\n", err)
	} else {
		fmt.Printf("Applied %s indicator to AAPL column\n", sma5.Name())
	}

	lastTime := candles[len(candles)-1].Time
	if val, ok := table.GetValue(lastTime, "AAPL"); ok {
		if ind, err := val.GetIndicator(sma5.Name()); err == nil {
			fmt.Printf("Last candle %s = %.2f\n", sma5.Name(), ind.(float64))
		}
	}

	fmt.Println("\n--- Portfolio Management ---")

	p := portfolio.New(&portfolio.Settings{
		InitialCapital:  100000.0,
		EnableShorts:    false,
		DefaultLeverage: 1.0,
	})
	fmt.Printf("Portfolio created with $%.2f initial capital\n", p.Cash())

	buyOrder := portfolio.NewOrder("AAPL", portfolio.Long, portfolio.Entry, 10, 1.0)
	buyOrder.Price = 115.0
	if err := p.ProcessOrder(buyOrder); err != nil {
		fmt.Printf("Buy order error: %v\n", err)
	} else {
		fmt.Printf("Bought 10 AAPL @ $115.00 | Cash remaining: $%.2f\n", p.Cash())
	}

	p.UpdatePositions(map[string]float64{"AAPL": 120.0})

	stats := p.GetPortfolioStats()
	fmt.Printf("\nPortfolio Stats:\n")
	fmt.Printf("  Total Value:      $%.2f\n", stats.TotalValue)
	fmt.Printf("  Cash:             $%.2f\n", stats.Cash)
	fmt.Printf("  Open Positions:   %d\n", stats.OpenPositions)
	fmt.Printf("  Unrealized P&L:   $%.2f\n", stats.TotalUnrealizedPnL)

	sellOrder := portfolio.NewOrder("AAPL", portfolio.Long, portfolio.Exit, 10, 1.0)
	sellOrder.Price = 120.0
	if err := p.ProcessOrder(sellOrder); err != nil {
		fmt.Printf("Sell order error: %v\n", err)
	} else {
		fmt.Printf("\nSold 10 AAPL @ $120.00 | Cash: $%.2f\n", p.Cash())
	}

	finalStats := p.GetPortfolioStats()
	fmt.Printf("\nFinal Portfolio Stats:\n")
	fmt.Printf("  Total Value:      $%.2f\n", finalStats.TotalValue)
	fmt.Printf("  Cash:             $%.2f\n", finalStats.Cash)
	fmt.Printf("  Closed Trades:    %d\n", finalStats.ClosedPositions)
	fmt.Printf("  Realized P&L:     $%.2f\n", finalStats.TotalRealizedPnL)
	fmt.Printf("  Winning Trades:   %d\n", finalStats.WinningTrades)

	fmt.Println("\n=== Demo Complete ===")
}
