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

type IColumnSpec interface {
	Name() string
	ColType() string
	Default() string
	IsSupportDefault() bool
	IsNullable() bool
	SetNullable(on bool)
	IsPrimary() bool
	IsUnique() bool
	IsIndex() bool
	ExtraDefs() string
	DefinitionString() string
	IsText() bool
	IsSearchable() bool
	IsAscii() bool
	IsNumeric() bool
	ConvertFromString(str string) string
	// ConvertToString(str string) string
	ConvertFromValue(val interface{}) interface{}
	// ConvertToValue(str interface{}) interface{}
	IsZero(val interface{}) bool
	AllowZero() bool
	// IsEqual(v1, v2 interface{}) bool
	Tags() map[string]string

	IsPointer() bool
}

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

func (c *SBaseColumn) IsPointer() bool {
	return c.isPointer
}

func (c *SBaseColumn) Name() string {
	if len(c.dbName) > 0 {
		return c.dbName
	} else {
		return c.name
	}
}

func (c *SBaseColumn) ColType() string {
	return c.sqlType
}

func (c *SBaseColumn) Default() string {
	return c.defaultString
}

func (c *SBaseColumn) IsSupportDefault() bool {
	return true
}

func (c *SBaseColumn) IsNullable() bool {
	return c.isNullable
}

func (c *SBaseColumn) SetNullable(on bool) {
	c.isNullable = on
}

func (c *SBaseColumn) IsPrimary() bool {
	return c.isPrimary
}

func (c *SBaseColumn) IsUnique() bool {
	return c.isUnique
}

func (c *SBaseColumn) IsIndex() bool {
	return c.isIndex
}

func (c *SBaseColumn) ExtraDefs() string {
	return ""
}

func (c *SBaseColumn) IsText() bool {
	return false
}

func (c *SBaseColumn) IsAscii() bool {
	return false
}

func (c *SBaseColumn) IsSearchable() bool {
	return false
}

func (c *SBaseColumn) IsNumeric() bool {
	return false
}

func (c *SBaseColumn) AllowZero() bool {
	return c.isAllowZero
}

func (c *SBaseColumn) ConvertFromString(str string) string {
	return str
}

func (c *SBaseColumn) ConvertToString(str string) string {
	return str
}

func (c *SBaseColumn) ConvertFromValue(val interface{}) interface{} {
	return val
}

func (c *SBaseColumn) ConvertToValue(val interface{}) interface{} {
	return val
}

func (c *SBaseColumn) Tags() map[string]string {
	return c.tags
}

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

type SBaseWidthColumn struct {
	SBaseColumn
	width int
}

func (c *SBaseWidthColumn) ColType() string {
	if c.width > 0 {
		return fmt.Sprintf("%s(%d)", c.sqlType, c.width)
	} else {
		return c.sqlType
	}
}

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

type SBooleanColumn struct {
	SBaseWidthColumn
}

func (c *SBooleanColumn) DefinitionString() string {
	buf := definitionBuffer(c)
	return buf.String()
}

func (c *SBooleanColumn) ConvertFromString(str string) string {
	switch strings.ToLower(str) {
	case "true", "yes", "on", "ok", "1":
		return "1"
	default:
		return "0"
	}
}

func (c *SBooleanColumn) ConvertFromValue(val interface{}) interface{} {
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
}

func (c *SBooleanColumn) IsZero(val interface{}) bool {
	if c.isPointer {
		bVal := val.(*bool)
		return bVal == nil
	} else {
		bVal := val.(bool)
		return bVal == false
	}
}

func NewBooleanColumn(name string, tagmap map[string]string, isPointer bool) SBooleanColumn {
	bc := SBooleanColumn{SBaseWidthColumn: NewBaseWidthColumn(name, "TINYINT", tagmap, isPointer)}
	if !bc.IsPointer() && len(bc.Default()) > 0 && bc.ConvertFromString(bc.Default()) == "1" {
		msg := fmt.Sprintf("Non-pointer boolean column should not default true: %s(%s)", name, tagmap)
		panic(msg)
	}
	return bc
}

type STristateColumn struct {
	SBaseWidthColumn
}

func (c *STristateColumn) DefinitionString() string {
	buf := definitionBuffer(c)
	return buf.String()
}

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

func (c *STristateColumn) ConvertFromValue(val interface{}) interface{} {
	bVal := val.(tristate.TriState)
	if bVal == tristate.True {
		return 1
	} else {
		return 0
	}
}

func (c *STristateColumn) IsZero(val interface{}) bool {
	if c.isPointer {
		bVal := val.(*tristate.TriState)
		return bVal == nil
	} else {
		bVal := val.(tristate.TriState)
		return bVal == tristate.None
	}
}

func NewTristateColumn(name string, tagmap map[string]string, isPointer bool) STristateColumn {
	bc := STristateColumn{SBaseWidthColumn: NewBaseWidthColumn(name, "TINYINT", tagmap, isPointer)}
	return bc
}

type SIntegerColumn struct {
	SBaseWidthColumn
	IsAutoIncrement bool
	IsAutoVersion   bool
	IsUnsigned      bool

	AutoIncrementOffset int64
}

func (c *SIntegerColumn) IsNumeric() bool {
	return true
}

func (c *SIntegerColumn) ExtraDefs() string {
	if c.IsAutoIncrement {
		return "AUTO_INCREMENT"
	}
	return ""
}

func (c *SIntegerColumn) DefinitionString() string {
	buf := definitionBuffer(c)
	return buf.String()
}

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

func (c *SIntegerColumn) ColType() string {
	str := (&c.SBaseWidthColumn).ColType()
	if c.IsUnsigned {
		str += " UNSIGNED"
	}
	return str
}

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

type SFloatColumn struct {
	SBaseColumn
}

func (c *SFloatColumn) IsNumeric() bool {
	return true
}

func (c *SFloatColumn) DefinitionString() string {
	buf := definitionBuffer(c)
	return buf.String()
}

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

func NewFloatColumn(name string, sqlType string, tagmap map[string]string, isPointer bool) SFloatColumn {
	return SFloatColumn{SBaseColumn: NewBaseColumn(name, sqlType, tagmap, isPointer)}
}

type SDecimalColumn struct {
	SBaseWidthColumn
	Precision int
}

func (c *SDecimalColumn) ColType() string {
	return fmt.Sprintf("%s(%d, %d)", c.sqlType, c.width, c.Precision)
}

func (c *SDecimalColumn) IsNumeric() bool {
	return true
}

func (c *SDecimalColumn) DefinitionString() string {
	buf := definitionBuffer(c)
	return buf.String()
}

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

type STextColumn struct {
	SBaseWidthColumn
	Charset string
}

func (c *STextColumn) IsSupportDefault() bool {
	// https://stackoverflow.com/questions/3466872/why-cant-a-text-column-have-a-default-value-in-mysql
	// MySQL does not support default for TEXT/BLOB
	if c.sqlType == "VARCHAR" {
		return true
	} else {
		return false
	}
}

func (c *STextColumn) ColType() string {
	return fmt.Sprintf("%s CHARACTER SET '%s'", c.SBaseWidthColumn.ColType(), c.Charset)
}

func (c *STextColumn) IsText() bool {
	return true
}

func (c *STextColumn) IsSearchable() bool {
	return true
}

func (c *STextColumn) IsAscii() bool {
	if c.Charset == "ascii" {
		return true
	} else {
		return false
	}
}

func (c *STextColumn) DefinitionString() string {
	buf := definitionBuffer(c)
	return buf.String()
}

func (c *STextColumn) IsZero(val interface{}) bool {
	if c.isPointer {
		return gotypes.IsNil(val)
	} else {
		return reflect.ValueOf(val).Len() == 0
	}
}

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

type STimeTypeColumn struct {
	SBaseColumn
}

func (c *STimeTypeColumn) IsText() bool {
	return true
}

func (c *STimeTypeColumn) DefinitionString() string {
	buf := definitionBuffer(c)
	return buf.String()
}

func (c *STimeTypeColumn) IsZero(val interface{}) bool {
	if c.isPointer {
		bVal := val.(*time.Time)
		return bVal == nil
	} else {
		bVal := val.(time.Time)
		return bVal.IsZero()
	}
}

func NewTimeTypeColumn(name string, typeStr string, tagmap map[string]string, isPointer bool) STimeTypeColumn {
	dc := STimeTypeColumn{
		NewBaseColumn(name, typeStr, tagmap, isPointer),
	}
	return dc
}

type SDateTimeColumn struct {
	STimeTypeColumn
	IsCreatedAt bool
	IsUpdatedAt bool
}

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

type CompoundColumn struct {
	STextColumn
}

func (c *CompoundColumn) DefinitionString() string {
	buf := definitionBuffer(c)
	return buf.String()
}

func (c *CompoundColumn) IsZero(val interface{}) bool {
	if val == nil {
		return true
	}
	if c.isPointer && reflect.ValueOf(val).IsNil() {
		return true
	}
	return false
}

func (c *CompoundColumn) ConvertFromValue(val interface{}) interface{} {
	bVal, ok := val.(gotypes.ISerializable)
	if ok && bVal != nil {
		return bVal.String()
	} else {
		return ""
	}
}

func NewCompoundColumn(name string, tagmap map[string]string, isPointer bool) CompoundColumn {
	dtc := CompoundColumn{NewTextColumn(name, tagmap, isPointer)}
	return dtc
}
