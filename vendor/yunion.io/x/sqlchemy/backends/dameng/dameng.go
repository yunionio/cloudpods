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

package dameng

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	_ "gitee.com/chunanyong/dm"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/sqlchemy"
)

func init() {
	sqlchemy.RegisterBackend(&SDamengBackend{})
}

type SDamengBackend struct {
	sqlchemy.SBaseBackend
}

func (dameng *SDamengBackend) Name() sqlchemy.DBBackendName {
	return sqlchemy.DamengBackend
}

// CanUpdate returns wether the backend supports update
func (dameng *SDamengBackend) CanUpdate() bool {
	return true
}

// CanInsert returns wether the backend supports Insert
func (dameng *SDamengBackend) CanInsert() bool {
	return true
}

func (dameng *SDamengBackend) CanSupportRowAffected() bool {
	return true
}

// CanInsertOrUpdate returns weather the backend supports InsertOrUpdate
func (dameng *SDamengBackend) CanInsertOrUpdate() bool {
	return true
}

func (dameng *SDamengBackend) InsertOrUpdateSQLTemplate() string {
	// use PrepareInsertOrUpdateSQL instead
	return ""
}

func (dameng *SDamengBackend) DropIndexSQLTemplate() string {
	return `DROP INDEX "{{ .Index }}"" ON "{{ .Table }}"`
}

func (dameng *SDamengBackend) InsertSQLTemplate() string {
	return `INSERT INTO "{{ .Table }}" ({{ .Columns }}) VALUES ({{ .Values }})`
}

func (dameng *SDamengBackend) UpdateSQLTemplate() string {
	return `UPDATE "{{ .Table }}" SET {{ .Columns }} WHERE {{ .Conditions }}`
}

func (dameng *SDamengBackend) PrepareInsertOrUpdateSQL(ts sqlchemy.ITableSpec, insertColNames []string, insertFields []string, onPrimaryCols []string, updateSetCols []string, insertValues []interface{}, updateValues []interface{}) (string, []interface{}) {
	sqlTemp := `MERGE INTO "{{ .Table }}" T1 USING (SELECT {{ .SelectValues }} FROM DUAL) T2 ON ({{ .OnConditions }}) WHEN NOT MATCHED THEN INSERT({{ .Columns }}) VALUES ({{ .Values }}) WHEN MATCHED THEN UPDATE SET {{ .SetValues }}`
	selectValues := make([]string, 0, len(insertColNames))
	onConditions := make([]string, 0, len(onPrimaryCols))

	colNameMap := make(map[string]struct{})
	for i := range insertColNames {
		colName := strings.Trim(insertColNames[i], "'\"")
		colNameMap[colName] = struct{}{}
		selectValues = append(selectValues, fmt.Sprintf("%s AS \"%s\"", insertFields[i], colName))
	}
	for _, primary := range onPrimaryCols {
		colName := strings.Trim(primary, "'\"")
		if _, ok := colNameMap[colName]; !ok {
			log.Fatalf("primary colume %s missing from insert columes for table %s", colName, ts.Name())
		}
		onConditions = append(onConditions, fmt.Sprintf("T1.%s=T2.%s", primary, primary))
	}
	for i := range updateSetCols {
		setCol := updateSetCols[i]
		equalPos := strings.Index(setCol, "=")
		key := strings.TrimSpace(setCol[0:equalPos])
		val := strings.TrimSpace(setCol[equalPos+1:])
		tkey := fmt.Sprintf("\"T1\".%s", key)
		val = strings.ReplaceAll(val, key, tkey)
		updateSetCols[i] = fmt.Sprintf("%s = %s", tkey, val)
	}
	values := make([]interface{}, 0, len(insertValues)*2+len(updateValues))
	values = append(values, insertValues...)
	values = append(values, insertValues...)
	values = append(values, updateValues...)
	sql := sqlchemy.TemplateEval(sqlTemp, struct {
		Table        string
		SelectValues string
		OnConditions string
		Columns      string
		Values       string
		SetValues    string
	}{
		Table:        ts.Name(),
		SelectValues: strings.Join(selectValues, ", "),
		OnConditions: strings.Join(onConditions, " AND "),
		Columns:      strings.Join(insertColNames, ", "),
		Values:       strings.Join(insertFields, ", "),
		SetValues:    strings.Join(updateSetCols, ", "),
	})
	return sql, values
}

func (dameng *SDamengBackend) GetTableSQL() string {
	return `SELECT table_name AS "name" FROM user_tables`
}

func (dameng *SDamengBackend) CurrentUTCTimeStampString() string {
	return "GETUTCDATE()"
}

func (dameng *SDamengBackend) CurrentTimeStampString() string {
	return "GETDATE()"
}

func (dameng *SDamengBackend) GetCreateSQLs(ts sqlchemy.ITableSpec) []string {
	cols := make([]string, 0)
	primaries := make([]string, 0)
	for _, c := range ts.Columns() {
		cols = append(cols, c.DefinitionString())
		if c.IsPrimary() {
			primaries = append(primaries, fmt.Sprintf(`"%s"`, c.Name()))
		}
	}
	if len(primaries) > 0 {
		cols = append(cols, fmt.Sprintf("NOT CLUSTER PRIMARY KEY (%s)", strings.Join(primaries, ", ")))
	}
	sqls := []string{
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" (%s);`, ts.Name(), strings.Join(cols, ", ")),
	}
	for _, idx := range ts.Indexes() {
		sqls = append(sqls, createIndexSQL(ts, idx))
	}
	return sqls
}

func (msyql *SDamengBackend) IsSupportIndexAndContraints() bool {
	return true
}

func (dameng *SDamengBackend) FetchTableColumnSpecs(ts sqlchemy.ITableSpec) ([]sqlchemy.IColumnSpec, error) {
	infos, err := fetchTableColInfo(ts)
	if err != nil {
		return nil, errors.Wrap(err, "fetchTableColInfo")
	}
	specs := make([]sqlchemy.IColumnSpec, 0)
	for _, info := range infos {
		specs = append(specs, info.toColumnSpec())
	}
	return specs, nil
}

func (dameng *SDamengBackend) FetchIndexesAndConstraints(ts sqlchemy.ITableSpec) ([]sqlchemy.STableIndex, []sqlchemy.STableConstraint, error) {
	indexes, err := fetchTableIndexes(ts)
	if err != nil {
		return nil, nil, errors.Wrap(err, "fetchTableIndexes")
	}
	retIdxes := make([]sqlchemy.STableIndex, 0)
	for k := range indexes {
		if indexes[k].isPrimary {
			continue
		}
		retIdxes = append(retIdxes, sqlchemy.NewTableIndex(ts, indexes[k].indexName, indexes[k].colnames, false))
	}
	return retIdxes, nil, nil
}

func getTextSqlType(tagmap map[string]string) (string, map[string]string) {
	var width int
	var sqltype string
	tagmap, widthStr, _ := utils.TagPop(tagmap, sqlchemy.TAG_WIDTH)
	// widthStr := tagmap[sqlchemy.TAG_WIDTH]
	if len(widthStr) > 0 && regutils.MatchInteger(widthStr) {
		width, _ = strconv.Atoi(widthStr)
	}
	if width == 0 || width > 975 {
		sqltype = "TEXT"
	} else {
		tagmap[sqlchemy.TAG_WIDTH] = widthStr
		sqltype = "VARCHAR"
	}
	return sqltype, tagmap
}

func (dameng *SDamengBackend) GetColumnSpecByFieldType(table *sqlchemy.STableSpec, fieldType reflect.Type, fieldname string, tagmap map[string]string, isPointer bool) sqlchemy.IColumnSpec {
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
		sqltype, tagmap := getTextSqlType(tagmap)
		col := NewTextColumn(fieldname, sqltype, tagmap, isPointer)
		return &col
	case reflect.Int, reflect.Int32:
		col := NewIntegerColumn(fieldname, "INT", tagmap, isPointer)
		return &col
	case reflect.Int8:
		col := NewIntegerColumn(fieldname, "TINYINT", tagmap, isPointer)
		return &col
	case reflect.Int16:
		col := NewIntegerColumn(fieldname, "SMALLINT", tagmap, isPointer)
		return &col
	case reflect.Int64:
		col := NewIntegerColumn(fieldname, "BIGINT", tagmap, isPointer)
		return &col
	case reflect.Uint, reflect.Uint32:
		col := NewIntegerColumn(fieldname, "INT", tagmap, isPointer)
		return &col
	case reflect.Uint8:
		col := NewIntegerColumn(fieldname, "TINYINT", tagmap, isPointer)
		return &col
	case reflect.Uint16:
		col := NewIntegerColumn(fieldname, "SMALLINT", tagmap, isPointer)
		return &col
	case reflect.Uint64:
		col := NewIntegerColumn(fieldname, "BIGINT", tagmap, isPointer)
		return &col
	case reflect.Bool:
		col := NewBooleanColumn(fieldname, tagmap, isPointer)
		return &col
	case reflect.Float32, reflect.Float64:
		if _, ok := tagmap[sqlchemy.TAG_WIDTH]; ok {
			col := NewDecimalColumn(fieldname, tagmap, isPointer)
			return &col
		}
		colType := "REAL"
		if fieldType == gotypes.Float64Type {
			colType = "DOUBLE"
		}
		col := NewFloatColumn(fieldname, colType, tagmap, isPointer)
		return &col
	case reflect.Map, reflect.Slice:
		sqltype, tagmap := getTextSqlType(tagmap)
		col := NewCompoundColumn(fieldname, sqltype, tagmap, isPointer)
		return &col
	}
	if fieldType.Implements(gotypes.ISerializableType) {
		sqltype, tagmap := getTextSqlType(tagmap)
		col := NewCompoundColumn(fieldname, sqltype, tagmap, isPointer)
		return &col
	}
	return nil
}

func (dameng *SDamengBackend) QuoteChar() string {
	return "\""
}
