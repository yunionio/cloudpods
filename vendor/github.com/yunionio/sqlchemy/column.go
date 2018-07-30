package sqlchemy

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/yunionio/log"
	"github.com/yunionio/pkg/gotypes"
	"github.com/yunionio/pkg/util/regutils"
	"github.com/yunionio/pkg/utils"
)

type IColumnSpec interface {
	Name() string
	ColType() string
	Default() string
	IsNullable() bool
	IsPrimary() bool
	IsKeyIndex() bool
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
	// IsEqual(v1, v2 interface{}) bool
	Tags() map[string]string
}

type SBaseColumn struct {
	name          string
	dbName        string
	sqlType       string
	defaultString string
	isNullable    bool
	isPrimary     bool
	isKeyIndex    bool
	isUnique      bool
	isIndex       bool
	tags          map[string]string
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

func (c *SBaseColumn) IsNullable() bool {
	return c.isNullable
}

func (c *SBaseColumn) IsPrimary() bool {
	return c.isPrimary
}

func (c *SBaseColumn) IsKeyIndex() bool {
	return c.isKeyIndex
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
	if len(def) > 0 {
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

func NewBaseColumn(name string, sqltype string, tagmap map[string]string) SBaseColumn {
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
	isKeyIndex := false
	tagmap, val, ok = utils.TagPop(tagmap, TAG_KEY_INDEX)
	if ok {
		isKeyIndex = utils.ToBool(val)
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
	return SBaseColumn{
		name:          name,
		dbName:        dbName,
		sqlType:       sqltype,
		defaultString: defStr,
		isNullable:    isNullable,
		isPrimary:     isPrimary,
		isKeyIndex:    isKeyIndex,
		isUnique:      isUnique,
		isIndex:       isIndex,
		tags:          tagmap,
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

func NewBaseWidthColumn(name string, sqltype string, tagmap map[string]string) SBaseWidthColumn {
	width := 0
	tagmap, v, ok := utils.TagPop(tagmap, TAG_WIDTH)
	if ok {
		width, _ = strconv.Atoi(v)
	}
	wc := SBaseWidthColumn{SBaseColumn: NewBaseColumn(name, sqltype, tagmap), width: width}
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

func (c *SBooleanColumn) ConvertToString(str string) string {
	if str == "0" {
		return "false"
	} else {
		return "true"
	}
}

func (c *SBooleanColumn) ConvertFromValue(val interface{}) interface{} {
	bVal := val.(bool)
	if bVal {
		return 1
	} else {
		return 0
	}
}

func (c *SBooleanColumn) ConvertToValue(val interface{}) interface{} {
	iVal := val.(int)
	if iVal == 0 {
		return false
	} else {
		return true
	}
}

func (c *SBooleanColumn) IsZero(val interface{}) bool {
	bVal := val.(bool)
	return bVal == false
}

func (c *SBooleanColumn) IsEqual(v1, v2 interface{}) bool {
	bVal1 := v1.(bool)
	bVal2 := v2.(bool)
	return bVal1 == bVal2
}

func NewBooleanColumn(name string, tagmap map[string]string) SBooleanColumn {
	bc := SBooleanColumn{SBaseWidthColumn: NewBaseWidthColumn(name, "TINYINT", tagmap)}
	return bc
}

type SIntegerColumn struct {
	SBaseWidthColumn
	IsAutoIncrement bool
	IsAutoVersion   bool
	IsUnsigned      bool
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
	switch val.(type) {
	case int8:
		return val.(int8) == 0
	case int16:
		return val.(int16) == 0
	case int32:
		return val.(int32) == 0
	case int64:
		return val.(int64) == 0
	case int:
		return val.(int) == 0
	case uint:
		return val.(uint) == 0
	case uint8:
		return val.(uint8) == 0
	case uint16:
		return val.(uint16) == 0
	case uint32:
		return val.(uint32) == 0
	case uint64:
		return val.(uint64) == 0
	}
	return true
}

func (c *SIntegerColumn) IsEqual(v1, v2 interface{}) bool {
	switch v1.(type) {
	case int:
		return v1.(int) == v2.(int)
	case int8:
		return v1.(int8) == v2.(int8)
	case int16:
		return v1.(int16) == v2.(int16)
	case int32:
		return v1.(int32) == v2.(int32)
	case int64:
		return v1.(int64) == v2.(int64)
	case uint:
		return v1.(uint) == v2.(uint)
	case uint8:
		return v1.(uint8) == v2.(uint8)
	case uint16:
		return v1.(uint16) == v2.(uint16)
	case uint32:
		return v1.(uint32) == v2.(uint32)
	case uint64:
		return v1.(uint64) == v2.(uint64)
	}
	return false
}

func (c *SIntegerColumn) ColType() string {
	str := (&c.SBaseWidthColumn).ColType()
	if c.IsUnsigned {
		str += " UNSIGNED"
	}
	return str
}

func NewIntegerColumn(name string, sqltype string, unsigned bool, tagmap map[string]string) SIntegerColumn {
	autoinc := false
	tagmap, v, ok := utils.TagPop(tagmap, TAG_AUTOINCREMENT)
	if ok {
		autoinc = utils.ToBool(v)
	}
	autover := false
	tagmap, v, ok = utils.TagPop(tagmap, TAG_AUTOVERSION)
	if ok {
		autover = utils.ToBool(v)
	}
	c := SIntegerColumn{SBaseWidthColumn: NewBaseWidthColumn(name, sqltype, tagmap),
		IsAutoIncrement: autoinc,
		IsAutoVersion:   autover,
		IsUnsigned:      unsigned,
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
	switch val.(type) {
	case float32:
		return val.(float32) == 0.0
	case float64:
		return val.(float64) == 0.0
	}
	return true
}

func (c *SFloatColumn) IsEqual(v1, v2 interface{}) bool {
	switch v1.(type) {
	case float32:
		return v1.(float32) == v2.(float32)
	case float64:
		return v1.(float64) == v2.(float64)
	}
	return false
}

func NewFloatColumn(name string, sqlType string, tagmap map[string]string) SFloatColumn {
	return SFloatColumn{SBaseColumn: NewBaseColumn(name, sqlType, tagmap)}
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
	switch val.(type) {
	case float32:
		return val.(float32) == 0.0
	case float64:
		return val.(float64) == 0.0
	}
	return true
}

func (c *SDecimalColumn) IsEqual(v1, v2 interface{}) bool {
	switch v1.(type) {
	case float32:
		return v1.(float32) == v2.(float32)
	case float64:
		return v1.(float64) == v2.(float64)
	}
	return false
}

func NewDecimalColumn(name string, tagmap map[string]string) SDecimalColumn {
	tagmap, v, ok := utils.TagPop(tagmap, TAG_PRECISION)
	if !ok {
		log.Fatalf("Field %s of float should have precision tag", name)
	}
	prec, err := strconv.Atoi(v)
	if err != nil {
		log.Fatalf("Field %s of float precision %s shoud be integer!", name, v)
	}
	return SDecimalColumn{SBaseWidthColumn: NewBaseWidthColumn(name, "DECIMAL", tagmap),
		Precision: prec}
}

type STextColumn struct {
	SBaseWidthColumn
	Charset string
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
	bVal := val.(string)
	return len(bVal) == 0
}

func (c *STextColumn) IsEqual(v1, v2 interface{}) bool {
	bVal1 := v1.(string)
	bVal2 := v2.(string)
	return bVal1 == bVal2
}

func NewTextColumn(name string, tagmap map[string]string) STextColumn {
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
		log.Fatalf("Unsupported charset %s for %s", charset, name)
	}
	return STextColumn{SBaseWidthColumn: NewBaseWidthColumn(name, sqltype, tagmap),
		Charset: charset}
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

type SDateTimeColumn struct {
	SBaseColumn
	IsCreatedAt bool
	IsUpdatedAt bool
}

func (c *SDateTimeColumn) IsText() bool {
	return true
}

func (c *SDateTimeColumn) DefinitionString() string {
	buf := definitionBuffer(c)
	return buf.String()
}

func (c *SDateTimeColumn) IsZero(val interface{}) bool {
	bVal := val.(time.Time)
	return bVal.IsZero()
}

func (c *SDateTimeColumn) IsEqual(v1, v2 interface{}) bool {
	bVal1 := v1.(time.Time)
	bVal2 := v2.(time.Time)
	return bVal1.Equal(bVal2)
}

func NewDateTimeColumn(name string, tagmap map[string]string) SDateTimeColumn {
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
	dtc := SDateTimeColumn{NewBaseColumn(name, "DATETIME", tagmap),
		createdAt, updatedAt}
	return dtc
}

type CompondColumn struct {
	STextColumn
}

func (c *CompondColumn) DefinitionString() string {
	buf := definitionBuffer(c)
	return buf.String()
}

/* func (c *CompondColumn) IsEqual(v1, v2 interface{}) bool {
	bVal1 := v1.(gotypes.ISerializable)
	bVal2 := v2.(gotypes.ISerializable)
	return bVal1.Equals(bVal2)
} */

func (c *CompondColumn) IsZero(val interface{}) bool {
	if val == nil {
		return true
	}
	// log.Debugf("%s", val)
	json := val.(gotypes.ISerializable)
	return json.IsZero()
}

func (c *CompondColumn) ConvertFromValue(val interface{}) interface{} {
	bVal, ok := val.(gotypes.ISerializable)
	if ok && bVal != nil {
		return bVal.String()
	} else {
		return ""
	}
}

func NewCompondColumn(name string, tagmap map[string]string) CompondColumn {
	dtc := CompondColumn{NewTextColumn(name, tagmap)}
	return dtc
}

/*type JSONDictColumn struct {
	JSONColumn
}

type JSONArrayColumn struct {
	JSONColumn
}

func (c *JSONDictColumn) ConvertFromValue(val interface{}) interface{} {
	bVal, ok := val.(*jsonutils.JSONDict)
	if ok && bVal != nil {
		return bVal.String()
	} else {
		return nil
	}
}

func (c *JSONDictColumn) ConvertToValue(val interface{}) interface{} {
	iVal, ok := val.(string)
	if ok && len(iVal) > 0 {
		json, err := jsonutils.ParseString(iVal)
		if err == nil {
			return json.(*jsonutils.JSONDict)
		}
	}
	return nil
}

func (c *JSONDictColumn) IsZero(val interface{}) bool {
	if val == nil {
		return true
	}
	dict := val.(*jsonutils.JSONDict)
	return dict.Size() == 0
}

func (c *JSONArrayColumn) ConvertFromValue(val interface{}) interface{} {
	bVal, ok := val.(*jsonutils.JSONArray)
	if ok && bVal != nil {
		return bVal.String()
	} else {
		return nil
	}
}

func (c *JSONArrayColumn) ConvertToValue(val interface{}) interface{} {
	iVal, ok := val.(string)
	if ok && len(iVal) > 0 {
		json, err := jsonutils.ParseString(iVal)
		if err == nil {
			return json.(*jsonutils.JSONArray)
		}
	}
	return nil
}

func (c *JSONArrayColumn) IsZero(val interface{}) bool {
	if val == nil {
		return true
	}
	dict := val.(*jsonutils.JSONArray)
	return dict.Size() == 0
}

func NewJSONDictColumn(name string, tagmap map[string]string) JSONDictColumn {
	dtc := JSONDictColumn{NewJSONColumn(name, tagmap)}
	return dtc
}

func NewJSONArrayColumn(name string, tagmap map[string]string) JSONArrayColumn {
	dtc := JSONArrayColumn{NewJSONColumn(name, tagmap)}
	return dtc
}*/
