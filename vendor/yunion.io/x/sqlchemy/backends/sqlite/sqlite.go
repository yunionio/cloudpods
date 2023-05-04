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

package sqlite

import (
	"fmt"
	"reflect"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"

	"yunion.io/x/sqlchemy"
)

func init() {
	sqlchemy.RegisterBackend(&SSqliteBackend{})
}

type SSqliteBackend struct {
	sqlchemy.SBaseBackend
}

func (sqlite *SSqliteBackend) Name() sqlchemy.DBBackendName {
	return sqlchemy.SQLiteBackend
}

// CanUpdate returns wether the backend supports update
func (sqlite *SSqliteBackend) CanUpdate() bool {
	return true
}

// CanInsert returns wether the backend supports Insert
func (sqlite *SSqliteBackend) CanInsert() bool {
	return true
}

// CanInsertOrUpdate returns weather the backend supports InsertOrUpdate
func (sqlite *SSqliteBackend) CanInsertOrUpdate() bool {
	return true
}

func (sqlite *SSqliteBackend) CurrentUTCTimeStampString() string {
	return "DATETIME('now')"
}

func (sqlite *SSqliteBackend) CurrentTimeStampString() string {
	return "DATETIME('now', 'localtime')"
}

func (sqlite *SSqliteBackend) DropIndexSQLTemplate() string {
	return "DROP INDEX IF EXISTS `{{ .Table }}`.`{{ .Index }}`"
}

func (sqlite *SSqliteBackend) InsertOrUpdateSQLTemplate() string {
	return "INSERT INTO `{{ .Table }}` ({{ .Columns }}) VALUES ({{ .Values }}) ON CONFLICT({{ .PrimaryKeys }}) DO UPDATE SET {{ .SetValues }}"
}

func (sqlite *SSqliteBackend) GetTableSQL() string {
	return "SELECT name FROM sqlite_master WHERE type='table'"
}

func (sqlite *SSqliteBackend) IsSupportIndexAndContraints() bool {
	return true
}

func (sqlite *SSqliteBackend) GetCreateSQLs(ts sqlchemy.ITableSpec) []string {
	cols := make([]string, 0)
	primaries := make([]string, 0)
	for _, c := range ts.Columns() {
		cols = append(cols, c.DefinitionString())
		if c.IsPrimary() && !c.IsAutoIncrement() {
			primaries = append(primaries, fmt.Sprintf("`%s`", c.Name()))
		}
	}
	if len(primaries) > 0 {
		cols = append(cols, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaries, ", ")))
	}
	ret := []string{
		"PRAGMA encoding=\"UTF-8\"",
		fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (\n%s\n)", ts.Name(), strings.Join(cols, ",\n")),
	}
	for _, idx := range ts.Indexes() {
		ret = append(ret, createIndexSQL(ts, idx))
	}
	return ret
}

func (sqlite *SSqliteBackend) FetchIndexesAndConstraints(ts sqlchemy.ITableSpec) ([]sqlchemy.STableIndex, []sqlchemy.STableConstraint, error) {
	sql := fmt.Sprintf("SELECT `name`, `sql` FROM `sqlite_master` WHERE `tbl_name`='%s' AND `type`='index' AND `sql`!=''", ts.Name())
	query := ts.Database().NewRawQuery(sql, "name", "sql")
	results := make([]sSqliteTableInfo, 0)
	err := query.All(&results)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Raw Query Scan %s", sql)
	}
	indexes := make([]sqlchemy.STableIndex, 0)
	for i := range results {
		ti, err := results[i].parseTableIndex(ts)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "parseTableIndex fail %s", results[i].Sql)
		}
		indexes = append(indexes, ti)
	}
	return indexes, nil, nil
}

func (sqlite *SSqliteBackend) FetchTableColumnSpecs(ts sqlchemy.ITableSpec) ([]sqlchemy.IColumnSpec, error) {
	sql := fmt.Sprintf("PRAGMA table_info(`%s`);", ts.Name())
	query := ts.Database().NewRawQuery(sql, "cid", "name", "type", "notnull", "dflt_value", "pk")
	infos := make([]sSqlColumnInfo, 0)
	err := query.All(&infos)
	if err != nil {
		return nil, err
	}
	specs := make([]sqlchemy.IColumnSpec, 0)
	// find out integer primary key
	var primaryCol sqlchemy.IColumnSpec
	var primaryCount int
	for _, info := range infos {
		spec := info.toColumnSpec()
		if spec.IsPrimary() {
			primaryCol = spec
			primaryCount++
		}
		specs = append(specs, spec)
	}
	if primaryCount == 1 {
		if intc, ok := primaryCol.(*SIntegerColumn); ok {
			intc.isAutoIncrement = true
		}
	}
	return specs, nil
}

func (sqlite *SSqliteBackend) GetColumnSpecByFieldType(table *sqlchemy.STableSpec, fieldType reflect.Type, fieldname string, tagmap map[string]string, isPointer bool) sqlchemy.IColumnSpec {
	switch fieldType {
	case tristate.TriStateType:
		col := NewTristateColumn(table.Name(), fieldname, tagmap, isPointer)
		return &col
	case gotypes.TimeType:
		col := NewDateTimeColumn(fieldname, tagmap, isPointer)
		return &col
	}
	switch fieldType.Kind() {
	case reflect.String:
		col := NewTextColumn(fieldname, tagmap, isPointer)
		return &col
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		col := NewIntegerColumn(fieldname, tagmap, isPointer)
		return &col
	case reflect.Bool:
		col := NewBooleanColumn(fieldname, tagmap, isPointer)
		return &col
	case reflect.Float32, reflect.Float64:
		col := NewFloatColumn(fieldname, tagmap, isPointer)
		return &col
	case reflect.Map, reflect.Slice:
		col := NewCompoundColumn(fieldname, tagmap, isPointer)
		return &col
	}
	if fieldType.Implements(gotypes.ISerializableType) {
		col := NewCompoundColumn(fieldname, tagmap, isPointer)
		return &col
	}
	return nil
}
