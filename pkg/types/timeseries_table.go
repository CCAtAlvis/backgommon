package types

import (
	"fmt"
	"sort"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
)

// TimeseriesTable is a time-indexed generic table that wraps Table with timestamp-based
// access. Each row is uniquely identified by a time.Time key rather than a numeric index.
//
// Rows may be inserted in any order; the table uses lazy sorting (isDirty flag) so that
// chronological ordering is only enforced when iterating via Iterator() or Rows().
//
// The type parameter T constrains the value type stored in cells. When T is core.Candle,
// the indicator methods (ApplyIndicator, ApplyIndicators, etc.) become usable — they
// extract candle data from columns, compute indicator values, and write results back
// via Candle.SetIndicator. For non-Candle T, indicator methods will silently skip columns.
type TimeseriesTable[T any] struct {
	table        *Table
	timestampMap map[time.Time]int
	timestampArr []time.Time
	isDirty      bool
}

// TimeseriesRow is a lightweight handle to a single row in a TimeseriesTable.
// It carries the Timestamp key and a back-pointer to the parent table, enabling
// deferred data retrieval via Get() and GetValue().
type TimeseriesRow[T any] struct {
	Timestamp time.Time
	table     *TimeseriesTable[T]
}

// NewTimeseriesTable creates an empty timeseries table with the specified column names.
// Columns typically represent instrument symbols
func NewTimeseriesTable[T any](columns []string) *TimeseriesTable[T] {
	return &TimeseriesTable[T]{
		table:        NewTable(columns),
		timestampMap: make(map[time.Time]int),
		timestampArr: []time.Time{},
		isDirty:      false,
	}
}

// CreateRow allocates a new empty row for the given timestamp. Returns an error if
// a row with that timestamp already exists. Marks the table as dirty (unsorted).
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

// SetRow overwrites cell values for an existing row identified by timestamp.
// The row must have been previously created via CreateRow or AddRow.
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

// AddRow is a convenience that combines CreateRow + SetRow in a single call.
// Returns an error if the timestamp already exists or if any column name is invalid.
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

// GetRow retrieves all column values for the row at the given timestamp.
// Returns (nil, false) if the timestamp does not exist in the table.
//
// NOTE: GetRow allocates a new map on every call. In hot loops (e.g. the runner
// bar loop), prefer FillRow with a reused destination map.
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

// FillRow copies the row at timestamp into dst, overwriting keys for this
// table's columns. It does not allocate a new map. Missing/nil cells write the
// zero value of T. Returns false if the timestamp is not in the table.
func (t *TimeseriesTable[T]) FillRow(timestamp time.Time, dst map[string]T) bool {
	if dst == nil {
		return false
	}
	index, ok := t.GetIndexFor(timestamp)
	if !ok {
		return false
	}
	if index < 0 || index >= len(t.table.rows) {
		return false
	}
	row := t.table.rows[index]
	cols := t.table.columns
	var zero T
	for i, col := range cols {
		if i >= len(row) || row[i] == nil {
			dst[col] = zero
			continue
		}
		if v, ok := row[i].(T); ok {
			dst[col] = v
		} else {
			dst[col] = zero
		}
	}
	return true
}

// GetIndexFor returns the internal Table row index for a timestamp.
// Returns (-1, false) if the timestamp is not registered.
func (t TimeseriesTable[T]) GetIndexFor(timestamp time.Time) (int, bool) {
	index, ok := t.timestampMap[timestamp]
	if !ok {
		return -1, false
	}

	return index, true
}

// GetValue retrieves a single cell by timestamp and column name.
// Returns the zero value of T and false if the timestamp or column is not found.
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

// SetValue updates a single cell identified by timestamp and column.
// The row must already exist; returns an error if the timestamp is not found.
func (t *TimeseriesTable[T]) SetValue(timestamp time.Time, column string, value T) error {
	index, ok := t.GetIndexFor(timestamp)
	if !ok {
		return fmt.Errorf("timestamp %s not found", timestamp)
	}

	t.table.Set(index, column, value)
	return nil
}

// Iterator returns a channel yielding rows in chronological order. If the table
// is dirty (rows were added out of order), it sorts the timestamp index first.
//
// WARNING: The sending goroutine blocks if the channel is not fully drained.
// If you break early, the goroutine will leak. Prefer Rows() for interruptible iteration.
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

// Rows returns all rows as TimeseriesRow handles in chronological order.
// Sorts the timestamp index if dirty. This is the primary iteration method used
// by the runner — prefer it over Iterator() to avoid goroutine leak risks.
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

// Cols returns the column names of the underlying table.
func (t TimeseriesTable[T]) Cols() []string {
	return t.table.Cols()
}

// Head returns the first n rows of the underlying Table (untyped). Delegates to
// Table.Head; when n <= 0 it defaults to 5. Useful for quick debugging.
func (t TimeseriesTable[T]) Head(n int) Table {
	return t.table.Head(n)
}

// Print writes a tab-separated dump of the underlying table to stdout.
func (t TimeseriesTable[T]) Print() {
	t.table.Print()
}

// Clone returns a new TimeseriesTable with the same schema and timestamps.
// Each cell is copied via cloneCell so callers can isolate mutable cell state
// (e.g. Candle indicator maps) across parallel or repeated backtests.
func (t *TimeseriesTable[T]) Clone(cloneCell func(T) T) *TimeseriesTable[T] {
	cols := append([]string(nil), t.Cols()...)
	out := NewTimeseriesTable[T](cols)

	// Preserve chronological order used by Rows()/Iterator.
	for _, ts := range t.Rows() {
		srcRow, ok := t.GetRow(ts.Timestamp)
		if !ok {
			continue
		}
		dstRow := make(map[string]T, len(srcRow))
		for col, val := range srcRow {
			dstRow[col] = cloneCell(val)
		}
		_ = out.AddRow(ts.Timestamp, dstRow)
	}
	return out
}

// CloneCandleTable deep-copies a candle timeseries so indicator application on
// the result does not mutate the template. Equivalent to t.Clone(Candle.Clone).
func CloneCandleTable(t *TimeseriesTable[core.Candle]) *TimeseriesTable[core.Candle] {
	if t == nil {
		return nil
	}
	return t.Clone(func(c core.Candle) core.Candle { return c.Clone() })
}

// Get retrieves all column values for this row's timestamp.
// Returns (nil, false) if the row's timestamp has been removed from the table.
func (r TimeseriesRow[T]) Get() (map[string]T, bool) {
	return r.table.GetRow(r.Timestamp)
}

// GetValue retrieves a single column value for this row's timestamp.
func (r TimeseriesRow[T]) GetValue(column string) (T, bool) {
	return r.table.GetValue(r.Timestamp, column)
}

// applyIndicatorWithDependencies recursively applies an indicator's dependency tree
// before applying the indicator itself. The applied map prevents duplicate computation.
func (t *TimeseriesTable[T]) applyIndicatorWithDependencies(indicator interfaces.Indicator, column string, applied map[string]bool) error {
	if applied[indicator.Name()] {
		return nil
	}

	for _, dep := range indicator.Dependencies() {
		err := t.applyIndicatorWithDependencies(dep, column, applied)
		if err != nil {
			return fmt.Errorf("failed to apply dependency %s: %v", dep.Name(), err)
		}
	}

	err := t.ApplyIndicatorToColumn(indicator, column)
	if err != nil {
		return err
	}

	applied[indicator.Name()] = true
	return nil
}

// ApplyIndicator computes an indicator across all columns that contain core.Candle data.
// Non-Candle columns are silently skipped. Dependencies are resolved automatically.
// Results are stored on each Candle via SetIndicator using the indicator's Name() as key.
//
// See also: pkg/indicators for built-in indicators (SMA, EMA, RSI, etc.).
func (t *TimeseriesTable[T]) ApplyIndicator(indicator interfaces.Indicator) error {
	for _, col := range t.Cols() {
		applied := make(map[string]bool)
		err := t.applyIndicatorWithDependencies(indicator, col, applied)
		if err != nil {
			continue
		}
	}
	return nil
}

// ApplyIndicatorToColumn computes an indicator for a specific column only.
// The column must contain core.Candle values (or *core.Candle); non-candle rows
// are skipped. The indicator's Calculate method receives candles in timestamp order
// and must return exactly len(candles) values.
//
// After computation, each candle is updated via SetIndicator and written back to the table.
func (t *TimeseriesTable[T]) ApplyIndicatorToColumn(indicator interfaces.Indicator, column string) error {
	candles := make([]core.Candle, 0, len(t.timestampArr))
	timestampsWithData := make([]time.Time, 0, len(t.timestampArr))

	for _, ts := range t.timestampArr {
		candleData, ok := t.GetRow(ts)
		if !ok {
			continue
		}

		candleValue, ok := candleData[column]
		if !ok {
			continue
		}

		if ptr, isPtr := any(candleValue).(*core.Candle); isPtr && ptr == nil {
			continue
		}

		var candle core.Candle
		if c, ok := any(candleValue).(core.Candle); ok {
			candle = c
		} else if c, ok := any(candleValue).(*core.Candle); ok {
			candle = *c
		} else {
			continue
		}

		candles = append(candles, candle)
		timestampsWithData = append(timestampsWithData, ts)
	}

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

// ApplyIndicators computes multiple indicators across all Candle columns, resolving
// inter-indicator dependencies automatically. Non-Candle columns are skipped.
// This is the batch equivalent of calling ApplyIndicator for each indicator.
func (t *TimeseriesTable[T]) ApplyIndicators(indicators []interfaces.Indicator) error {
	for _, col := range t.Cols() {
		applied := make(map[string]bool)
		for _, ind := range indicators {
			err := t.applyIndicatorWithDependencies(ind, col, applied)
			if err != nil {
				break
			}
		}
	}
	return nil
}

// ApplyIndicatorsToColumn computes multiple indicators on a single named column,
// resolving dependencies in topological order. Returns an error if any indicator
// fails to compute (unlike ApplyIndicators which skips failing columns).
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
