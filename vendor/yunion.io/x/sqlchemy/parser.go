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

	"yunion.io/x/pkg/util/reflectutils"
)

func (table *STableSpec) structField2ColumnSpec(field *reflectutils.SStructFieldValue) IColumnSpec {
	fieldname := field.Info.MarshalName()
	tagmap := field.Info.Tags
	if _, ok := tagmap[TAG_IGNORE]; ok {
		return nil
	}
	db := table.Database()
	if db == nil {
		panic("structField2ColumnSpec: empty database")
	}
	if db.backend == nil {
		panic("structField2ColumnSpec: empty backend")
	}
	fieldType := field.Value.Type()
	var retCol = db.backend.GetColumnSpecByFieldType(table, fieldType, fieldname, tagmap, false)
	if retCol == nil && fieldType.Kind() == reflect.Ptr {
		retCol = db.backend.GetColumnSpecByFieldType(table, fieldType.Elem(), fieldname, tagmap, true)
	}
	if retCol == nil {
		panic(fmt.Sprintf("unsupported colume %s data type %s", fieldname, fieldType.Name()))
	}
	return retCol
}

func (table *STableSpec) struct2TableSpec(sv reflect.Value) {
	fields := reflectutils.FetchStructFieldValueSet(sv)
	autoIncCnt := 0
	tmpCols := make([]IColumnSpec, 0)
	for i := 0; i < len(fields); i++ {
		column := table.structField2ColumnSpec(&fields[i])
		if column != nil {
			if column.IsAutoIncrement() {
				autoIncCnt++
				if autoIncCnt > 1 {
					panic(fmt.Sprintf("Table %s contains multiple autoincremental columns!!", table.name))
				}
			}
			if column.IsIndex() {
				table.AddIndex(column.IsUnique(), column.Name())
			}
			tmpCols = append(tmpCols, column)
		}
	}
	// make column assignment atomic
	table._columns = tmpCols
}
