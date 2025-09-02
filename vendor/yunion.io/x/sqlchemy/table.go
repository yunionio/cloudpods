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
	"sort"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
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

	// Indexes
	Indexes() []STableIndex

	// Expression returns expression of the table
	Expression() string

	// Instance returns an instance of STable for this spec
	Instance() *STable

	// DropForeignKeySQL returns the SQL statements to drop foreignkeys for this table
	DropForeignKeySQL() []string

	// AddIndex adds index to table
	AddIndex(unique bool, cols ...string) bool

	// SyncSQL returns SQL strings to synchronize the data and model definition of the table
	SyncSQL() []string

	// Sync forces synchronize the data and model definition of the table
	Sync() error

	// Fetch query a struct
	Fetch(dt interface{}) error

	// Database returns the database of this table
	Database() *SDatabase

	// Drop drops table
	Drop() error
}

// STableSpec defines the table specification, which implements ITableSpec
type STableSpec struct {
	structType  reflect.Type
	name        string
	_columns    []IColumnSpec
	_indexes    []STableIndex
	_contraints []STableConstraint

	sDBReferer
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
	return NewTableSpecFromStructWithDBName(s, name, DefaultDB)
}

func NewTableSpecFromStructWithDBName(s interface{}, name string, dbName DBName) *STableSpec {
	val := reflect.Indirect(reflect.ValueOf(s))
	st := val.Type()
	if st.Kind() != reflect.Struct {
		panic("expect Struct kind")
	}
	table := &STableSpec{
		name:       name,
		structType: st,
		sDBReferer: sDBReferer{
			dbName: dbName,
		},
	}
	// table.struct2TableSpec(val)
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

func (ts *STableSpec) SyncColumnIndexes() error {
	if !ts.Exists() {
		return errors.Wrap(errors.ErrNotFound, "table not exists")
	}

	cols, err := ts.Database().backend.FetchTableColumnSpecs(ts)
	if err != nil {
		log.Errorf("fetchColumnDefs fail: %s", err)
		return errors.Wrap(err, "FetchTableColumnSpecs")
	}
	if len(cols) != len(ts._columns) {
		return errors.Wrapf(errors.ErrInvalidStatus, "ts col %d != actual col %d", len(ts._columns), len(cols))
	}
	for i := range cols {
		cols[i].SetColIndex(i)
	}
	// sort colums
	sort.Slice(cols, func(i, j int) bool {
		return compareColumnSpec(cols[i], cols[j]) < 0
	})
	sort.Slice(ts._columns, func(i, j int) bool {
		return compareColumnSpec(ts._columns[i], ts._columns[j]) < 0
	})
	// compare columns and assign colindex
	for i := range ts._columns {
		comp := compareColumnSpec(cols[i], ts._columns[i])
		if comp != 0 {
			return errors.Wrapf(errors.ErrInvalidStatus, "colname %s != %s", cols[i].Name(), ts._columns[i].Name())
		}
		ts._columns[i].SetColIndex(cols[i].GetColIndex())
	}

	// sort columns according to colindex
	sort.Slice(ts._columns, func(i, j int) bool {
		return compareColumnIndex(ts._columns[i], ts._columns[j]) < 0
	})

	return nil
}

// Clone makes a clone of a table, so we may create a new table of the same schema
func (ts *STableSpec) Clone(name string, autoIncOffset int64) *STableSpec {
	nts, _ := ts.CloneWithSyncColumnOrder(name, autoIncOffset, false)
	return nts
}

// Clone makes a clone of a table, so we may create a new table of the same schema
func (ts *STableSpec) CloneWithSyncColumnOrder(name string, autoIncOffset int64, syncColOrder bool) (*STableSpec, error) {
	if ts.Exists() && syncColOrder {
		// if table exists, sync column index
		err := ts.SyncColumnIndexes()
		if err != nil {
			return nil, errors.Wrap(err, "SyncColumnIndexes")
		}
	}
	columns := ts.Columns()
	newCols := make([]IColumnSpec, len(columns))
	for i := range newCols {
		col := columns[i]
		if col.IsAutoIncrement() {
			colValue := reflect.Indirect(reflect.ValueOf(col))
			newColValue := reflect.Indirect(reflect.New(colValue.Type()))
			newColValue.Set(colValue)
			newCol := newColValue.Addr().Interface().(IColumnSpec)
			newCol.SetAutoIncrementOffset(autoIncOffset)
			newCols[i] = newCol
		} else {
			newCols[i] = col
		}
	}
	nts := &STableSpec{
		structType:  ts.structType,
		name:        name,
		_columns:    newCols,
		_contraints: ts._contraints,
		sDBReferer:  ts.sDBReferer,
	}
	newIndexes := make([]STableIndex, len(ts._indexes))
	for i := range ts._indexes {
		newIndexes[i] = ts._indexes[i].clone(nts)
	}
	nts._indexes = newIndexes
	return nts, nil
}

// Columns implementation of STableSpec for ITableSpec
func (ts *STableSpec) Columns() []IColumnSpec {
	if ts._columns == nil {
		val := reflect.Indirect(reflect.New(ts.structType))
		ts.struct2TableSpec(val)
	}
	return ts._columns
}

// PrimaryColumns implementation of STableSpec for ITableSpec
func (ts *STableSpec) PrimaryColumns() []IColumnSpec {
	ret := make([]IColumnSpec, 0)
	columns := ts.Columns()
	for i := range columns {
		if columns[i].IsPrimary() {
			ret = append(ret, columns[i])
		}
	}
	return ret
}

// Indexes implementation of STableSpec for ITableSpec
func (ts *STableSpec) Indexes() []STableIndex {
	return ts._indexes
}

// DataType implementation of STableSpec for ITableSpec
func (ts *STableSpec) DataType() reflect.Type {
	return ts.structType
}

// CreateSQL returns the SQL for creating this table
func (ts *STableSpec) CreateSQLs() []string {
	return ts.Database().backend.GetCreateSQLs(ts)
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

// Database implementaion of STable for IQuerySource
func (tbl *STable) database() *SDatabase {
	return tbl.spec.Database()
}

// Expression implementation of STable for IQuerySource
func (tbl *STable) Expression() string {
	return tbl.spec.Expression()
}

// Alias implementation of STable for IQuerySource
func (tbl *STable) Alias() string {
	return tbl.alias
}

// Variables implementation of STable for IQuerySource
func (tbl *STable) Variables() []interface{} {
	return []interface{}{}
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
	if len(label) > 0 {
		c.alias = label
	}
	return c
}

// Variables implementation of STableField for IQueryField
func (c *STableField) Variables() []interface{} {
	return nil
}

// database implementation of STableField for IQueryField
func (c *STableField) database() *SDatabase {
	return c.table.database()
}
