// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sqlchemy

import (
	"fmt"
	"reflect"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"
)

// ITableSpec is the interface represents a table
type ITableSpec interface {
	// Insert performs an insert operation that insert one record at a time
	Insert(dt interface{}) error

	// InsertOrUpdate performs an atomic insert or update operation that insert a new record to update the record with current value
	InsertOrUpdate(dt interface{}) error

	// Update performs an update operation
	Update(dt interface{}, onUpdate func() error) (UpdateDiffs, error)

	// Increment performs a special update that do an atomic incremental update of the numeric fields
	Increment(diff, target interface{}) error

	// Decrement performs a special update that do an atomic decremental update of the numeric fields
	Decrement(diff, target interface{}) error

	// DataType returns the data type corresponding to the table
	DataType() reflect.Type

	// ColumnSpec returns the column definition of a spcific column
	ColumnSpec(name string) IColumnSpec

	// Name returns the name of the table
	Name() string

	// Columns returns the array of columns definitions
	Columns() []IColumnSpec

	// PrimaryColumns returns the array of columns of primary keys
	PrimaryColumns() []IColumnSpec

	// Expression returns expression of the table
	Expression() string

	// Instance returns an instance of STable for this spec
	Instance() *STable

	// DropForeignKeySQL returns the SQL statements to drop foreignkeys for this table
	DropForeignKeySQL() []string

	// AddIndex adds index to table
	AddIndex(unique bool, cols ...string) bool

	// SyncSQL forces synchronize the data definition and model definition of the table
	SyncSQL() []string

	// Fetch query a struct
	Fetch(dt interface{}) error
}

// STableSpec defines the table specification, which implements ITableSpec
type STableSpec struct {
	structType reflect.Type
	name       string
	columns    []IColumnSpec
	indexes    []sTableIndex
	contraints []sTableConstraint
}

// STable is an instance of table for query, system will automatically give a alias to this table
type STable struct {
	spec  ITableSpec
	alias string
}

// STableField represents a field in a table, implements IQueryField
type STableField struct {
	table *STable
	spec  IColumnSpec
	alias string
}

// NewTableSpecFromStruct generates STableSpec based on the information of a struct model
func NewTableSpecFromStruct(s interface{}, name string) *STableSpec {
	val := reflect.Indirect(reflect.ValueOf(s))
	st := val.Type()
	if st.Kind() != reflect.Struct {
		panic("expect Struct kind")
	}
	table := &STableSpec{
		columns:    []IColumnSpec{},
		name:       name,
		structType: st,
	}
	struct2TableSpec(val, table)
	return table
}

// Name implementation of STableSpec for ITableSpec
func (ts *STableSpec) Name() string {
	return ts.name
}

// Expression implementation of STableSpec for ITableSpec
func (ts *STableSpec) Expression() string {
	return fmt.Sprintf("`%s`", ts.name)
}

// Clone makes a clone of a table, so we may create a new table of the same schema
func (ts *STableSpec) Clone(name string, autoIncOffset int64) *STableSpec {
	newCols := make([]IColumnSpec, len(ts.columns))
	for i := range newCols {
		col := ts.columns[i]
		if intCol, ok := col.(*SIntegerColumn); ok && intCol.IsAutoIncrement {
			newCol := *intCol
			newCol.AutoIncrementOffset = autoIncOffset
			newCols[i] = &newCol
		} else {
			newCols[i] = col
		}
	}
	return &STableSpec{
		structType: ts.structType,
		name:       name,
		columns:    newCols,
		indexes:    ts.indexes,
		contraints: ts.contraints,
	}
}

// Columns implementation of STableSpec for ITableSpec
func (ts *STableSpec) Columns() []IColumnSpec {
	return ts.columns
}

// PrimaryColumns implementation of STableSpec for ITableSpec
func (ts *STableSpec) PrimaryColumns() []IColumnSpec {
	ret := make([]IColumnSpec, 0)
	for i := range ts.columns {
		if ts.columns[i].IsPrimary() {
			ret = append(ret, ts.columns[i])
		}
	}
	return ret
}

// DataType implementation of STableSpec for ITableSpec
func (ts *STableSpec) DataType() reflect.Type {
	return ts.structType
}

// CreateSQL returns the SQL for creating this table
func (ts *STableSpec) CreateSQL() string {
	cols := make([]string, 0)
	primaries := make([]string, 0)
	indexes := make([]string, 0)
	autoInc := ""
	for _, c := range ts.columns {
		cols = append(cols, c.DefinitionString())
		if c.IsPrimary() {
			primaries = append(primaries, fmt.Sprintf("`%s`", c.Name()))
			if intC, ok := c.(*SIntegerColumn); ok && intC.AutoIncrementOffset > 1 {
				autoInc = fmt.Sprintf(" AUTO_INCREMENT=%d", intC.AutoIncrementOffset)
			}
		}
		if c.IsIndex() {
			indexes = append(indexes, fmt.Sprintf("KEY `ix_%s_%s` (`%s`)", ts.name, c.Name(), c.Name()))
		}
	}
	if len(primaries) > 0 {
		cols = append(cols, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaries, ", ")))
	}
	if len(indexes) > 0 {
		cols = append(cols, indexes...)
	}
	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (\n%s\n) ENGINE=InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci%s", ts.name, strings.Join(cols, ",\n"), autoInc)
}

// NewTableInstance return an new table instance from an ITableSpec
func NewTableInstance(ts ITableSpec) *STable {
	table := STable{spec: ts, alias: getTableAliasName()}
	return &table
}

// Instance return an new table instance from an instance of STableSpec
func (ts *STableSpec) Instance() *STable {
	return NewTableInstance(ts)
}

// ColumnSpec implementation of STableSpec for ITableSpec
func (ts *STableSpec) ColumnSpec(name string) IColumnSpec {
	for _, c := range ts.Columns() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

// Field implementation of STableSpec for IQuerySource
func (tbl *STable) Field(name string, alias ...string) IQueryField {
	// name = reflectutils.StructFieldName(name)
	name = utils.CamelSplit(name, "_")
	spec := tbl.spec.ColumnSpec(name)
	if spec == nil {
		log.Warningf("column %s not found in table %s", name, tbl.spec.Name())
		return nil
	}
	col := STableField{table: tbl, spec: spec}
	if len(alias) > 0 {
		col.Label(alias[0])
	}
	return &col
}

// Fields implementation of STable for IQuerySource
func (tbl *STable) Fields() []IQueryField {
	ret := make([]IQueryField, 0)
	for _, c := range tbl.spec.Columns() {
		ret = append(ret, tbl.Field(c.Name()))
	}
	return ret
}

// Expression implementation of STableField for IQueryField
func (c *STableField) Expression() string {
	if len(c.alias) > 0 {
		return fmt.Sprintf("`%s`.`%s` as `%s`", c.table.Alias(), c.spec.Name(), c.alias)
	}
	return fmt.Sprintf("`%s`.`%s`", c.table.Alias(), c.spec.Name())
}

// Name implementation of STableField for IQueryField
func (c *STableField) Name() string {
	if len(c.alias) > 0 {
		return c.alias
	}
	return c.spec.Name()
}

// Reference implementation of STableField for IQueryField
func (c *STableField) Reference() string {
	return fmt.Sprintf("`%s`.`%s`", c.table.Alias(), c.Name())
}

// Label implementation of STableField for IQueryField
func (c *STableField) Label(label string) IQueryField {
	if len(label) > 0 && label != c.spec.Name() {
		c.alias = label
	}
	return c
}

// Variables implementation of STableField for IQueryField
func (c *STableField) Variables() []interface{} {
	return nil
}
