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

package mysql

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/sqlchemy"
)

func init() {
	sqlchemy.RegisterBackend(&SMySQLBackend{})
}

type SMySQLBackend struct {
	sqlchemy.SBaseBackend
}

func (mysql *SMySQLBackend) Name() sqlchemy.DBBackendName {
	return sqlchemy.MySQLBackend
}

// CanUpdate returns wether the backend supports update
func (mysql *SMySQLBackend) CanUpdate() bool {
	return true
}

// CanInsert returns wether the backend supports Insert
func (mysql *SMySQLBackend) CanInsert() bool {
	return true
}

// CanInsertOrUpdate returns weather the backend supports InsertOrUpdate
func (mysql *SMySQLBackend) CanInsertOrUpdate() bool {
	return true
}

func (mysql *SMySQLBackend) InsertOrUpdateSQLTemplate() string {
	return "INSERT INTO `{{ .Table }}` ({{ .Columns }}) VALUES ({{ .Values }}) ON DUPLICATE KEY UPDATE {{ .SetValues }}"
}

func (mysql *SMySQLBackend) CurrentUTCTimeStampString() string {
	return "UTC_TIMESTAMP()"
}

func (mysql *SMySQLBackend) CurrentTimeStampString() string {
	return "NOW()"
}

func (mysql *SMySQLBackend) GetCreateSQLs(ts sqlchemy.ITableSpec) []string {
	cols := make([]string, 0)
	primaries := make([]string, 0)
	autoInc := ""
	for _, c := range ts.Columns() {
		cols = append(cols, c.DefinitionString())
		if c.IsPrimary() {
			primaries = append(primaries, fmt.Sprintf("`%s`", c.Name()))
			if intC, ok := c.(*SIntegerColumn); ok && intC.autoIncrementOffset > 0 {
				autoInc = fmt.Sprintf(" AUTO_INCREMENT=%d", intC.autoIncrementOffset)
			}
		}
	}
	if len(primaries) > 0 {
		cols = append(cols, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaries, ", ")))
	}
	sqls := []string{
		fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (\n%s\n) ENGINE=InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci%s", ts.Name(), strings.Join(cols, ",\n"), autoInc),
	}
	for _, idx := range ts.Indexes() {
		sqls = append(sqls, createIndexSQL(ts, idx))
	}
	return sqls
}

func (msyql *SMySQLBackend) IsSupportIndexAndContraints() bool {
	return true
}

func (mysql *SMySQLBackend) FetchTableColumnSpecs(ts sqlchemy.ITableSpec) ([]sqlchemy.IColumnSpec, error) {
	sql := fmt.Sprintf("SHOW FULL COLUMNS IN `%s`", ts.Name())
	query := ts.Database().NewRawQuery(sql, "field", "type", "collation", "null", "key", "default", "extra", "privileges", "comment")
	infos := make([]sSqlColumnInfo, 0)
	err := query.All(&infos)
	if err != nil {
		return nil, err
	}
	specs := make([]sqlchemy.IColumnSpec, 0)
	for _, info := range infos {
		specs = append(specs, info.toColumnSpec())
	}
	return specs, nil
}

func (mysql *SMySQLBackend) FetchIndexesAndConstraints(ts sqlchemy.ITableSpec) ([]sqlchemy.STableIndex, []sqlchemy.STableConstraint, error) {
	sql := fmt.Sprintf("SHOW CREATE TABLE `%s`", ts.Name())
	query := ts.Database().NewRawQuery(sql, "table", "create table")
	row := query.Row()
	var name, defStr string
	err := row.Scan(&name, &defStr)
	if err != nil {
		if isMysqlError(err, mysqlErrorTableNotExist) {
			err = sqlchemy.ErrTableNotExists
		}
		return nil, nil, err
	}
	indexes := parseIndexes(ts, defStr)
	constraints := parseConstraints(defStr)
	return indexes, constraints, nil
}

func getTextSqlType(tagmap map[string]string) string {
	var width int
	var sqltype string
	widthStr, _ := tagmap[sqlchemy.TAG_WIDTH]
	if len(widthStr) > 0 && regutils.MatchInteger(widthStr) {
		width, _ = strconv.Atoi(widthStr)
	}
	txtLen, _ := tagmap[sqlchemy.TAG_TEXT_LENGTH]
	if width == 0 {
		switch strings.ToLower(txtLen) {
		case "medium":
			sqltype = "MEDIUMTEXT"
		case "long":
			sqltype = "LONGTEXT"
		default:
			sqltype = "TEXT"
		}
	} else {
		sqltype = "VARCHAR"
	}
	return sqltype
}

func (mysql *SMySQLBackend) GetColumnSpecByFieldType(table *sqlchemy.STableSpec, fieldType reflect.Type, fieldname string, tagmap map[string]string, isPointer bool) sqlchemy.IColumnSpec {
	switch fieldType {
	case tristate.TriStateType:
		tagmap[sqlchemy.TAG_WIDTH] = "1"
		col := NewTristateColumn(table.Name(), fieldname, tagmap, isPointer)
		return &col
	case gotypes.TimeType:
		col := NewDateTimeColumn(fieldname, tagmap, isPointer)
		return &col
	}
	switch fieldType.Kind() {
	case reflect.String:
		col := NewTextColumn(fieldname, getTextSqlType(tagmap), tagmap, isPointer)
		return &col
	case reflect.Int, reflect.Int32:
		tagmap[sqlchemy.TAG_WIDTH] = intWidthString("INT")
		col := NewIntegerColumn(fieldname, "INT", false, tagmap, isPointer)
		return &col
	case reflect.Int8:
		tagmap[sqlchemy.TAG_WIDTH] = intWidthString("TINYINT")
		col := NewIntegerColumn(fieldname, "TINYINT", false, tagmap, isPointer)
		return &col
	case reflect.Int16:
		tagmap[sqlchemy.TAG_WIDTH] = intWidthString("SMALLINT")
		col := NewIntegerColumn(fieldname, "SMALLINT", false, tagmap, isPointer)
		return &col
	case reflect.Int64:
		tagmap[sqlchemy.TAG_WIDTH] = intWidthString("BIGINT")
		col := NewIntegerColumn(fieldname, "BIGINT", false, tagmap, isPointer)
		return &col
	case reflect.Uint, reflect.Uint32:
		tagmap[sqlchemy.TAG_WIDTH] = uintWidthString("INT")
		col := NewIntegerColumn(fieldname, "INT", true, tagmap, isPointer)
		return &col
	case reflect.Uint8:
		tagmap[sqlchemy.TAG_WIDTH] = uintWidthString("TINYINT")
		col := NewIntegerColumn(fieldname, "TINYINT", true, tagmap, isPointer)
		return &col
	case reflect.Uint16:
		tagmap[sqlchemy.TAG_WIDTH] = uintWidthString("SMALLINT")
		col := NewIntegerColumn(fieldname, "SMALLINT", true, tagmap, isPointer)
		return &col
	case reflect.Uint64:
		tagmap[sqlchemy.TAG_WIDTH] = uintWidthString("BIGINT")
		col := NewIntegerColumn(fieldname, "BIGINT", true, tagmap, isPointer)
		return &col
	case reflect.Bool:
		tagmap[sqlchemy.TAG_WIDTH] = "1"
		col := NewBooleanColumn(fieldname, tagmap, isPointer)
		return &col
	case reflect.Float32, reflect.Float64:
		if _, ok := tagmap[sqlchemy.TAG_WIDTH]; ok {
			col := NewDecimalColumn(fieldname, tagmap, isPointer)
			return &col
		}
		colType := "FLOAT"
		if fieldType == gotypes.Float64Type {
			colType = "DOUBLE"
		}
		col := NewFloatColumn(fieldname, colType, tagmap, isPointer)
		return &col
	case reflect.Map, reflect.Slice:
		col := NewCompoundColumn(fieldname, getTextSqlType(tagmap), tagmap, isPointer)
		return &col
	}
	if fieldType.Implements(gotypes.ISerializableType) {
		col := NewCompoundColumn(fieldname, getTextSqlType(tagmap), tagmap, isPointer)
		return &col
	}
	return nil
}
