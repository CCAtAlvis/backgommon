package types

import "time"

// Results holds backtesting results and analytics
type Results struct {
	StartTime      time.Time
	EndTime        time.Time
	InitialCapital float64
	FinalCapital   float64
	TotalTrades    int
	WinningTrades  int
	LosingTrades   int
	MaxDrawdown    float64
	SharpeRatio    float64
	SortinoRatio   float64
	Returns        float64
	Metrics        map[string]float64 // For custom metrics
}

// AccountValue represents a snapshot of account value at a point in time
type AccountValue struct {
	Time          time.Time
	Value         float64
	Cash          float64
	OpenPositions int
	UnrealizedPnL float64
}
