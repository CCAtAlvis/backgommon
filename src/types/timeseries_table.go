package types

import (
	"fmt"
	"sort"
	"time"
)

type TimeseriesTable struct {
	table        *Table
	timestampMap map[time.Time]int
	timestampArr []time.Time
	isDirty      bool
}

func NewTimeseriesTable(columns []string) *TimeseriesTable {
	return &TimeseriesTable{
		table:        NewTable(columns),
		timestampMap: make(map[time.Time]int),
		timestampArr: make([]time.Time, len(columns)),
		isDirty:      false,
	}
}

func (t *TimeseriesTable) CreateRow(timestamp time.Time) error {
	if _, ok := t.timestampMap[timestamp]; ok {
		return fmt.Errorf("timestamp %s already exists, failed creating new row", timestamp)
	}

	index := t.table.NewRow()
	t.timestampMap[timestamp] = index
	t.timestampArr = append(t.timestampArr, timestamp)
	t.isDirty = true

	return nil
}

func (t *TimeseriesTable) SetRow(timestamp time.Time, values map[string]interface{}) error {
	index, _ := t.GetIndexFor(timestamp)
	err := t.table.SetRow(index, values)
	if err != nil {
		return err
	}

	return nil
}

func (t *TimeseriesTable) AddRow(timestamp time.Time, values map[string]interface{}) error {
	err := t.CreateRow(timestamp)
	if err != nil {
		return err
	}

	err = t.SetRow(timestamp, values)
	if err != nil {
		return err
	}

	return nil
}

func (t TimeseriesTable) GetRow(timestamp time.Time) (map[string]interface{}, bool) {
	index, ok := t.GetIndexFor(timestamp)
	if !ok {
		return nil, false
	} else {
		return t.table.GetRow(index)
	}
}

func (t TimeseriesTable) GetIndexFor(timestamp time.Time) (int, bool) {
	index, ok := t.timestampMap[timestamp]
	if !ok {
		return -1, false
	}

	return index, true
}

func (t TimeseriesTable) GetValue(timestamp time.Time, column string) (interface{}, bool) {
	index, ok := t.GetIndexFor(timestamp)
	if !ok {
		return nil, false
	} else {
		return t.table.Get(index, column)
	}
}

func (t *TimeseriesTable) SetValue(timestamp time.Time, column string, value interface{}) error {
	index, ok := t.GetIndexFor(timestamp)
	if !ok {
		return fmt.Errorf("timestamp %s not found", timestamp)
	}

	t.table.Set(index, column, value)
	return nil
}

func (t *TimeseriesTable) Iterator() <-chan map[string]interface{} {
	if t.isDirty {
		sort.Slice(t.timestampArr, func(i, j int) bool {
			return t.timestampArr[i].Before(t.timestampArr[j])
		})
		t.isDirty = false
	}

	ch := make(chan map[string]interface{})
	go func() {
		for _, timestamp := range t.timestampArr {
			row, _ := t.GetRow(timestamp)
			ch <- row
		}
		close(ch)
	}()
	return ch
}
