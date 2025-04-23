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
	"bytes"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/sqlchemy"
)

type IClickhouseColumnSpec interface {
	sqlchemy.IColumnSpec

	// IsOrderBy defines whether the column appears in order by clause
	IsOrderBy() bool

	// PartitionBy defines expression that the column appaers in Partition by clause
	PartitionBy() string

	// SetOrderBy set isOrderBy field
	SetOrderBy(on bool)

	// SetPartitionBy set partitonby field
	SetPartitionBy(expr string)

	// GetTTL returns the ttl setting of a time column
	GetTTL() (int, string)

	// SetTTL sets the ttl parameters of a time column
	SetTTL(int, string)
}

func columnDefinitionBuffer(c sqlchemy.IColumnSpec) bytes.Buffer {
	var buf bytes.Buffer
	buf.WriteByte('`')
	buf.WriteString(c.Name())
	buf.WriteByte('`')
	buf.WriteByte(' ')

	if c.IsNullable() {
		buf.WriteString("Nullable(")
	}

	buf.WriteString(c.ColType())

	if c.IsNullable() {
		buf.WriteString(")")
	}

	def := c.Default()
	defOk := c.IsSupportDefault()
	if def != "" {
		if !defOk {
			panic(fmt.Errorf("column %q type %q does not support having default value: %q",
				c.Name(), c.ColType(), def,
			))
		}
		def = sqlchemy.GetStringValue(c.ConvertFromString(def))
		buf.WriteString(" DEFAULT ")
		if c.IsText() {
			buf.WriteByte('\'')
		}
		buf.WriteString(def)
		if c.IsText() {
			buf.WriteByte('\'')
		}
	}

	return buf
}

type SClickhouseBaseColumn struct {
	sqlchemy.SBaseColumn

	partionBy string
	isOrderBy bool
}

func (c *SClickhouseBaseColumn) IsOrderBy() bool {
	return c.isOrderBy
}

func (c *SClickhouseBaseColumn) SetOrderBy(on bool) {
	c.isOrderBy = on
}

func (c *SClickhouseBaseColumn) PartitionBy() string {
	return c.partionBy
}

func (c *SClickhouseBaseColumn) SetPartitionBy(expr string) {
	c.partionBy = expr
}

func (c *SClickhouseBaseColumn) GetTTL() (int, string) {
	return 0, ""
}

func (c *SClickhouseBaseColumn) SetTTL(int, string) {
	// null ops
}

func NewClickhouseBaseColumn(name string, sqltype string, tagmap map[string]string, isPointer bool) SClickhouseBaseColumn {
	var ok bool
	var val string
	partition := ""
	tagmap, val, ok = utils.TagPop(tagmap, TAG_PARTITION)
	if ok {
		partition = val
	}
	orderBy := false
	tagmap, val, ok = utils.TagPop(tagmap, TAG_ORDER)
	if ok {
		orderBy = utils.ToBool(val)
	}
	return SClickhouseBaseColumn{
		SBaseColumn: sqlchemy.NewBaseColumn(name, sqltype, tagmap, isPointer),
		partionBy:   partition,
		isOrderBy:   orderBy,
	}
}

// SBooleanColumn represents a boolean type column, which is a int(1) for mysql, with value of true or false
type SBooleanColumn struct {
	SClickhouseBaseColumn
}

// DefinitionString implementation of SBooleanColumn for IColumnSpec
func (c *SBooleanColumn) DefinitionString() string {
	buf := columnDefinitionBuffer(c)
	return buf.String()
}

// ConvertFromString implementation of SBooleanColumn for IColumnSpec
func (c *SBooleanColumn) ConvertFromString(str string) interface{} {
	switch sqlchemy.ConvertValueToBool(str) {
	case true:
		return uint8(1)
	default:
		return uint8(0)
	}
}

// ConvertFromValue implementation of STristateColumn for IColumnSpec
func (c *SBooleanColumn) ConvertFromValue(val interface{}) interface{} {
	switch sqlchemy.ConvertValueToBool(val) {
	case true:
		return uint8(1)
	default:
		return uint8(0)
	}
}

// IsZero implementation of SBooleanColumn for IColumnSpec
func (c *SBooleanColumn) IsZero(val interface{}) bool {
	if c.IsPointer() {
		bVal := val.(*bool)
		return bVal == nil
	}
	bVal := val.(bool)
	return !bVal
}

// NewBooleanColumn return an instance of SBooleanColumn
func NewBooleanColumn(name string, tagmap map[string]string, isPointer bool) SBooleanColumn {
	bc := SBooleanColumn{SClickhouseBaseColumn: NewClickhouseBaseColumn(name, "UInt8", tagmap, isPointer)}
	if !bc.IsPointer() && len(bc.Default()) > 0 && bc.ConvertFromString(bc.Default()) == uint8(1) {
		msg := fmt.Sprintf("Non-pointer boolean column should not default true: %s(%s)", name, tagmap)
		panic(msg)
	}
	return bc
}

// STristateColumn represents a tristate type column, with value of true, false or none
type STristateColumn struct {
	SClickhouseBaseColumn
}

// DefinitionString implementation of STristateColumn for IColumnSpec
func (c *STristateColumn) DefinitionString() string {
	buf := columnDefinitionBuffer(c)
	return buf.String()
}

// ConvertFromString implementation of STristateColumn for IColumnSpec
func (c *STristateColumn) ConvertFromString(str string) interface{} {
	switch sqlchemy.ConvertValueToTriState(str) {
	case tristate.True:
		return uint8(1)
	case tristate.False:
		return uint8(0)
	default:
		return sql.NullInt32{}
	}
}

// ConvertFromValue implementation of STristateColumn for IColumnSpec
func (c *STristateColumn) ConvertFromValue(val interface{}) interface{} {
	switch sqlchemy.ConvertValueToTriState(val) {
	case tristate.True:
		return uint8(1)
	case tristate.False:
		return uint8(0)
	default:
		return sql.NullInt32{}
	}
}

// IsZero implementation of STristateColumn for IColumnSpec
func (c *STristateColumn) IsZero(val interface{}) bool {
	if c.IsPointer() {
		bVal := val.(*tristate.TriState)
		return bVal == nil
	}
	bVal := val.(tristate.TriState)
	return bVal == tristate.None
}

// NewTristateColumn return an instance of STristateColumn
func NewTristateColumn(table, name string, tagmap map[string]string, isPointer bool) STristateColumn {
	//if _, ok := tagmap[sqlchemy.TAG_NULLABLE]; ok {
	// tristate always nullable
	delete(tagmap, sqlchemy.TAG_NULLABLE)
	//}
	bc := STristateColumn{SClickhouseBaseColumn: NewClickhouseBaseColumn(name, "UInt8", tagmap, isPointer)}
	return bc
}

// SIntegerColumn represents an integer type of column, with value of integer
type SIntegerColumn struct {
	SClickhouseBaseColumn

	// Is this column is a version column for this records
	isAutoVersion bool
}

// IsNumeric implementation of SIntegerColumn for IColumnSpec
func (c *SIntegerColumn) IsNumeric() bool {
	return true
}

// DefinitionString implementation of SIntegerColumn for IColumnSpec
func (c *SIntegerColumn) DefinitionString() string {
	buf := columnDefinitionBuffer(c)
	return buf.String()
}

// IsZero implementation of SIntegerColumn for IColumnSpec
func (c *SIntegerColumn) IsZero(val interface{}) bool {
	if val == nil || (c.IsPointer() && reflect.ValueOf(val).IsNil()) {
		return true
	}
	switch intVal := val.(type) {
	case int8, int16, int32, int64, int, uint, uint8, uint16, uint32, uint64:
		return intVal == 0
	}
	return true
}

// ConvertFromString implementation of STristateColumn for IColumnSpec
func (c *SIntegerColumn) ConvertFromString(str string) interface{} {
	val := sqlchemy.ConvertValueToInteger(str)
	switch c.ColType() {
	case "UInt8":
		return uint8(val)
	case "UInt16":
		return uint16(val)
	case "UInt32":
		return uint32(val)
	case "UInt64":
		return val
	case "Int8":
		return int8(val)
	case "Int16":
		return int16(val)
	case "Int32":
		return int32(val)
	case "Int64":
		return val
	}
	panic(fmt.Sprintf("unsupported type %s", c.ColType()))
}

// IsAutoVersion implements IsAutoVersion for IColumnSpec
func (c *SIntegerColumn) IsAutoVersion() bool {
	return c.isAutoVersion
}

// NewIntegerColumn return an instance of SIntegerColumn
func NewIntegerColumn(name string, sqltype string, tagmap map[string]string, isPointer bool) SIntegerColumn {
	isAutoVersion := false
	if _, ok := tagmap[sqlchemy.TAG_AUTOVERSION]; ok {
		isAutoVersion = true
	}
	if _, ok := tagmap[sqlchemy.TAG_AUTOINCREMENT]; ok {
		log.Warningf("auto_increment field %s not supported by ClickHouse", name)
	}
	c := SIntegerColumn{
		SClickhouseBaseColumn: NewClickhouseBaseColumn(name, sqltype, tagmap, isPointer),
		isAutoVersion:         isAutoVersion,
	}
	return c
}

// SFloatColumn represents a float type column, e.g. float32 or float64
type SFloatColumn struct {
	SClickhouseBaseColumn
}

// IsNumeric implementation of SFloatColumn for IColumnSpec
func (c *SFloatColumn) IsNumeric() bool {
	return true
}

// DefinitionString implementation of SFloatColumn for IColumnSpec
func (c *SFloatColumn) DefinitionString() string {
	buf := columnDefinitionBuffer(c)
	return buf.String()
}

// IsZero implementation of SFloatColumn for IColumnSpec
func (c *SFloatColumn) IsZero(val interface{}) bool {
	if c.IsPointer() {
		switch val.(type) {
		case *float32:
			return val == nil
		case *float64:
			return val == nil
		}
	} else {
		switch val.(type) {
		case float32:
			return val == 0.0
		case float64:
			return val == 0.0
		}
	}
	return true
}

// ConvertFromString implementation of STristateColumn for IColumnSpec
func (c *SFloatColumn) ConvertFromString(str string) interface{} {
	floatVal := sqlchemy.ConvertValueToFloat(str)
	switch c.ColType() {
	case "Float32":
		return float32(floatVal)
	case "Float64":
		return floatVal
	}
	panic(fmt.Sprintf("unsupported type %s", c.ColType()))
}

// NewFloatColumn returns an instance of SFloatColumn
func NewFloatColumn(name string, sqlType string, tagmap map[string]string, isPointer bool) SFloatColumn {
	return SFloatColumn{
		SClickhouseBaseColumn: NewClickhouseBaseColumn(name, sqlType, tagmap, isPointer),
	}
}

// SDecimalColumn represents a DECIMAL type of column, i.e. a float with fixed width of digits
type SDecimalColumn struct {
	SClickhouseBaseColumn
	width     int
	Precision int
}

// ColType implementation of SDecimalColumn for IColumnSpec
func (c *SDecimalColumn) ColType() string {
	str := c.SClickhouseBaseColumn.ColType()
	if str == "Decimal" {
		return fmt.Sprintf("%s(%d, %d)", str, c.width, c.Precision)
	}
	return fmt.Sprintf("%s(%d)", str, c.Precision)
}

// IsNumeric implementation of SDecimalColumn for IColumnSpec
func (c *SDecimalColumn) IsNumeric() bool {
	return true
}

// DefinitionString implementation of SDecimalColumn for IColumnSpec
func (c *SDecimalColumn) DefinitionString() string {
	buf := columnDefinitionBuffer(c)
	return buf.String()
}

// IsZero implementation of SDecimalColumn for IColumnSpec
func (c *SDecimalColumn) IsZero(val interface{}) bool {
	if c.IsPointer() {
		switch val.(type) {
		case *float32:
			return val == nil
		case *float64:
			return val == nil
		}
	} else {
		switch val.(type) {
		case float32:
			return val == 0.0
		case float64:
			return val == 0.0
		}
	}
	return true
}

// ConvertFromString implementation of STristateColumn for IColumnSpec
func (c *SDecimalColumn) ConvertFromString(str string) interface{} {
	return sqlchemy.ConvertValueToFloat(str)
}

// NewDecimalColumn returns an instance of SDecimalColumn
func NewDecimalColumn(name string, tagmap map[string]string, isPointer bool) SDecimalColumn {
	tagmap, v, ok := utils.TagPop(tagmap, sqlchemy.TAG_PRECISION)
	if !ok {
		panic(fmt.Sprintf("Field %q of float misses precision tag", name))
	}
	prec, err := strconv.Atoi(v)
	if err != nil {
		panic(fmt.Sprintf("Field precision of %q shoud be integer (%q)", name, v))
	}
	tagmap, v, ok = utils.TagPop(tagmap, sqlchemy.TAG_WIDTH)
	if !ok {
		panic(fmt.Sprintf("Field %q of float misses width tag", name))
	}
	width, err := strconv.Atoi(v)
	if err != nil {
		panic(fmt.Sprintf("Field width of %q shoud be integer (%q)", name, v))
	}
	var sqlType string
	if width <= 9 {
		sqlType = "Decimal32"
	} else if width <= 18 {
		sqlType = "Decimal64"
	} else if width <= 38 {
		sqlType = "Decimal128"
	} else if width <= 76 {
		sqlType = "Decimal256"
	} else {
		panic(fmt.Sprintf("unsupported decimal width %d", width))
	}
	c := SDecimalColumn{
		SClickhouseBaseColumn: NewClickhouseBaseColumn(name, sqlType, tagmap, isPointer),
		width:                 width,
		Precision:             prec,
	}
	return c
}

// STextColumn represents a text type of column
type STextColumn struct {
	SClickhouseBaseColumn
}

// IsText implementation of STextColumn for IColumnSpec
func (c *STextColumn) IsText() bool {
	return true
}

// IsSearchable implementation of STextColumn for IColumnSpec
func (c *STextColumn) IsSearchable() bool {
	return true
}

// IsAscii implementation of STextColumn for IColumnSpec
func (c *STextColumn) IsAscii() bool {
	return false
}

// DefinitionString implementation of STextColumn for IColumnSpec
func (c *STextColumn) DefinitionString() string {
	buf := columnDefinitionBuffer(c)
	return buf.String()
}

// IsZero implementation of STextColumn for IColumnSpec
func (c *STextColumn) IsZero(val interface{}) bool {
	if c.IsPointer() {
		return gotypes.IsNil(val)
	}
	return reflect.ValueOf(val).Len() == 0
}

func (c *STextColumn) IsString() bool {
	return true
}

// ConvertFromString implementation of STristateColumn for IColumnSpec
func (c *STextColumn) ConvertFromString(str string) interface{} {
	return str
}

// NewTextColumn return an instance of STextColumn
func NewTextColumn(name string, sqlType string, tagmap map[string]string, isPointer bool) STextColumn {
	return STextColumn{
		SClickhouseBaseColumn: NewClickhouseBaseColumn(name, sqlType, tagmap, isPointer),
	}
}

// STimeTypeColumn represents a Detetime type of column, e.g. DateTime
type STimeTypeColumn struct {
	SClickhouseBaseColumn

	ttl sTTL
}

// IsText implementation of STimeTypeColumn for IColumnSpec
func (c *STimeTypeColumn) IsText() bool {
	return true
}

// DefinitionString implementation of STimeTypeColumn for IColumnSpec
func (c *STimeTypeColumn) DefinitionString() string {
	buf := columnDefinitionBuffer(c)
	return buf.String()
}

// IsZero implementation of STimeTypeColumn for IColumnSpec
func (c *STimeTypeColumn) IsZero(val interface{}) bool {
	if c.IsPointer() {
		bVal := val.(*time.Time)
		return bVal == nil
	}
	bVal := val.(time.Time)
	return bVal.IsZero()
}

// ConvertFromString implementation of STristateColumn for IColumnSpec
func (c *STimeTypeColumn) ConvertFromString(str string) interface{} {
	return sqlchemy.ConvertValueToTime(str)
}

// ConvertFromValue implementation of STimeTypeColumn for IColumnSpec
func (c *STimeTypeColumn) ConvertFromValue(val interface{}) interface{} {
	return sqlchemy.ConvertValueToTime(val)
}

func (c *STimeTypeColumn) GetTTL() (int, string) {
	return c.ttl.Count, c.ttl.Unit
}

func (c *STimeTypeColumn) SetTTL(cnt int, u string) {
	c.ttl.Count = cnt
	c.ttl.Unit = u
}

// NewTimeTypeColumn return an instance of STimeTypeColumn
func NewTimeTypeColumn(name string, typeStr string, tagmap map[string]string, isPointer bool) STimeTypeColumn {
	var ttlCfg sTTL
	var ttl string
	var ok bool
	tagmap, ttl, ok = utils.TagPop(tagmap, TAG_TTL)
	if ok {
		var err error
		ttlCfg, err = parseTTL(ttl)
		if err != nil {
			log.Warningf("invalid ttl %s: %s", ttl, err)
		}
	}
	dc := STimeTypeColumn{
		SClickhouseBaseColumn: NewClickhouseBaseColumn(name, typeStr, tagmap, isPointer),
		ttl:                   ttlCfg,
	}
	return dc
}

// SDateTimeColumn represents a DateTime type of column
type SDateTimeColumn struct {
	STimeTypeColumn

	// Is this column a 'created_at' field, whichi records the time of create this record
	isCreatedAt bool

	// Is this column a 'updated_at' field, whichi records the time when this record was updated
	isUpdatedAt bool
}

// DefinitionString implementation of SDateTimeColumn for IColumnSpec
func (c *SDateTimeColumn) DefinitionString() string {
	buf := columnDefinitionBuffer(c)
	return buf.String()
}

func (c *SDateTimeColumn) IsCreatedAt() bool {
	return c.isCreatedAt
}

func (c *SDateTimeColumn) IsUpdatedAt() bool {
	return c.isUpdatedAt
}

func (c *SDateTimeColumn) IsDateTime() bool {
	return true
}

// NewDateTimeColumn returns an instance of DateTime column
func NewDateTimeColumn(name string, tagmap map[string]string, isPointer bool) SDateTimeColumn {
	createdAt := false
	updatedAt := false
	tagmap, v, ok := utils.TagPop(tagmap, sqlchemy.TAG_CREATE_TIMESTAMP)
	if ok {
		createdAt = utils.ToBool(v)
	}
	tagmap, v, ok = utils.TagPop(tagmap, sqlchemy.TAG_UPDATE_TIMESTAMP)
	if ok {
		updatedAt = utils.ToBool(v)
	}
	dtc := SDateTimeColumn{
		STimeTypeColumn: NewTimeTypeColumn(name, "DateTime('UTC')", tagmap, isPointer),
		isCreatedAt:     createdAt,
		isUpdatedAt:     updatedAt,
	}
	return dtc
}

// CompoundColumn represents a column of compound tye, e.g. a JSON, an Array, or a struct
type CompoundColumn struct {
	STextColumn
	sqlchemy.SBaseCompoundColumn
}

// DefinitionString implementation of CompoundColumn for IColumnSpec
func (c *CompoundColumn) DefinitionString() string {
	buf := columnDefinitionBuffer(c)
	return buf.String()
}

// IsZero implementation of CompoundColumn for IColumnSpec
func (c *CompoundColumn) IsZero(val interface{}) bool {
	if val == nil {
		return true
	}
	if c.IsPointer() && reflect.ValueOf(val).IsNil() {
		return true
	}
	return false
}

// ConvertFromString implementation of CompoundColumn for IColumnSpec
func (c *CompoundColumn) ConvertFromString(str string) interface{} {
	return c.SBaseCompoundColumn.ConvertFromString(str)
}

// ConvertFromValue implementation of CompoundColumn for IColumnSpec
func (c *CompoundColumn) ConvertFromValue(val interface{}) interface{} {
	return c.SBaseCompoundColumn.ConvertFromValue(val)
}

// NewCompoundColumn returns an instance of CompoundColumn
func NewCompoundColumn(name string, tagmap map[string]string, isPointer bool) CompoundColumn {
	dtc := CompoundColumn{STextColumn: NewTextColumn(name, "String", tagmap, isPointer)}
	return dtc
}
