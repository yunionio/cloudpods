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
)

var defaultBackend IBackend = (*SBaseBackend)(nil)

type SBaseBackend struct{}

func (bb *SBaseBackend) Name() DBBackendName {
	return "default"
}

func (bb *SBaseBackend) GetCreateSQLs(ts ITableSpec) []string {
	return []string{}
}

func (bb *SBaseBackend) IsSupportIndexAndContraints() bool {
	return false
}

func (bb *SBaseBackend) GetTableSQL() string {
	return "SHOW TABLES"
}

func (bb *SBaseBackend) FetchTableColumnSpecs(ts ITableSpec) ([]IColumnSpec, error) {
	return nil, nil
}

func (bb *SBaseBackend) GetColumnSpecByFieldType(table *STableSpec, fieldType reflect.Type, fieldname string, tagmap map[string]string, isPointer bool) IColumnSpec {
	return nil
}

func (bb *SBaseBackend) CurrentUTCTimeStampString() string {
	return "NOW('UTC')"
}

func (bb *SBaseBackend) CurrentTimeStampString() string {
	return "NOW()"
}

func (bb *SBaseBackend) CaseInsensitiveLikeString() string {
	return "LIKE"
}

func (bb *SBaseBackend) RegexpWhereClause(cond *SRegexpConition) string {
	return tupleConditionWhereClause(&cond.STupleCondition, SQL_OP_REGEXP)
}

func (bb *SBaseBackend) UnionAllString() string {
	return "UNION ALL"
}

func (bb *SBaseBackend) UnionDistinctString() string {
	return "UNION"
}

func (bb *SBaseBackend) DropTableSQL(table string) string {
	return fmt.Sprintf("DROP TABLE `%s`", table)
}

func (bb *SBaseBackend) SupportMixedInsertVariables() bool {
	return true
}

func (bb *SBaseBackend) CanUpdate() bool {
	return false
}

func (bb *SBaseBackend) CanInsert() bool {
	return false
}

func (bb *SBaseBackend) CanInsertOrUpdate() bool {
	return false
}

func (bb *SBaseBackend) CommitTableChangeSQL(ts ITableSpec, changes STableChanges) []string {
	return nil
}

func (bb *SBaseBackend) FetchIndexesAndConstraints(ts ITableSpec) ([]STableIndex, []STableConstraint, error) {
	return nil, nil, nil
}

func (bb *SBaseBackend) DropIndexSQLTemplate() string {
	return "DROP INDEX `{{ .Index }}` ON `{{ .Table }}`"
}

func (bb *SBaseBackend) CanSupportRowAffected() bool {
	return true
}

func (bb *SBaseBackend) InsertSQLTemplate() string {
	return "INSERT INTO `{{ .Table }}` ({{ .Columns }}) VALUES ({{ .Values }})"
}

func (bb *SBaseBackend) UpdateSQLTemplate() string {
	return "UPDATE `{{ .Table }}` SET {{ .Columns }} WHERE {{ .Conditions }}"
}

func (bb *SBaseBackend) InsertOrUpdateSQLTemplate() string {
	return ""
}

func (bb *SBaseBackend) CAST(field IQueryField, typeStr string, fieldname string) IQueryField {
	return NewFunctionField(fieldname, false, `CAST(%s AS `+typeStr+`)`, field)
}

// TimestampAdd represents a SQL function TimestampAdd
func (bb *SBaseBackend) TIMESTAMPADD(name string, field IQueryField, offsetSeconds int) IQueryField {
	return NewFunctionField(name, false, `TIMESTAMPADD(SECOND, `+fmt.Sprintf("%d", offsetSeconds)+`, %s)`, field)
}

// DATE_FORMAT represents a SQL function DATE_FORMAT
func (bb *SBaseBackend) DATE_FORMAT(name string, field IQueryField, format string) IQueryField {
	return NewFunctionField(name, false, `DATE_FORMAT(%s, "`+strings.ReplaceAll(format, "%", "%%")+`")`, field)
}

// INET_ATON represents a SQL function INET_ATON
func (bb *SBaseBackend) INET_ATON(field IQueryField) IQueryField {
	return NewFunctionField("", false, `INET_ATON(%s)`, field)
}

// SubStr represents a SQL function SUBSTR
func (bb *SBaseBackend) SUBSTR(name string, field IQueryField, pos, length int) IQueryField {
	var rightStr string
	if length <= 0 {
		rightStr = fmt.Sprintf("%d)", pos)
	} else {
		rightStr = fmt.Sprintf("%d, %d)", pos, length)
	}
	return NewFunctionField(name, false, `SUBSTR(%s, `+rightStr, field)
}

// OR_Val represents a SQL function that does binary | operation on a field
func (bb *SBaseBackend) OR_Val(name string, field IQueryField, v interface{}) IQueryField {
	rightStr := fmt.Sprintf("|%v", v)
	return NewFunctionField(name, false, "%s"+rightStr, field)
}

// AND_Val represents a SQL function that does binary & operation on a field
func (bb *SBaseBackend) AND_Val(name string, field IQueryField, v interface{}) IQueryField {
	rightStr := fmt.Sprintf("&%v", v)
	return NewFunctionField(name, false, "%s"+rightStr, field)
}

// CONCAT represents a SQL function CONCAT
func (bb *SBaseBackend) CONCAT(name string, fields ...IQueryField) IQueryField {
	params := []string{}
	for i := 0; i < len(fields); i++ {
		params = append(params, "%s")
	}
	return NewFunctionField(name, false, `CONCAT(`+strings.Join(params, ",")+`)`, fields...)
}

// REPLACE represents a SQL function REPLACE
func (bb *SBaseBackend) REPLACE(name string, field IQueryField, old string, new string) IQueryField {
	return NewFunctionField(name, false, fmt.Sprintf(`REPLACE(%s, "%s", "%s")`, "%s", old, new), field)
}

// DISTINCT represents the SQL function DISTINCT
func (bb *SBaseBackend) DISTINCT(name string, field IQueryField) IQueryField {
	return NewFunctionField(name, false, "DISTINCT(%s)", field)
}

// COUNT represents the SQL function COUNT
func (bb *SBaseBackend) COUNT(name string, field ...IQueryField) IQueryField {
	var expr string
	if len(field) == 0 {
		expr = "COUNT(*)"
	} else {
		expr = "COUNT(%s)"
	}
	return NewFunctionField(name, true, expr, field...)
}

// MAX represents the SQL function MAX
func (bb *SBaseBackend) MAX(name string, field IQueryField) IQueryField {
	return NewFunctionField(name, true, "MAX(%s)", field)
}

// MIN represents the SQL function MIN
func (bb *SBaseBackend) MIN(name string, field IQueryField) IQueryField {
	return NewFunctionField(name, true, "MIN(%s)", field)
}

// SUM represents the SQL function SUM
func (bb *SBaseBackend) SUM(name string, field IQueryField) IQueryField {
	return NewFunctionField(name, true, "SUM(%s)", field)
}

// LENGTH represents SQL function LENGTH
func (bb *SBaseBackend) LENGTH(name string, field IQueryField) IQueryField {
	return NewFunctionField(name, false, "LENGTH(%s)", field)
}

func (bb *SBaseBackend) GROUP_CONCAT2(name string, sep string, field IQueryField) IQueryField {
	return NewFunctionField(name, true, fmt.Sprintf("GROUP_CONCAT(%%s SEPARATOR '%s')", sep), field)
}

// LOWER represents SQL function of LOWER
func (bb *SBaseBackend) LOWER(name string, field IQueryField) IQueryField {
	return NewFunctionField(name, false, "LOWER(%s)", field)
}

// UPPER represents SQL function of UPPER
func (bb *SBaseBackend) UPPER(name string, field IQueryField) IQueryField {
	return NewFunctionField(name, false, "UPPER(%s)", field)
}

// DATEDIFF represents SQL function of DATEDIFF
func (bb *SBaseBackend) DATEDIFF(unit string, field1, field2 IQueryField) IQueryField {
	return NewFunctionField("", false, fmt.Sprintf("DATEDIFF('%s',%s,%s)", unit, "%s", "%s"), field1, field2)
}

func (bb *SBaseBackend) QuoteChar() string {
	return "`"
}

func (bb *SBaseBackend) PrepareInsertOrUpdateSQL(ts ITableSpec, insertColNames []string, insertFields []string, onPrimaryCols []string, updateSetCols []string, insertValues []interface{}, updateValues []interface{}) (string, []interface{}) {
	return "", nil
}
