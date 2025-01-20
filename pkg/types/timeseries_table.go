package types

import (
	"fmt"
	"sort"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
)

// TimeseriesTable represents a time-series data structure
type TimeseriesTable[T any] struct {
	table        *Table
	timestampMap map[time.Time]int
	timestampArr []time.Time
	isDirty      bool
}

type TimeseriesRow[T any] struct {
	Timestamp time.Time
	table     *TimeseriesTable[T]
}

func NewTimeseriesTable[T any](columns []string) *TimeseriesTable[T] {
	return &TimeseriesTable[T]{
		table:        NewTable(columns),
		timestampMap: make(map[time.Time]int),
		timestampArr: []time.Time{},
		isDirty:      false,
	}
}

func (t *TimeseriesTable[T]) CreateRow(timestamp time.Time) error {
	if _, ok := t.timestampMap[timestamp]; ok {
		return fmt.Errorf("timestamp %s already exists, failed creating new row", timestamp)
	}

	index := t.table.NewRow()
	t.timestampMap[timestamp] = index
	t.timestampArr = append(t.timestampArr, timestamp)
	t.isDirty = true

	return nil
}

func (t *TimeseriesTable[T]) SetRow(timestamp time.Time, row map[string]T) error {
	index, _ := t.GetIndexFor(timestamp)

	interfaceValues := make(map[string]interface{})
	for key, value := range row {
		interfaceValues[key] = value
	}

	err := t.table.SetRow(index, interfaceValues)
	if err != nil {
		return err
	}

	return nil
}

func (t *TimeseriesTable[T]) AddRow(timestamp time.Time, row map[string]T) error {
	err := t.CreateRow(timestamp)
	if err != nil {
		return err
	}

	err = t.SetRow(timestamp, row)
	if err != nil {
		return err
	}

	return nil
}

func (t TimeseriesTable[T]) GetRow(timestamp time.Time) (map[string]T, bool) {
	index, ok := t.GetIndexFor(timestamp)
	if !ok {
		return nil, false
	} else {
		interfaceMap, _ := t.table.GetRow(index) // ignoring ok as GetIndexFor is already checked
		typedMap := make(map[string]T)
		for key, value := range interfaceMap {
			typedValue, _ := value.(T) // ignoring type assertion error as setting of values is type checked
			typedMap[key] = typedValue
		}
		return typedMap, true
	}
}

func (t TimeseriesTable[T]) GetIndexFor(timestamp time.Time) (int, bool) {
	index, ok := t.timestampMap[timestamp]
	if !ok {
		return -1, false
	}

	return index, true
}

func (t TimeseriesTable[T]) GetValue(timestamp time.Time, column string) (T, bool) {
	index, ok := t.GetIndexFor(timestamp)
	if !ok {
		var zero T
		return zero, false
	} else {
		value, ok := t.table.Get(index, column)
		if !ok {
			var zero T
			return zero, false
		}
		assertedValue, _ := value.(T) // ignoring type assertion error as setting of values is type checked
		return assertedValue, true
	}
}

func (t *TimeseriesTable[T]) SetValue(timestamp time.Time, column string, value T) error {
	index, ok := t.GetIndexFor(timestamp)
	if !ok {
		return fmt.Errorf("timestamp %s not found", timestamp)
	}

	t.table.Set(index, column, value)
	return nil
}

func (t *TimeseriesTable[T]) Iterator() <-chan map[string]T {
	if t.isDirty {
		sort.Slice(t.timestampArr, func(i, j int) bool {
			return t.timestampArr[i].Before(t.timestampArr[j])
		})
		t.isDirty = false
	}

	ch := make(chan map[string]T)
	go func() {
		for _, timestamp := range t.timestampArr {
			row, _ := t.GetRow(timestamp)
			ch <- row
		}
		close(ch)
	}()
	return ch
}

func (t *TimeseriesTable[T]) Rows() []TimeseriesRow[T] {
	if t.isDirty {
		sort.Slice(t.timestampArr, func(i, j int) bool {
			return t.timestampArr[i].Before(t.timestampArr[j])
		})
		t.isDirty = false
	}

	rows := make([]TimeseriesRow[T], len(t.timestampArr))
	for i, timestamp := range t.timestampArr {
		rows[i] = TimeseriesRow[T]{
			Timestamp: timestamp,
			table:     t,
		}
	}
	return rows
}

func (t TimeseriesTable[T]) Cols() []string {
	return t.table.Cols()
}

func (t TimeseriesTable[T]) Head(n int) Table {
	return t.table.Head(n)
}

func (t TimeseriesTable[T]) Print() {
	t.table.Print()
}

func (r TimeseriesRow[T]) Get() (map[string]T, bool) {
	return r.table.GetRow(r.Timestamp)
}

func (r TimeseriesRow[T]) GetValue(column string) (T, bool) {
	return r.table.GetValue(r.Timestamp, column)
}

// applyIndicatorWithDependencies applies an indicator and its dependencies
func (t *TimeseriesTable[T]) applyIndicatorWithDependencies(indicator interfaces.Indicator, column string, applied map[string]bool) error {
	// Check if already applied
	if applied[indicator.Name()] {
		return nil
	}

	// First apply dependencies
	for _, dep := range indicator.Dependencies() {
		err := t.applyIndicatorWithDependencies(dep, column, applied)
		if err != nil {
			return fmt.Errorf("failed to apply dependency %s: %v", dep.Name(), err)
		}
	}

	// Then apply this indicator
	err := t.ApplyIndicatorToColumn(indicator, column)
	if err != nil {
		return err
	}

	applied[indicator.Name()] = true
	return nil
}

// ApplyIndicator applies an indicator to all Candle columns in the timeseries data
func (t *TimeseriesTable[T]) ApplyIndicator(indicator interfaces.Indicator) error {
	// Apply to all columns that contain Candle data
	for _, col := range t.Cols() {
		applied := make(map[string]bool)
		err := t.applyIndicatorWithDependencies(indicator, col, applied)
		if err != nil {
			// Skip columns that don't contain Candle data
			continue
		}
	}
	return nil
}

// ApplyIndicatorToColumn applies an indicator to a specific column in the timeseries data
func (t *TimeseriesTable[T]) ApplyIndicatorToColumn(indicator interfaces.Indicator, column string) error {
	// Collect only the candles and timestamps where the column has a valid candle
	candles := make([]core.Candle, 0, len(t.timestampArr))
	timestampsWithData := make([]time.Time, 0, len(t.timestampArr))

	for _, ts := range t.timestampArr {
		candleData, ok := t.GetRow(ts)
		if !ok {
			continue // skip if row is missing
		}

		candleValue, ok := candleData[column]
		if !ok {
			continue // skip if column is missing
		}

		// Check for nil if candleValue is a pointer
		if ptr, isPtr := any(candleValue).(*core.Candle); isPtr && ptr == nil {
			continue // skip if pointer is nil
		}

		var candle core.Candle
		if c, ok := any(candleValue).(core.Candle); ok {
			candle = c
		} else if c, ok := any(candleValue).(*core.Candle); ok {
			candle = *c
		} else {
			continue // skip non-candle data
		}

		candles = append(candles, candle)
		timestampsWithData = append(timestampsWithData, ts)
	}

	// Calculate indicator values for each candle
	values := indicator.Calculate(candles)
	if len(values) != len(candles) {
		return fmt.Errorf("indicator.Calculate returned %d values for %d candles", len(values), len(candles))
	}

	for i, ts := range timestampsWithData {
		candles[i].SetIndicator(indicator.Name(), values[i])
		err := t.SetValue(ts, column, any(candles[i]).(T))
		if err != nil {
			return fmt.Errorf("failed to update indicator value: %v", err)
		}
	}

	return nil
}

// ApplyIndicators applies multiple indicators to all Candle columns
func (t *TimeseriesTable[T]) ApplyIndicators(indicators []interfaces.Indicator) error {
	applied := make(map[string]bool)
	for _, ind := range indicators {
		err := t.applyIndicatorWithDependencies(ind, "", applied)
		if err != nil {
			return fmt.Errorf("failed to apply indicator %s: %v", ind.Name(), err)
		}
	}
	return nil
}

// ApplyIndicatorsToColumn applies multiple indicators to a specific column
func (t *TimeseriesTable[T]) ApplyIndicatorsToColumn(indicators []interfaces.Indicator, column string) error {
	applied := make(map[string]bool)
	for _, ind := range indicators {
		err := t.applyIndicatorWithDependencies(ind, column, applied)
		if err != nil {
			return fmt.Errorf("failed to apply indicator %s to column %s: %v", ind.Name(), column, err)
		}
	}
	return nil
}
