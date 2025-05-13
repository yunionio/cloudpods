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
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
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

	SetPrimary(on bool)

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

	GetWidth() int

	// ConvertFromString returns the SQL representation of a value in string format for this column
	ConvertFromString(str string) interface{}

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

	// IsAutoVersion
	IsAutoVersion() bool

	// IsUpdatedAt
	IsUpdatedAt() bool

	// IsCreatedAt
	IsCreatedAt() bool

	// IsAutoIncrement
	IsAutoIncrement() bool

	AutoIncrementOffset() int64

	SetAutoIncrement(val bool)

	SetAutoIncrementOffset(offset int64)

	IsString() bool

	IsDateTime() bool

	// index of column, to preserve the column position
	GetColIndex() int
	// setter of column index
	SetColIndex(idx int)
}

type iColumnInternal interface {
	IColumnSpec

	Oldname() string
}

// SBaseColumn is the base structure represents a column
type SBaseColumn struct {
	name          string
	dbName        string
	oldName       string
	sqlType       string
	defaultString string
	isPointer     bool
	isNullable    bool
	isPrimary     bool
	isUnique      bool
	isIndex       bool
	isAllowZero   bool
	tags          map[string]string
	colIndex      int
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

// Name implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) Oldname() string {
	return c.oldName
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

func (c *SBaseColumn) SetPrimary(on bool) {
	c.isPrimary = on
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
//func (c *SBaseColumn) ConvertFromString(str string) interface{} {
//	return str
//}

// ConvertFromValue implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) ConvertFromValue(val interface{}) interface{} {
	return val
}

// Tags implementation of SBaseColumn for IColumnSpec
func (c *SBaseColumn) Tags() map[string]string {
	return c.tags
}

func (c *SBaseColumn) IsAutoVersion() bool {
	return false
}

func (c *SBaseColumn) IsUpdatedAt() bool {
	return false
}

func (c *SBaseColumn) IsCreatedAt() bool {
	return false
}

func (c *SBaseColumn) IsAutoIncrement() bool {
	return false
}

func (c *SBaseColumn) AutoIncrementOffset() int64 {
	return 0
}

func (c *SBaseColumn) SetAutoIncrement(val bool) {
}

func (c *SBaseColumn) SetAutoIncrementOffset(offset int64) {
}

func (c *SBaseColumn) IsString() bool {
	return false
}

func (c *SBaseColumn) IsDateTime() bool {
	return false
}

func (c *SBaseColumn) GetColIndex() int {
	return c.colIndex
}

func (c *SBaseColumn) SetColIndex(idx int) {
	c.colIndex = idx
}

func (c *SBaseColumn) GetWidth() int {
	return 0
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
	oldName := ""
	tagmap, val, ok = utils.TagPop(tagmap, TAG_OLD_NAME)
	if ok {
		oldName = val
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
		oldName:       oldName,
		sqlType:       sqltype,
		defaultString: defStr,
		isNullable:    isNullable,
		isPrimary:     isPrimary,
		isUnique:      isUnique,
		isIndex:       isIndex,
		tags:          tagmap,
		isPointer:     isPointer,
		isAllowZero:   isAllowZero,
		colIndex:      -1,
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

func (c *SBaseWidthColumn) GetWidth() int {
	return c.width
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

type SBaseCompoundColumn struct{}

// ConvertFromString implementation of CompoundColumn for IColumnSpec
func (c *SBaseCompoundColumn) ConvertFromString(str string) interface{} {
	json, err := jsonutils.ParseString(str)
	if err != nil {
		json = jsonutils.JSONNull
	}
	return json.String()
}

// ConvertFromValue implementation of CompoundColumn for IColumnSpec
func (c *SBaseCompoundColumn) ConvertFromValue(val interface{}) interface{} {
	bVal, ok := val.(gotypes.ISerializable)
	if ok && bVal != nil {
		return bVal.String()
	}
	if _, ok := val.(string); ok {
		return val
	}
	return jsonutils.Marshal(val).String()
}
