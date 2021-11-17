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
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/utils"
)

// IColumnSpec is an interface that represents a column of a table
type IColumnSpec interface {
	// Name returns the name of the column
	Name() string

	// ColType returns type of the column, e.g. INTEGER, VARCHAR
	ColType() string

	// Default returns default value of the column, represents in string
	Default() string

	// IsSupportDefault returns whether this column supports being given a default value
	IsSupportDefault() bool

	// IsNullable returns whether this column is nullable
	IsNullable() bool

	// SetNullable sets this column as nullable
	SetNullable(on bool)

	// IsPrimary returns whether this column is part of the primary keys
	IsPrimary() bool

	// IsUnique returns whether the value of this column unique for each row
	IsUnique() bool

	// IsIndex returns whether this column is indexable, if it is true, a index of this column will be automatically created
	IsIndex() bool

	// ExtraDefs returns some extra column attribute definitions, not covered by the standard fields
	ExtraDefs() string

	// DefinitionString return the SQL presentation of this column
	DefinitionString() string

	// IsText returns whether this column is actually a text, such a Datetime column is actually a text
	IsText() bool

	// IsSearchable returns whether this column is searchable, e.g. a integer column is not searchable, but a text field is searchable
	IsSearchable() bool

	// IsAscii returns whether this column is an ASCII type text, if true, the column should be compared with a UTF8 string
	IsAscii() bool

	// IsNumeric returns whether this column is a numeric type column, e.g. integer or float
	IsNumeric() bool

	// ConvertFromString returns the SQL representation of a value in string format for this column
	ConvertFromString(str string) string

	// ConvertToString(str string) string

	// ConvertFromValue returns the SQL representation of a value for this column
	ConvertFromValue(val interface{}) interface{}

	// ConvertToValue(str interface{}) interface{}

	// IsZero is used to determine a value is the zero value for this column
	IsZero(val interface{}) bool

	// AllowZero returns whether this column allow a zero value
	AllowZero() bool

	// IsEqual(v1, v2 interface{}) bool

	// Tags returns the field tags for this column, which is in the struct definition
	Tags() map[string]string

	// IsPointer returns whether this column is a pointer type definition, e.g. *int, *bool
	IsPointer() bool

	// SetDefault sets the default value in the format of string for this column
	SetDefault(defStr string)
}

// SBaseColumn is the base structure represents a column
type SBaseColumn struct {
	name          string
	dbName        string
	sqlType       string
	defaultString string
	isPointer     bool
	isNullable    bool
	isPrimary     bool
	isUnique      bool
	isIndex       bool
	isAllowZero   bool
	tags          map[string]string
}

// IsPointer implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) IsPointer() bool {
	return c.isPointer
}

// Name implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) Name() string {
	if len(c.dbName) > 0 {
		return c.dbName
	}
	return c.name
}

// ColType implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) ColType() string {
	return c.sqlType
}

// Default implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) Default() string {
	return c.defaultString
}

// SetDefault implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) SetDefault(defStr string) {
	c.defaultString = defStr
}

// IsSupportDefault implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) IsSupportDefault() bool {
	return true
}

// IsNullable implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) IsNullable() bool {
	return c.isNullable
}

// SetNullable implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) SetNullable(on bool) {
	c.isNullable = on
}

// IsPrimary implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) IsPrimary() bool {
	return c.isPrimary
}

// IsUnique implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) IsUnique() bool {
	return c.isUnique
}

// IsIndex implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) IsIndex() bool {
	return c.isIndex
}

// ExtraDefs implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) ExtraDefs() string {
	return ""
}

// IsText implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) IsText() bool {
	return false
}

// IsAscii implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) IsAscii() bool {
	return false
}

// IsSearchable implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) IsSearchable() bool {
	return false
}

// IsNumeric implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) IsNumeric() bool {
	return false
}

// AllowZero implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) AllowZero() bool {
	return c.isAllowZero
}

// ConvertFromString implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) ConvertFromString(str string) string {
	return str
}

/*func (c *SBaseColumn) ConvertToString(str string) string {
	return str
}*/

// ConvertFromValue implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) ConvertFromValue(val interface{}) interface{} {
	return val
}

/*func (c *SBaseColumn) ConvertToValue(val interface{}) interface{} {
	return val
}*/

// Tags implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) Tags() map[string]string {
	return c.tags
}

// generate SQL representation of a column
func definitionBuffer(c IColumnSpec) bytes.Buffer {
	var buf bytes.Buffer
	buf.WriteByte('`')
	buf.WriteString(c.Name())
	buf.WriteByte('`')
	buf.WriteByte(' ')
	buf.WriteString(c.ColType())

	extra := c.ExtraDefs()
	if len(extra) > 0 {
		buf.WriteString(" ")
		buf.WriteString(extra)
	}

	if !c.IsNullable() {
		buf.WriteString(" NOT NULL")
	}

	def := c.Default()
	defOk := c.IsSupportDefault()
	if def != "" {
		if !defOk {
			panic(fmt.Errorf("column %q type %q does not support having default value: %q",
				c.Name(), c.ColType(), def,
			))
		}
		def = c.ConvertFromString(def)
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

// NewBaseColumn returns an instance of SBaseColumn
func NewBaseColumn(name string, sqltype string, tagmap map[string]string, isPointer bool) SBaseColumn {
	var val string
	var ok bool
	dbName := ""
	tagmap, val, ok = utils.TagPop(tagmap, TAG_NAME)
	if ok {
		dbName = val
	}
	defStr := ""
	tagmap, val, ok = utils.TagPop(tagmap, TAG_DEFAULT)
	if ok {
		defStr = val
	}
	isNullable := true
	tagmap, val, ok = utils.TagPop(tagmap, TAG_NULLABLE)
	if ok {
		isNullable = utils.ToBool(val)
	}
	isPrimary := false
	tagmap, val, ok = utils.TagPop(tagmap, TAG_PRIMARY)
	if ok {
		isPrimary = utils.ToBool(val)
	}
	isUnique := false
	tagmap, val, ok = utils.TagPop(tagmap, TAG_UNIQUE)
	if ok {
		isUnique = utils.ToBool(val)
	}
	isIndex := false
	tagmap, val, ok = utils.TagPop(tagmap, TAG_INDEX)
	if ok {
		isIndex = utils.ToBool(val)
	}
	if isPrimary {
		isNullable = false
	}
	isAllowZero := false
	tagmap, val, ok = utils.TagPop(tagmap, TAG_ALLOW_ZERO)
	if ok {
		isAllowZero = utils.ToBool(val)
	}
	return SBaseColumn{
		name:          name,
		dbName:        dbName,
		sqlType:       sqltype,
		defaultString: defStr,
		isNullable:    isNullable,
		isPrimary:     isPrimary,
		isUnique:      isUnique,
		isIndex:       isIndex,
		tags:          tagmap,
		isPointer:     isPointer,
		isAllowZero:   isAllowZero,
	}
}

// SBaseWidthColumn represents a type of column that with width attribute, such as VARCHAR(20), INT(10)
type SBaseWidthColumn struct {
	SBaseColumn
	width int
}

// ColType implementation of SBaseWidthColumn for IColumnSpec
func (c *SBaseWidthColumn) ColType() string {
	if c.width > 0 {
		return fmt.Sprintf("%s(%d)", c.sqlType, c.width)
	}
	return c.sqlType
}

// NewBaseWidthColumn return an instance of SBaseWidthColumn
func NewBaseWidthColumn(name string, sqltype string, tagmap map[string]string, isPointer bool) SBaseWidthColumn {
	width := 0
	tagmap, v, ok := utils.TagPop(tagmap, TAG_WIDTH)
	if ok {
		width, _ = strconv.Atoi(v)
	}
	wc := SBaseWidthColumn{
		SBaseColumn: NewBaseColumn(name, sqltype, tagmap, isPointer),
		width:       width,
	}
	return wc
}

// SBooleanColumn represents a boolean type column, which is a int(1) for mysql, with value of true or false
type SBooleanColumn struct {
	SBaseWidthColumn
}

// DefinitionString implementation of SBooleanColumn for IColumnSpec
func (c *SBooleanColumn) DefinitionString() string {
	buf := definitionBuffer(c)
	return buf.String()
}

// ConvertFromString implementation of SBooleanColumn for IColumnSpec
func (c *SBooleanColumn) ConvertFromString(str string) string {
	switch strings.ToLower(str) {
	case "true", "yes", "on", "ok", "1":
		return "1"
	default:
		return "0"
	}
}

/*func (c *SBooleanColumn) ConvertFromValue(val interface{}) interface{} {
	switch bVal := val.(type) {
	case bool:
		if bVal {
			return 1
		} else {
			return 0
		}
	case *bool:
		if gotypes.IsNil(bVal) {
			return 0
		} else if *bVal {
			return 1
		} else {
			return 0
		}
	default:
		return 0
	}
}*/

// IsZero implementation of SBooleanColumn for IColumnSpec
func (c *SBooleanColumn) IsZero(val interface{}) bool {
	if c.isPointer {
		bVal := val.(*bool)
		return bVal == nil
	}
	bVal := val.(bool)
	return bVal == false
}

// NewBooleanColumn return an instance of SBooleanColumn
func NewBooleanColumn(name string, tagmap map[string]string, isPointer bool) SBooleanColumn {
	bc := SBooleanColumn{SBaseWidthColumn: NewBaseWidthColumn(name, "TINYINT", tagmap, isPointer)}
	if !bc.IsPointer() && len(bc.Default()) > 0 && bc.ConvertFromString(bc.Default()) == "1" {
		msg := fmt.Sprintf("Non-pointer boolean column should not default true: %s(%s)", name, tagmap)
		panic(msg)
	}
	return bc
}

// STristateColumn represents a tristate type column, with value of true, false or none
type STristateColumn struct {
	SBaseWidthColumn
}

// DefinitionString implementation of STristateColumn for IColumnSpec
func (c *STristateColumn) DefinitionString() string {
	buf := definitionBuffer(c)
	return buf.String()
}

// ConvertFromString implementation of STristateColumn for IColumnSpec
func (c *STristateColumn) ConvertFromString(str string) string {
	switch strings.ToLower(str) {
	case "true", "yes", "on", "ok", "1":
		return "1"
	case "none", "null", "unknown":
		return ""
	default:
		return "0"
	}
}

// ConvertFromValue implementation of STristateColumn for IColumnSpec
func (c *STristateColumn) ConvertFromValue(val interface{}) interface{} {
	bVal := val.(tristate.TriState)
	if bVal == tristate.True {
		return 1
	}
	return 0
}

// IsZero implementation of STristateColumn for IColumnSpec
func (c *STristateColumn) IsZero(val interface{}) bool {
	if c.isPointer {
		bVal := val.(*tristate.TriState)
		return bVal == nil
	}
	bVal := val.(tristate.TriState)
	return bVal == tristate.None
}

// NewTristateColumn return an instance of STristateColumn
func NewTristateColumn(name string, tagmap map[string]string, isPointer bool) STristateColumn {
	bc := STristateColumn{SBaseWidthColumn: NewBaseWidthColumn(name, "TINYINT", tagmap, isPointer)}
	return bc
}

// SIntegerColumn represents an integer type of column, with value of integer
type SIntegerColumn struct {
	SBaseWidthColumn

	// Is this column an autoincrement colmn
	IsAutoIncrement bool

	// Is this column is a version column for this records
	IsAutoVersion bool

	// Is this column a unsigned integer?
	IsUnsigned bool

	// If this column is an autoincrement column, AutoIncrementOffset records the initial offset
	AutoIncrementOffset int64
}

// IsNumeric implementation of SIntegerColumn for IColumnSpec
func (c *SIntegerColumn) IsNumeric() bool {
	return true
}

// ExtraDefs implementation of SIntegerColumn for IColumnSpec
func (c *SIntegerColumn) ExtraDefs() string {
	if c.IsAutoIncrement {
		return "AUTO_INCREMENT"
	}
	return ""
}

// DefinitionString implementation of SIntegerColumn for IColumnSpec
func (c *SIntegerColumn) DefinitionString() string {
	buf := definitionBuffer(c)
	return buf.String()
}

// IsZero implementation of SIntegerColumn for IColumnSpec
func (c *SIntegerColumn) IsZero(val interface{}) bool {
	if val == nil || (c.isPointer && reflect.ValueOf(val).IsNil()) {
		return true
	}
	switch intVal := val.(type) {
	case int8, int16, int32, int64, int, uint, uint8, uint16, uint32, uint64:
		return intVal == 0
	}
	return true
}

// ColType implementation of SIntegerColumn for IColumnSpec
func (c *SIntegerColumn) ColType() string {
	str := (&c.SBaseWidthColumn).ColType()
	if c.IsUnsigned {
		str += " UNSIGNED"
	}
	return str
}

// NewIntegerColumn return an instance of SIntegerColumn
func NewIntegerColumn(name string, sqltype string, unsigned bool, tagmap map[string]string, isPointer bool) SIntegerColumn {
	autoinc := false
	autoincBase := int64(0)
	tagmap, v, ok := utils.TagPop(tagmap, TAG_AUTOINCREMENT)
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
	tagmap, v, ok = utils.TagPop(tagmap, TAG_AUTOVERSION)
	if ok {
		autover = utils.ToBool(v)
	}
	c := SIntegerColumn{
		SBaseWidthColumn:    NewBaseWidthColumn(name, sqltype, tagmap, isPointer),
		IsAutoIncrement:     autoinc,
		AutoIncrementOffset: autoincBase,
		IsAutoVersion:       autover,
		IsUnsigned:          unsigned,
	}
	if autoinc {
		c.isPrimary = true // autoincrement column must be primary key
		c.isNullable = false
		c.IsAutoVersion = false
	} else if autover {
		c.isPrimary = false
		c.isNullable = false
		if len(c.defaultString) == 0 {
			c.defaultString = "0"
		}
	}
	return c
}

// SFloatColumn represents a float type column, e.g. float32 or float64
type SFloatColumn struct {
	SBaseColumn
}

// IsNumeric implementation of SFloatColumn for IColumnSpec
func (c *SFloatColumn) IsNumeric() bool {
	return true
}

// DefinitionString implementation of SFloatColumn for IColumnSpec
func (c *SFloatColumn) DefinitionString() string {
	buf := definitionBuffer(c)
	return buf.String()
}

// IsZero implementation of SFloatColumn for IColumnSpec
func (c *SFloatColumn) IsZero(val interface{}) bool {
	if c.isPointer {
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

// NewFloatColumn returns an instance of SFloatColumn
func NewFloatColumn(name string, sqlType string, tagmap map[string]string, isPointer bool) SFloatColumn {
	return SFloatColumn{SBaseColumn: NewBaseColumn(name, sqlType, tagmap, isPointer)}
}

// SDecimalColumn represents a DECIMAL type of column, i.e. a float with fixed width of digits
type SDecimalColumn struct {
	SBaseWidthColumn
	Precision int
}

// ColType implementation of SDecimalColumn for IColumnSpec
func (c *SDecimalColumn) ColType() string {
	return fmt.Sprintf("%s(%d, %d)", c.sqlType, c.width, c.Precision)
}

// IsNumeric implementation of SDecimalColumn for IColumnSpec
func (c *SDecimalColumn) IsNumeric() bool {
	return true
}

// DefinitionString implementation of SDecimalColumn for IColumnSpec
func (c *SDecimalColumn) DefinitionString() string {
	buf := definitionBuffer(c)
	return buf.String()
}

// IsZero implementation of SDecimalColumn for IColumnSpec
func (c *SDecimalColumn) IsZero(val interface{}) bool {
	if c.isPointer {
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

// NewDecimalColumn returns an instance of SDecimalColumn
func NewDecimalColumn(name string, tagmap map[string]string, isPointer bool) SDecimalColumn {
	tagmap, v, ok := utils.TagPop(tagmap, TAG_PRECISION)
	if !ok {
		panic(fmt.Sprintf("Field %q of float misses precision tag", name))
	}
	prec, err := strconv.Atoi(v)
	if err != nil {
		panic(fmt.Sprintf("Field precision of %q shoud be integer (%q)", name, v))
	}
	return SDecimalColumn{
		SBaseWidthColumn: NewBaseWidthColumn(name, "DECIMAL", tagmap, isPointer),
		Precision:        prec,
	}
}

// STextColumn represents a text type of column
type STextColumn struct {
	SBaseWidthColumn
	Charset string
}

// IsSupportDefault implementation of STextColumn for IColumnSpec
func (c *STextColumn) IsSupportDefault() bool {
	// https://stackoverflow.com/questions/3466872/why-cant-a-text-column-have-a-default-value-in-mysql
	// MySQL does not support default for TEXT/BLOB
	if c.sqlType == "VARCHAR" {
		return true
	}
	return false
}

// ColType implementation of STextColumn for IColumnSpec
func (c *STextColumn) ColType() string {
	var charset string
	var collate string
	switch c.Charset {
	case "ascii":
		charset = "ascii"
		collate = "ascii_general_ci"
	default:
		charset = "utf8mb4"
		collate = "utf8mb4_unicode_ci"
	}
	return fmt.Sprintf("%s CHARACTER SET '%s' COLLATE '%s'", c.SBaseWidthColumn.ColType(), charset, collate)
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
	if c.Charset == "ascii" {
		return true
	}
	return false
}

// DefinitionString implementation of STextColumn for IColumnSpec
func (c *STextColumn) DefinitionString() string {
	buf := definitionBuffer(c)
	return buf.String()
}

// IsZero implementation of STextColumn for IColumnSpec
func (c *STextColumn) IsZero(val interface{}) bool {
	if c.isPointer {
		return gotypes.IsNil(val)
	}
	return reflect.ValueOf(val).Len() == 0
}

// NewTextColumn return an instance of STextColumn
func NewTextColumn(name string, tagmap map[string]string, isPointer bool) STextColumn {
	var width int
	var sqltype string
	widthStr, _ := tagmap[TAG_WIDTH]
	if len(widthStr) > 0 && regutils.MatchInteger(widthStr) {
		width, _ = strconv.Atoi(widthStr)
	}
	tagmap, txtLen, _ := utils.TagPop(tagmap, TAG_TEXT_LENGTH)
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
	tagmap, charset, _ := utils.TagPop(tagmap, TAG_CHARSET)
	if len(charset) == 0 {
		charset = "utf8"
	} else if charset != "utf8" && charset != "ascii" {
		panic(fmt.Sprintf("Unsupported charset %s for %s", charset, name))
	}
	return STextColumn{
		SBaseWidthColumn: NewBaseWidthColumn(name, sqltype, tagmap, isPointer),
		Charset:          charset,
	}
}

/*type SStringColumn struct {
	STextColumn
}

func (c *SStringColumn) DefinitionString() string {
	buf := definitionBuffer(c)
	return buf.String()
}

func NewStringColumn(name string, sqltype string, tagmap map[string]string) SStringColumn {
	sc := SStringColumn{STextColumn: NewTextColumn(name, sqltype, tagmap)}
	// if sc.width > 768 {
	//	log.Fatalf("Field %s width %d too with(>768)", name, sc.width)
	// }
	return sc
}*/

// STimeTypeColumn represents a Detetime type of column, e.g. DateTime
type STimeTypeColumn struct {
	SBaseColumn
}

// IsText implementation of STimeTypeColumn for IColumnSpec
func (c *STimeTypeColumn) IsText() bool {
	return true
}

// DefinitionString implementation of STimeTypeColumn for IColumnSpec
func (c *STimeTypeColumn) DefinitionString() string {
	buf := definitionBuffer(c)
	return buf.String()
}

// IsZero implementation of STimeTypeColumn for IColumnSpec
func (c *STimeTypeColumn) IsZero(val interface{}) bool {
	if c.isPointer {
		bVal := val.(*time.Time)
		return bVal == nil
	}
	bVal := val.(time.Time)
	return bVal.IsZero()
}

// NewTimeTypeColumn return an instance of STimeTypeColumn
func NewTimeTypeColumn(name string, typeStr string, tagmap map[string]string, isPointer bool) STimeTypeColumn {
	dc := STimeTypeColumn{
		NewBaseColumn(name, typeStr, tagmap, isPointer),
	}
	return dc
}

// SDateTimeColumn represents a DateTime type of column
type SDateTimeColumn struct {
	STimeTypeColumn

	// Is this column a 'created_at' field, whichi records the time of create this record
	IsCreatedAt bool

	// Is this column a 'updated_at' field, whichi records the time when this record was updated
	IsUpdatedAt bool
}

// NewDateTimeColumn returns an instance of DateTime column
func NewDateTimeColumn(name string, tagmap map[string]string, isPointer bool) SDateTimeColumn {
	createdAt := false
	updatedAt := false
	tagmap, v, ok := utils.TagPop(tagmap, TAG_CREATE_TIMESTAMP)
	if ok {
		createdAt = utils.ToBool(v)
	}
	tagmap, v, ok = utils.TagPop(tagmap, TAG_UPDATE_TIMESTAMP)
	if ok {
		updatedAt = utils.ToBool(v)
	}
	dtc := SDateTimeColumn{
		NewTimeTypeColumn(name, "DATETIME", tagmap, isPointer),
		createdAt, updatedAt,
	}
	return dtc
}

// CompoundColumn represents a column of compound tye, e.g. a JSON, an Array, or a struct
type CompoundColumn struct {
	STextColumn
}

// DefinitionString implementation of CompoundColumn for IColumnSpec
func (c *CompoundColumn) DefinitionString() string {
	buf := definitionBuffer(c)
	return buf.String()
}

// IsZero implementation of CompoundColumn for IColumnSpec
func (c *CompoundColumn) IsZero(val interface{}) bool {
	if val == nil {
		return true
	}
	if c.isPointer && reflect.ValueOf(val).IsNil() {
		return true
	}
	return false
}

// ConvertFromValue implementation of CompoundColumn for IColumnSpec
func (c *CompoundColumn) ConvertFromValue(val interface{}) interface{} {
	bVal, ok := val.(gotypes.ISerializable)
	if ok && bVal != nil {
		return bVal.String()
	}
	return ""
}

// NewCompoundColumn returns an instance of CompoundColumn
func NewCompoundColumn(name string, tagmap map[string]string, isPointer bool) CompoundColumn {
	dtc := CompoundColumn{NewTextColumn(name, tagmap, isPointer)}
	return dtc
}
