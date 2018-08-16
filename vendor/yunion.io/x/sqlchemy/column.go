package sqlchemy

import (
	"bytes"
	"fmt"
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

func (c *SBaseColumn) IsSupportDefault() bool {
	return true
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
	if len(def) > 0 && c.IsSupportDefault() {
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

func (c *SBooleanColumn) ConvertFromValue(val interface{}) interface{} {
	bVal := val.(bool)
	if bVal {
		return 1
	} else {
		return 0
	}
}

func (c *SBooleanColumn) IsZero(val interface{}) bool {
	bVal := val.(bool)
	return bVal == false
}

func NewBooleanColumn(name string, tagmap map[string]string) SBooleanColumn {
	bc := SBooleanColumn{SBaseWidthColumn: NewBaseWidthColumn(name, "TINYINT", tagmap)}
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
	bVal := val.(tristate.TriState)
	return bVal == tristate.None
}

func NewTristateColumn(name string, tagmap map[string]string) STristateColumn {
	bc := STristateColumn{SBaseWidthColumn: NewBaseWidthColumn(name, "TINYINT", tagmap)}
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

func NewDecimalColumn(name string, tagmap map[string]string) SDecimalColumn {
	tagmap, v, ok := utils.TagPop(tagmap, TAG_PRECISION)
	if !ok {
		panic(fmt.Sprintf("Field %q of float misses precision tag", name))
	}
	prec, err := strconv.Atoi(v)
	if err != nil {
		panic(fmt.Sprintf("Field precision of %q shoud be integer (%q)", name, v))
	}
	return SDecimalColumn{SBaseWidthColumn: NewBaseWidthColumn(name, "DECIMAL", tagmap),
		Precision: prec}
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
	bVal := val.(string)
	return len(bVal) == 0
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
		panic(fmt.Sprintf("Unsupported charset %s for %s", charset, name))
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
	// log.Debugf("%s", val)
	json := val.(gotypes.ISerializable)
	return json.IsZero()
}

func (c *CompoundColumn) ConvertFromValue(val interface{}) interface{} {
	bVal, ok := val.(gotypes.ISerializable)
	if ok && bVal != nil {
		return bVal.String()
	} else {
		return ""
	}
}

func NewCompoundColumn(name string, tagmap map[string]string) CompoundColumn {
	dtc := CompoundColumn{NewTextColumn(name, tagmap)}
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
