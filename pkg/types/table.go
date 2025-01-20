package types

import (
	"fmt"
)

type Row []interface{}

type Table struct {
	columns   []string
	columnMap map[string]int
	rows      []Row
}

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

func (t Table) GetColumnValues(column string) ([]interface{}, bool) {
	index, ok := t.columnMap[column]
	if !ok {
		return nil, false
	}

	values := make([]interface{}, t.NumRows())
	for _, row := range t.rows {
		value := row[index]
		values = append(values, value)
	}

	return values, true
}

func (t *Table) NewRow() int {
	row := make(Row, len(t.columns))
	for i := range t.columns {
		row[i] = nil
	}

	t.rows = append(t.rows, row)
	index := len(t.rows)
	return index - 1
}

func (t *Table) AddRow(row map[string]interface{}) (int, error) {
	newRowIndex := t.NewRow()
	err := t.InsertRowAtIndex(newRowIndex, row)
	if err != nil {
		return -1, err
	}
	return newRowIndex, nil
}

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

func (t Table) GetRow(index int) (map[string]interface{}, bool) {
	if index < 0 || index >= len(t.rows) {
		return nil, false
	}

	return t.convertRow(index), true
}

func (t *Table) SetRow(index int, row map[string]interface{}) error {
	return t.InsertRowAtIndex(index, row)
}

func (t Table) GetValueByIndex(index int, column string) (interface{}, bool) {
	if index < 0 || index >= len(t.rows) {
		return nil, false
	}

	if columnIndex, ok := t.columnMap[column]; ok {
		return t.rows[index][columnIndex], true
	}

	return nil, false
}

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

func (t Table) NumRows() int {
	return len(t.rows)
}

func (t Table) NumCols() int {
	return len(t.columns)
}

func (t Table) Cols() []string {
	return t.columns
}

func (t Table) Rows() []Row {
	return t.rows
}

func (t Table) Get(index int, column string) (interface{}, bool) {
	return t.GetValueByIndex(index, column)
}

func (t *Table) Set(index int, column string, value interface{}) error {
	return t.SetValueByIndex(index, column, value)
}
