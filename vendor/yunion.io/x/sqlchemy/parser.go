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

	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/reflectutils"
)

func structField2ColumnSpec(field *reflectutils.SStructFieldValue) IColumnSpec {
	fieldname := field.Info.MarshalName()
	tagmap := field.Info.Tags
	if _, ok := tagmap[TAG_IGNORE]; ok {
		return nil
	}
	fieldType := field.Value.Type()
	var retCol = getFiledTypeCol(fieldType, fieldname, tagmap, false)
	if retCol == nil && fieldType.Kind() == reflect.Ptr {
		retCol = getFiledTypeCol(fieldType.Elem(), fieldname, tagmap, true)
	}
	if retCol == nil {
		panic(fmt.Sprintf("unsupported colume %s data type %s", fieldname, fieldType.Name()))
	}
	return retCol
}

func getFiledTypeCol(fieldType reflect.Type, fieldname string, tagmap map[string]string, isPointer bool) IColumnSpec {
	switch fieldType {
	case tristate.TriStateType:
		tagmap[TAG_WIDTH] = "1"
		col := NewTristateColumn(fieldname, tagmap, isPointer)
		return &col
	case gotypes.TimeType:
		col := NewDateTimeColumn(fieldname, tagmap, isPointer)
		return &col
	}
	switch fieldType.Kind() {
	case reflect.String:
		col := NewTextColumn(fieldname, tagmap, isPointer)
		return &col
	case reflect.Int, reflect.Int32:
		tagmap[TAG_WIDTH] = intWidthString("INT")
		col := NewIntegerColumn(fieldname, "INT", false, tagmap, isPointer)
		return &col
	case reflect.Int8:
		tagmap[TAG_WIDTH] = intWidthString("TINYINT")
		col := NewIntegerColumn(fieldname, "TINYINT", false, tagmap, isPointer)
		return &col
	case reflect.Int16:
		tagmap[TAG_WIDTH] = intWidthString("SMALLINT")
		col := NewIntegerColumn(fieldname, "SMALLINT", false, tagmap, isPointer)
		return &col
	case reflect.Int64:
		tagmap[TAG_WIDTH] = intWidthString("BIGINT")
		col := NewIntegerColumn(fieldname, "BIGINT", false, tagmap, isPointer)
		return &col
	case reflect.Uint, reflect.Uint32:
		tagmap[TAG_WIDTH] = uintWidthString("INT")
		col := NewIntegerColumn(fieldname, "INT", true, tagmap, isPointer)
		return &col
	case reflect.Uint8:
		tagmap[TAG_WIDTH] = uintWidthString("TINYINT")
		col := NewIntegerColumn(fieldname, "TINYINT", true, tagmap, isPointer)
		return &col
	case reflect.Uint16:
		tagmap[TAG_WIDTH] = uintWidthString("SMALLINT")
		col := NewIntegerColumn(fieldname, "SMALLINT", true, tagmap, isPointer)
		return &col
	case reflect.Uint64:
		tagmap[TAG_WIDTH] = uintWidthString("BIGINT")
		col := NewIntegerColumn(fieldname, "BIGINT", true, tagmap, isPointer)
		return &col
	case reflect.Bool:
		tagmap[TAG_WIDTH] = "1"
		col := NewBooleanColumn(fieldname, tagmap, isPointer)
		return &col
	case reflect.Float32, reflect.Float64:
		if _, ok := tagmap[TAG_WIDTH]; ok {
			col := NewDecimalColumn(fieldname, tagmap, isPointer)
			return &col
		} else {
			colType := "FLOAT"
			if fieldType == gotypes.Float64Type {
				colType = "DOUBLE"
			}
			col := NewFloatColumn(fieldname, colType, tagmap, isPointer)
			return &col
		}
	}
	if fieldType.Implements(gotypes.ISerializableType) {
		col := NewCompoundColumn(fieldname, tagmap, isPointer)
		return &col
	}
	return nil
}

func struct2TableSpec(sv reflect.Value, table *STableSpec) {
	fields := reflectutils.FetchStructFieldValueSet(sv)
	autoIncCnt := 0
	for i := 0; i < len(fields); i += 1 {
		column := structField2ColumnSpec(&fields[i])
		if column != nil {
			if intC, ok := column.(*SIntegerColumn); ok && intC.IsAutoIncrement {
				autoIncCnt += 1
				if autoIncCnt > 1 {
					panic(fmt.Sprintf("Table %s contains multiple autoincremental columns!!", table.name))
				}
			}
			if column.IsIndex() {
				table.AddIndex(column.IsUnique(), column.Name())
			}
			table.columns = append(table.columns, column)
		}
	}
}
