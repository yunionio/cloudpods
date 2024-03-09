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
	"bytes"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/sqlchemy"
)

func columnDefinitionBuffer(c sqlchemy.IColumnSpec) bytes.Buffer {
	var buf bytes.Buffer
	buf.WriteByte('"')
	buf.WriteString(c.Name())
	buf.WriteByte('"')
	buf.WriteByte(' ')
	buf.WriteString(c.ColType())

	extra := c.ExtraDefs()
	if len(extra) > 0 {
		buf.WriteString(" ")
		buf.WriteString(extra)
	}

	def := c.Default()
	if hasDefault(c) {
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

	if !c.IsNullable() {
		buf.WriteString(" NOT NULL")
	}

	return buf
}

func hasDefault(c sqlchemy.IColumnSpec) bool {
	// if c is NOT NULL, default value must be set
	// if datetime, empty default value is allowed for not null
	return (c.Default() != "" || (!c.IsNullable() && !c.IsDateTime())) && !c.IsAutoIncrement()
}

// SBooleanColumn represents a boolean type column, which is a int(1) for mysql, with value of true or false
type SBooleanColumn struct {
	sqlchemy.SBaseWidthColumn
}

// DefinitionString implementation of SBooleanColumn for IColumnSpec
func (c *SBooleanColumn) DefinitionString() string {
	buf := columnDefinitionBuffer(c)
	return buf.String()
}

// ConvertFromString implementation of SBooleanColumn for IColumnSpec
func (c *SBooleanColumn) ConvertFromString(str string) interface{} {
	switch strings.ToLower(str) {
	case "true", "yes", "on", "ok", "1":
		return 1
	default:
		return 0
	}
}

// ConvertFromValue implementation of STristateColumn for IColumnSpec
func (c *SBooleanColumn) ConvertFromValue(val interface{}) interface{} {
	var bVal bool
	if c.IsPointer() {
		bVal = *val.(*bool)
	} else {
		bVal = val.(bool)
	}
	if bVal {
		return 1
	}
	return 0
}

// IsZero implementation of SBooleanColumn for IColumnSpec
func (c *SBooleanColumn) IsZero(val interface{}) bool {
	if c.IsPointer() {
		bVal := val.(*bool)
		return bVal == nil
	}
	bVal := val.(bool)
	return bVal == false
}

// NewBooleanColumn return an instance of SBooleanColumn
func NewBooleanColumn(name string, tagmap map[string]string, isPointer bool) SBooleanColumn {
	bc := SBooleanColumn{SBaseWidthColumn: sqlchemy.NewBaseWidthColumn(name, "TINYINT", tagmap, isPointer)}
	if !bc.IsPointer() && len(bc.Default()) > 0 && bc.ConvertFromString(bc.Default()) == 1 {
		msg := fmt.Sprintf("Non-pointer boolean column should not default true: %s(%s)", name, tagmap)
		panic(msg)
	}
	return bc
}

// STristateColumn represents a tristate type column, with value of true, false or none
type STristateColumn struct {
	sqlchemy.SBaseWidthColumn
}

// DefinitionString implementation of STristateColumn for IColumnSpec
func (c *STristateColumn) DefinitionString() string {
	buf := columnDefinitionBuffer(c)
	return buf.String()
}

// ConvertFromString implementation of STristateColumn for IColumnSpec
func (c *STristateColumn) ConvertFromString(str string) interface{} {
	switch strings.ToLower(str) {
	case "true", "yes", "on", "ok", "1":
		return 1
	case "none", "null", "unknown":
		return sql.NullInt32{}
	default:
		return 0
	}
}

// ConvertFromValue implementation of STristateColumn for IColumnSpec
func (c *STristateColumn) ConvertFromValue(val interface{}) interface{} {
	bVal := val.(tristate.TriState)
	if bVal == tristate.True {
		return 1
	} else if bVal == tristate.False {
		return 0
	} else {
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
	if _, ok := tagmap[sqlchemy.TAG_NULLABLE]; ok {
		// simply warning, for backward compatiblity reason
		// tristate always nullable
		// delete(tagmap, sqlchemy.TAG_NULLABLE)
		log.Warningf("%s TristateColumn %s should have no nullable tag", table, name)
	}
	bc := STristateColumn{SBaseWidthColumn: sqlchemy.NewBaseWidthColumn(name, "TINYINT", tagmap, isPointer)}
	return bc
}

// SIntegerColumn represents an integer type of column, with value of integer
type SIntegerColumn struct {
	sqlchemy.SBaseColumn

	// Is this column an autoincrement colmn
	isAutoIncrement bool

	// Is this column is a version column for this records
	isAutoVersion bool

	// Is this column a unsigned integer?
	// isUnsigned bool

	// If this column is an autoincrement column, AutoIncrementOffset records the initial offset
	autoIncrementOffset int64
}

// IsNumeric implementation of SIntegerColumn for IColumnSpec
func (c *SIntegerColumn) IsNumeric() bool {
	return true
}

// ExtraDefs implementation of SIntegerColumn for IColumnSpec
func (c *SIntegerColumn) ExtraDefs() string {
	if c.isAutoIncrement {
		return fmt.Sprintf("IDENTITY(%d, %d)", c.autoIncrementOffset, 1)
	}
	return ""
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

// ConvertFromString implementation of SBooleanColumn for IColumnSpec
func (c *SIntegerColumn) ConvertFromString(str string) interface{} {
	val, _ := strconv.ParseInt(str, 10, 64)
	return val
}

func (c *SIntegerColumn) IsAutoVersion() bool {
	return c.isAutoVersion
}

func (c *SIntegerColumn) IsAutoIncrement() bool {
	return c.isAutoIncrement
}

func (c *SIntegerColumn) AutoIncrementOffset() int64 {
	return c.autoIncrementOffset
}

func (c *SIntegerColumn) SetAutoIncrement(on bool) {
	c.isAutoIncrement = on
}

func (c *SIntegerColumn) SetAutoIncrementOffset(offset int64) {
	c.autoIncrementOffset = offset
}

// NewIntegerColumn return an instance of SIntegerColumn
func NewIntegerColumn(name string, sqltype string, tagmap map[string]string, isPointer bool) SIntegerColumn {
	autoinc := false
	autoincBase := int64(1)
	tagmap, v, ok := utils.TagPop(tagmap, sqlchemy.TAG_AUTOINCREMENT)
	if ok {
		base, err := strconv.ParseInt(v, 10, 64)
		if err == nil && base > 0 {
			autoinc = true
			autoincBase = base
		} else {
			autoinc = utils.ToBool(v)
		}
	}
	autover := false
	tagmap, v, ok = utils.TagPop(tagmap, sqlchemy.TAG_AUTOVERSION)
	if ok {
		autover = utils.ToBool(v)
	}
	c := SIntegerColumn{
		SBaseColumn:         sqlchemy.NewBaseColumn(name, sqltype, tagmap, isPointer),
		isAutoIncrement:     autoinc,
		autoIncrementOffset: autoincBase,
		isAutoVersion:       autover,
	}
	if autoinc {
		c.SetPrimary(true) // autoincrement column must be primary key
		c.SetNullable(false)
		c.isAutoVersion = false
	} else if autover {
		c.SetPrimary(false)
		c.SetNullable(false)
		if len(c.Default()) == 0 {
			c.SetDefault("0")
		}
	}
	return c
}

// SFloatColumn represents a float type column, e.g. float32 or float64
type SFloatColumn struct {
	sqlchemy.SBaseColumn
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
			return val.(*float32) == nil
		case *float64:
			return val.(*float64) == nil
		}
	} else {
		switch val.(type) {
		case float32:
			return val.(float32) == 0.0
		case float64:
			return val.(float64) == 0.0
		}
	}
	return true
}

// ConvertFromString implementation of SBooleanColumn for IColumnSpec
func (c *SFloatColumn) ConvertFromString(str string) interface{} {
	val, _ := strconv.ParseFloat(str, 64)
	return val
}

// NewFloatColumn returns an instance of SFloatColumn
func NewFloatColumn(name string, sqlType string, tagmap map[string]string, isPointer bool) SFloatColumn {
	return SFloatColumn{SBaseColumn: sqlchemy.NewBaseColumn(name, sqlType, tagmap, isPointer)}
}

// SDecimalColumn represents a DECIMAL type of column, i.e. a float with fixed width of digits
type SDecimalColumn struct {
	sqlchemy.SBaseWidthColumn
	Precision int
}

// ColType implementation of SDecimalColumn for IColumnSpec
func (c *SDecimalColumn) ColType() string {
	str := c.SBaseWidthColumn.ColType()
	if str[len(str)-1] == ')' {
		str = str[:len(str)-1]
	}
	return fmt.Sprintf("%s, %d)", str, c.Precision)
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
			return val.(*float32) == nil
		case *float64:
			return val.(*float64) == nil
		}
	} else {
		switch val.(type) {
		case float32:
			return val.(float32) == 0.0
		case float64:
			return val.(float64) == 0.0
		}
	}
	return true
}

// ConvertFromString implementation of SBooleanColumn for IColumnSpec
func (c *SDecimalColumn) ConvertFromString(str string) interface{} {
	val, _ := strconv.ParseFloat(str, 64)
	return val
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
	return SDecimalColumn{
		SBaseWidthColumn: sqlchemy.NewBaseWidthColumn(name, "DECIMAL", tagmap, isPointer),
		Precision:        prec,
	}
}

// STextColumn represents a text type of column
type STextColumn struct {
	sqlchemy.SBaseWidthColumn
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

// ConvertFromString implementation of SBooleanColumn for IColumnSpec
func (c *STextColumn) ConvertFromString(str string) interface{} {
	return str
}

func (c *STextColumn) IsString() bool {
	return true
}

// NewTextColumn return an instance of STextColumn
func NewTextColumn(name string, sqlType string, tagmap map[string]string, isPointer bool) STextColumn {
	return STextColumn{
		SBaseWidthColumn: sqlchemy.NewBaseWidthColumn(name, sqlType, tagmap, isPointer),
	}
}

// STimeTypeColumn represents a Detetime type of column, e.g. DateTime
type STimeTypeColumn struct {
	sqlchemy.SBaseColumn

	// timestamp precision of nanoseconds, 0-6
	precision int
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

// ColType implementation of SIntegerColumn for IColumnSpec
func (c *STimeTypeColumn) ColType() string {
	return fmt.Sprintf("%s(%d)", (&c.SBaseColumn).ColType(), c.precision)
}

// ConvertFromString implementation of SBooleanColumn for IColumnSpec
func (c *STimeTypeColumn) ConvertFromString(str string) interface{} {
	tm, _ := timeutils.ParseTimeStr(str)
	return tm
}

// NewTimeTypeColumn return an instance of STimeTypeColumn
func NewTimeTypeColumn(name string, typeStr string, tagmap map[string]string, isPointer bool) STimeTypeColumn {
	tagmap, v, ok := utils.TagPop(tagmap, sqlchemy.TAG_PRECISION)
	prec := 0
	if ok {
		prec, _ = strconv.Atoi(v)
		if prec > 6 || prec < 0 {
			log.Warningf("datetime field %s precision %d change to 6", name, prec)
			prec = 0
		}
	}
	dc := STimeTypeColumn{
		SBaseColumn: sqlchemy.NewBaseColumn(name, typeStr, tagmap, isPointer),
		precision:   prec,
	}
	return dc
}

// SDateTimeColumn represents a DateTime type of column
type SDateTimeColumn struct {
	STimeTypeColumn

	// Is this column a 'created_at' field, which records the time of create this record
	isCreatedAt bool

	// Is this column a 'updated_at' field, which records the time when this record was updated
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
		STimeTypeColumn: NewTimeTypeColumn(name, "TIMESTAMP", tagmap, isPointer),
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
func NewCompoundColumn(name string, sqlType string, tagmap map[string]string, isPointer bool) CompoundColumn {
	dtc := CompoundColumn{STextColumn: NewTextColumn(name, sqlType, tagmap, isPointer)}
	return dtc
}
