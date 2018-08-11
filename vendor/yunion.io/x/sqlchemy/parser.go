package sqlchemy

import (
	"reflect"

	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/pkg/utils"
)

func fieldToColumnSpec(field *reflect.StructField) IColumnSpec {
	fieldname := reflectutils.GetStructFieldName(field)
	tagmap := utils.TagMap(field.Tag)
	if _, ok := tagmap[TAG_IGNORE]; ok {
		return nil
	}
	switch field.Type {
	case gotypes.StringType:
		col := NewTextColumn(fieldname, tagmap)
		return &col
	case gotypes.IntType, gotypes.Int32Type:
		tagmap[TAG_WIDTH] = "11"
		col := NewIntegerColumn(fieldname, "INT", false, tagmap)
		return &col
	case gotypes.Int8Type:
		tagmap[TAG_WIDTH] = "4"
		col := NewIntegerColumn(fieldname, "TINYINT", false, tagmap)
		return &col
	case gotypes.Int16Type:
		tagmap[TAG_WIDTH] = "6"
		col := NewIntegerColumn(fieldname, "SMALLINT", false, tagmap)
		return &col
	case gotypes.Int64Type:
		tagmap[TAG_WIDTH] = "20"
		col := NewIntegerColumn(fieldname, "BIGINT", false, tagmap)
		return &col
	case gotypes.UintType, gotypes.Uint32Type:
		tagmap[TAG_WIDTH] = "11"
		col := NewIntegerColumn(fieldname, "INT", true, tagmap)
		return &col
	case gotypes.Uint8Type:
		tagmap[TAG_WIDTH] = "4"
		col := NewIntegerColumn(fieldname, "TINYINT", true, tagmap)
		return &col
	case gotypes.Uint16Type:
		tagmap[TAG_WIDTH] = "6"
		col := NewIntegerColumn(fieldname, "SMALLINT", true, tagmap)
		return &col
	case gotypes.Uint64Type:
		tagmap[TAG_WIDTH] = "20"
		col := NewIntegerColumn(fieldname, "BIGINT", true, tagmap)
		return &col
	case gotypes.BoolType:
		tagmap[TAG_WIDTH] = "1"
		col := NewBooleanColumn(fieldname, tagmap)
		return &col
	case tristate.TriStateType:
		tagmap[TAG_WIDTH] = "1"
		col := NewTristateColumn(fieldname, tagmap)
		return &col
	case gotypes.Float32Type, gotypes.Float64Type:
		if _, ok := tagmap[TAG_WIDTH]; ok {
			col := NewDecimalColumn(fieldname, tagmap)
			return &col
		} else {
			colType := "FLOAT"
			if field.Type == gotypes.Float64Type {
				colType = "DOUBLE"
			}
			col := NewFloatColumn(fieldname, colType, tagmap)
			return &col
		}
	case gotypes.TimeType:
		col := NewDateTimeColumn(fieldname, tagmap)
		return &col
	/*case jsonutils.JSONDictType:
		col := NewJSONColumn(fieldname, tagmap)
		return &col
	case jsonutils.JSONArrayType:
		col := NewJSONColumn(fieldname, tagmap)
		return &col
	case jsonutils.JSONObjectType:
		col := NewJSONColumn(fieldname, tagmap)
		return &col*/
	default:
		if gotypes.IsSerializable(field.Type) {
			col := NewCompoundColumn(fieldname, tagmap)
			return &col
		} else {
			log.Fatalf("Unsupported type! %s", field.Type)
			return nil
		}
	}
}

func struct2TableSpec(st reflect.Type, table *STableSpec) {
	for i := 0; i < st.NumField(); i++ {
		f := st.Field(i)
		if f.Type.Kind() == reflect.Struct && f.Type != gotypes.TimeType {
			struct2TableSpec(f.Type, table)
		} else {
			coldef := fieldToColumnSpec(&f)
			if coldef != nil {
				table.columns = append(table.columns, coldef)
			}
		}
	}
}
