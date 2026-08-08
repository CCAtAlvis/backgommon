// Package types provides generic tabular data structures used throughout the
// backtesting framework. Table is a column-oriented, dynamically-typed data store;
// TimeseriesTable extends it with time-indexed access, lazy sorting, and indicator
// application. Results and AccountValue capture backtest output metrics.
package types

import (
	"fmt"
)

// Row is a single table record stored as an ordered slice of column values.
// Element positions correspond to the column order defined at table creation.
type Row []interface{}

// Table is a column-oriented, dynamically-typed data store. Columns are defined
// at creation and rows are appended or modified by index. All cell values are
// stored as interface{}, so type assertions are required on retrieval.
//
// Table is not safe for concurrent use.
type Table struct {
	columns   []string
	columnMap map[string]int
	rows      []Row
}

// NewTable creates an empty Table with the given column names. Column order is
// preserved and determines the positional index used internally for storage.
func NewTable(columns []string) *Table {
	columnMap := make(map[string]int, len(columns))

	for i, columnName := range columns {
		columnMap[columnName] = i
	}

	return &Table{
		columns:   columns,
		columnMap: columnMap,
		rows:      make([]Row, 0),
	}
}

// AddColumn appends a new column to the schema and backfills all existing rows
// with defaultValue. Returns an error if the column name is empty or already exists.
func (t *Table) AddColumn(newColumnName string, defaultValue interface{}) error {
	if newColumnName == "" {
		return fmt.Errorf("column name cannot be empty")
	}

	if _, ok := t.columnMap[newColumnName]; ok {
		return fmt.Errorf("column %s already exists", newColumnName)
	}

	t.columns = append(t.columns, newColumnName)
	t.columnMap[newColumnName] = len(t.columns) - 1

	for i, row := range t.rows {
		row = append(row, defaultValue)
		t.rows[i] = row
	}

	return nil
}

// GetColumnValues returns all values for the named column as a slice, preserving
// row order. Returns (nil, false) if the column does not exist.
func (t Table) GetColumnValues(column string) ([]interface{}, bool) {
	index, ok := t.columnMap[column]
	if !ok {
		return nil, false
	}

	values := make([]interface{}, 0, t.NumRows())
	for _, row := range t.rows {
		value := row[index]
		values = append(values, value)
	}

	return values, true
}

// NewRow appends a nil-initialized row and returns its zero-based index.
// Use SetValueByIndex or InsertRowAtIndex to populate the row afterward.
func (t *Table) NewRow() int {
	row := make(Row, len(t.columns))
	for i := range t.columns {
		row[i] = nil
	}

	t.rows = append(t.rows, row)
	index := len(t.rows)
	return index - 1
}

// AddRow appends a new row and populates it from the given column→value map.
// Returns the new row index, or (-1, error) if any column name is invalid.
func (t *Table) AddRow(row map[string]interface{}) (int, error) {
	newRowIndex := t.NewRow()
	err := t.InsertRowAtIndex(newRowIndex, row)
	if err != nil {
		return -1, err
	}
	return newRowIndex, nil
}

// InsertRowAtIndex overwrites cells at an existing row index using the provided
// column→value map. Only named columns are updated; unlisted columns retain their
// current value. Returns an error if the index is out of range or a column is invalid.
func (t *Table) InsertRowAtIndex(index int, row map[string]interface{}) error {
	if index < 0 || index >= len(t.rows) {
		return fmt.Errorf("index %d out of range", index)
	}

	for col, val := range row {
		err := t.SetValueByIndex(index, col, val)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetRow returns all column values for the row at the given index as a map.
// Returns (nil, false) if the index is out of range.
func (t Table) GetRow(index int) (map[string]interface{}, bool) {
	if index < 0 || index >= len(t.rows) {
		return nil, false
	}

	return t.convertRow(index), true
}

// SetRow overwrites cells at an existing row index. Equivalent to InsertRowAtIndex.
func (t *Table) SetRow(index int, row map[string]interface{}) error {
	return t.InsertRowAtIndex(index, row)
}

// GetValueByIndex retrieves a single cell value by row index and column name.
// Returns (nil, false) if the index is out of range or the column does not exist.
func (t Table) GetValueByIndex(index int, column string) (interface{}, bool) {
	if index < 0 || index >= len(t.rows) {
		return nil, false
	}

	if columnIndex, ok := t.columnMap[column]; ok {
		return t.rows[index][columnIndex], true
	}

	return nil, false
}

// SetValueByIndex updates a single cell. Returns an error if the column does not
// exist or the row index is out of range.
func (t *Table) SetValueByIndex(index int, column string, value interface{}) error {
	if _, ok := t.columnMap[column]; !ok {
		return fmt.Errorf("column %s does not exist", column)
	}

	if index < 0 || index >= len(t.rows) {
		return fmt.Errorf("row by index %d does not exist", index)
	}

	t.rows[index][t.columnMap[column]] = value
	return nil
}

// Iterator returns a channel that yields rows in order. The channel is closed
// after the last row is sent.
//
// WARNING: The sending goroutine blocks if the channel is not drained. If you
// break out of a range loop early, the goroutine will leak. Prefer Rows() for
// iteration that may terminate early.
func (t *Table) Iterator() <-chan Row {
	ch := make(chan Row)
	go func() {
		for _, row := range t.rows {
			ch <- row
		}
		close(ch)
	}()
	return ch
}

// Head returns a new Table containing the first n rows. If n exceeds the row
// count, returns a copy of the entire table. When n <= 0 it defaults to 5.
func (t Table) Head(n int) Table {
	if n >= len(t.rows) {
		return t
	}

	if n <= 0 {
		n = 5
	}

	newTable := NewTable(t.columns)
	for i := 0; i < n; i++ {
		row, _ := t.GetRow(i)
		newTable.AddRow(row)
	}

	return *newTable
}

// Print writes a tab-separated representation of the table to stdout.
// Intended for quick debugging; not suitable for production output.
func (t Table) Print() {
	fmt.Println("Table:")
	for _, column := range t.columns {
		fmt.Printf("%s\t", column)
	}
	fmt.Println()
	for _, row := range t.rows {
		fmt.Println(row)
	}
}

/* HELPER FUNCTIONS */
func (t Table) convertRow(index int) map[string]interface{} {
	result := make(map[string]interface{})
	for _, columnName := range t.columns {
		value, _ := t.GetValueByIndex(index, columnName)
		result[columnName] = value
	}
	return result
}

// NumRows returns the number of rows currently in the table.
func (t Table) NumRows() int {
	return len(t.rows)
}

// NumCols returns the number of columns in the table schema.
func (t Table) NumCols() int {
	return len(t.columns)
}

// Cols returns the ordered column names as defined at table creation.
func (t Table) Cols() []string {
	return t.columns
}

// Rows returns the internal row slice directly. Callers should treat this as
// read-only; appending or modifying elements may corrupt table state.
func (t Table) Rows() []Row {
	return t.rows
}

// Get is a convenience alias for GetValueByIndex.
func (t Table) Get(index int, column string) (interface{}, bool) {
	return t.GetValueByIndex(index, column)
}

// Set is a convenience alias for SetValueByIndex.
func (t *Table) Set(index int, column string, value interface{}) error {
	return t.SetValueByIndex(index, column, value)
}
