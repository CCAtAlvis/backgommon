package types

import (
	"fmt"
	"sort"
	"time"
)

type TimeseriesTable[T any] struct {
	table        *Table
	timestampMap map[time.Time]int
	timestampArr []time.Time
	isDirty      bool
}

func NewTimeseriesTable[T any](columns []string) *TimeseriesTable[T] {
	return &TimeseriesTable[T]{
		table:        NewTable(columns),
		timestampMap: make(map[time.Time]int),
		timestampArr: make([]time.Time, len(columns)),
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

type TimeseriesRow[T any] struct {
	Timestamp time.Time
	table     *TimeseriesTable[T]
}

func (r TimeseriesRow[T]) Get() (map[string]T, bool) {
	return r.table.GetRow(r.Timestamp)
}

func (r TimeseriesRow[T]) GetValue(column string) (T, bool) {
	return r.table.GetValue(r.Timestamp, column)
}
