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

package clickhouse

import (
	"fmt"
	"reflect"
	"strings"

	_ "github.com/ClickHouse/clickhouse-go"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/sqlchemy"
)

func init() {
	sqlchemy.RegisterBackend(&SClickhouseBackend{})
}

type SClickhouseBackend struct {
	sqlchemy.SBaseBackend
}

func (click *SClickhouseBackend) Name() sqlchemy.DBBackendName {
	return sqlchemy.ClickhouseBackend
}

func (click *SClickhouseBackend) CaseInsensitiveLikeString() string {
	return "ILIKE"
}

// CanUpdate returns wether the backend supports update
func (click *SClickhouseBackend) CanUpdate() bool {
	return true
}

// CanInsert returns wether the backend supports Insert
func (click *SClickhouseBackend) CanInsert() bool {
	return true
}

// CanInsertOrUpdate returns weather the backend supports InsertOrUpdate
func (click *SClickhouseBackend) CanInsertOrUpdate() bool {
	return false
}

func (click *SClickhouseBackend) IsSupportIndexAndContraints() bool {
	return false
}

func (click *SClickhouseBackend) CanSupportRowAffected() bool {
	return false
}

func (click *SClickhouseBackend) CurrentUTCTimeStampString() string {
	return "NOW('UTC')"
}

func (click *SClickhouseBackend) CurrentTimeStampString() string {
	return "NOW()"
}

func (click *SClickhouseBackend) UnionAllString() string {
	return "UNION ALL"
}

func (click *SClickhouseBackend) UnionDistinctString() string {
	return "UNION DISTINCT"
}

func (click *SClickhouseBackend) SupportMixedInsertVariables() bool {
	return false
}

func (click *SClickhouseBackend) UpdateSQLTemplate() string {
	return "ALTER TABLE `{{ .Table }}` UPDATE {{ .Columns }} WHERE {{ .Conditions }}"
}

func (click *SClickhouseBackend) GetCreateSQLs(ts sqlchemy.ITableSpec) []string {
	cols := make([]string, 0)
	primaries := make([]string, 0)
	orderbys := make([]string, 0)
	partitions := make([]string, 0)
	var ttlCol IClickhouseColumnSpec
	for _, c := range ts.Columns() {
		cols = append(cols, c.DefinitionString())
		if c.IsPrimary() {
			primaries = append(primaries, fmt.Sprintf("`%s`", c.Name()))
		}
		if cc, ok := c.(IClickhouseColumnSpec); ok {
			if cc.IsOrderBy() {
				orderbys = append(orderbys, fmt.Sprintf("`%s`", c.Name()))
			}
			partition := cc.PartitionBy()
			if len(partition) > 0 && !utils.IsInStringArray(partition, partitions) {
				partitions = append(partitions, partition)
			}
			ttlC, ttlU := cc.GetTTL()
			if ttlC > 0 && len(ttlU) > 0 {
				ttlCol = cc
			}
		}
	}
	createSql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (\n%s\n) ENGINE MergeTree", ts.Name(), strings.Join(cols, ",\n"))
	if len(orderbys) == 0 {
		orderbys = primaries
	}
	if len(partitions) > 0 {
		createSql += fmt.Sprintf("\nPARTITION BY (%s)", strings.Join(partitions, ", "))
	}
	if len(primaries) > 0 {
		createSql += fmt.Sprintf("\nPRIMARY KEY (%s)", strings.Join(primaries, ", "))
		newOrderBys := make([]string, len(primaries))
		copy(newOrderBys, primaries)
		for _, f := range orderbys {
			if !utils.IsInStringArray(f, newOrderBys) {
				newOrderBys = append(newOrderBys, f)
			}
		}
		orderbys = newOrderBys
	}
	if len(orderbys) > 0 {
		createSql += fmt.Sprintf("\nORDER BY (%s)", strings.Join(orderbys, ", "))
	} else {
		createSql += fmt.Sprintf("\nORDER BY tuple()")
	}
	if ttlCol != nil {
		ttlCount, ttlUnit := ttlCol.GetTTL()
		createSql += fmt.Sprintf("\nTTL `%s` + INTERVAL %d %s", ttlCol.Name(), ttlCount, ttlUnit)
	}
	// set default time zone of table to UTC
	createSql += "\nSETTINGS index_granularity=8192"
	return []string{
		createSql,
	}
}

func (click *SClickhouseBackend) FetchTableColumnSpecs(ts sqlchemy.ITableSpec) ([]sqlchemy.IColumnSpec, error) {
	sql := fmt.Sprintf("DESCRIBE `%s`", ts.Name())
	query := ts.Database().NewRawQuery(sql, "name", "type", "default_type", "default_expression", "comment", "codec_expression", "ttl_expression")
	infos := make([]sSqlColumnInfo, 0)
	err := query.All(&infos)
	if err != nil {
		return nil, errors.Wrap(err, "describe table")
	}
	specs := make([]sqlchemy.IColumnSpec, 0)
	for _, info := range infos {
		spec := info.toColumnSpec()
		specs = append(specs, spec)
	}

	sql = fmt.Sprintf("SHOW CREATE TABLE `%s`", ts.Name())
	query = ts.Database().NewRawQuery(sql, "statement")
	row := query.Row()
	var defStr string
	err = row.Scan(&defStr)
	if err != nil {
		return nil, errors.Wrap(err, "show create table")
	}
	primaries, orderbys, partitions, ttl := parseCreateTable(defStr)
	var ttlCfg sColumnTTL
	if len(ttl) > 0 {
		ttlCfg, err = parseTTLExpression(ttl)
		if err != nil {
			return nil, errors.Wrap(err, "parseTTLExpression")
		}
	}
	for _, spec := range specs {
		if utils.IsInStringArray(spec.Name(), primaries) {
			spec.SetPrimary(true)
		}
		if clickSpec, ok := spec.(IClickhouseColumnSpec); ok {
			if utils.IsInStringArray(clickSpec.Name(), orderbys) {
				clickSpec.SetOrderBy(true)
			}
			for _, part := range partitions {
				if stringutils.ContainsWord(part, clickSpec.Name()) {
					clickSpec.SetPartitionBy(part)
				}
			}
			if ttlCfg.ColName == clickSpec.Name() {
				clickSpec.SetTTL(ttlCfg.Count, ttlCfg.Unit)
			}
		}
	}

	return specs, nil
}

func (click *SClickhouseBackend) GetColumnSpecByFieldType(table *sqlchemy.STableSpec, fieldType reflect.Type, fieldname string, tagmap map[string]string, isPointer bool) sqlchemy.IColumnSpec {
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
		col := NewTextColumn(fieldname, "String", tagmap, isPointer)
		return &col
	case reflect.Int, reflect.Int32:
		col := NewIntegerColumn(fieldname, "Int32", tagmap, isPointer)
		return &col
	case reflect.Int8:
		col := NewIntegerColumn(fieldname, "Int8", tagmap, isPointer)
		return &col
	case reflect.Int16:
		col := NewIntegerColumn(fieldname, "Int16", tagmap, isPointer)
		return &col
	case reflect.Int64:
		col := NewIntegerColumn(fieldname, "Int64", tagmap, isPointer)
		return &col
	case reflect.Uint, reflect.Uint32:
		col := NewIntegerColumn(fieldname, "UInt32", tagmap, isPointer)
		return &col
	case reflect.Uint8:
		col := NewIntegerColumn(fieldname, "UInt8", tagmap, isPointer)
		return &col
	case reflect.Uint16:
		col := NewIntegerColumn(fieldname, "UInt16", tagmap, isPointer)
		return &col
	case reflect.Uint64:
		col := NewIntegerColumn(fieldname, "UInt64", tagmap, isPointer)
		return &col
	case reflect.Bool:
		col := NewBooleanColumn(fieldname, tagmap, isPointer)
		return &col
	case reflect.Float32:
		if _, ok := tagmap[sqlchemy.TAG_WIDTH]; ok {
			col := NewDecimalColumn(fieldname, tagmap, isPointer)
			return &col
		}
		col := NewFloatColumn(fieldname, "Float32", tagmap, isPointer)
		return &col
	case reflect.Float64:
		if _, ok := tagmap[sqlchemy.TAG_WIDTH]; ok {
			col := NewDecimalColumn(fieldname, tagmap, isPointer)
			return &col
		}
		col := NewFloatColumn(fieldname, "Float64", tagmap, isPointer)
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
