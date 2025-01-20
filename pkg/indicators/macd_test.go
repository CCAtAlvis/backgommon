package indicators

import (
	"testing"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
)

func TestMACDWithAndWithoutDependencies(t *testing.T) {
	// Create test data
	candles := []core.Candle{
		{Time: time.Now(), Close: 10},
		{Time: time.Now(), Close: 11},
		{Time: time.Now(), Close: 12},
		{Time: time.Now(), Close: 13},
		{Time: time.Now(), Close: 14},
	}

	// Case 1: Without pre-calculated dependencies
	macd := NewMACD(2, 3, 2)
	macdValues1 := macd.Calculate(candles)

	// Case 2: With pre-calculated dependencies
	// Pre-calculate EMAs
	for i := range candles {
		fastEMA := macd.fastEMA.Calculate(candles[:i+1])
		slowEMA := macd.slowEMA.Calculate(candles[:i+1])
		candles[i].SetIndicator(macd.fastEMA.Name(), fastEMA)
		candles[i].SetIndicator(macd.slowEMA.Name(), slowEMA)
	}

	macdValues2 := macd.Calculate(candles)

	// Both calculations should yield the same result for the last value (most recent candle)
	lastIdx := len(candles) - 1
	mv1, ok1 := macdValues1[lastIdx].(MACDValue)
	mv2, ok2 := macdValues2[lastIdx].(MACDValue)
	if !ok1 || !ok2 {
		t.Fatalf("Expected MACDValue at last index, got %v and %v", macdValues1[lastIdx], macdValues2[lastIdx])
	}
	if mv1.Value() != mv2.Value() {
		t.Errorf("MACD values differ: without deps = %v, with deps = %v", mv1.Value(), mv2.Value())
	}
	if mv1.Signal() != mv2.Signal() {
		t.Errorf("Signal values differ: without deps = %v, with deps = %v", mv1.Signal(), mv2.Signal())
	}
	if mv1.Histogram() != mv2.Histogram() {
		t.Errorf("Histogram values differ: without deps = %v, with deps = %v", mv1.Histogram(), mv2.Histogram())
	}
}
