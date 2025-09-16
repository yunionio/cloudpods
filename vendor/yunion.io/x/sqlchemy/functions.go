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
	"strings"
)

// IFunctionQueryField is a special type of field that is an aggregate function
type IFunctionQueryField interface {
	IQueryField

	// is this an aggregate function?
	IsAggregate() bool
}

// IFunction is the interface for a SQL embedded function, such as MIN, MAX, NOW, etc.
type IFunction interface {
	expression() string
	variables() []interface{}
	database() *SDatabase
	queryFields() []IQueryField
}

// NewFunction creates a field with SQL function
// for example: SUM(count) as total
func NewFunction(ifunc IFunction, name string, isAggre bool) IQueryField {
	return &SFunctionFieldBase{
		IFunction: ifunc,
		alias:     name,
		aggregate: isAggre,
	}
}

// SFunctionFieldBase is a query field that is the result of a SQL embedded function, e.g. COUNT(*) as count
type SFunctionFieldBase struct {
	IFunction
	alias string

	aggregate bool

	convertFunc func(interface{}) interface{}
}

func (ff *SFunctionFieldBase) getQuoteChar() string {
	qChar := ""
	if ff.database() != nil {
		qChar = ff.database().backend.QuoteChar()
	}
	return qChar
}

// Reference implementation of SFunctionFieldBase for IQueryField
func (ff *SFunctionFieldBase) Reference() string {
	if len(ff.alias) == 0 {
		// log.Warningf("reference a function field without alias! %s", ff.expression())
		return ff.expression()
	}
	qChar := ff.getQuoteChar()
	return fmt.Sprintf("%s%s%s", qChar, ff.alias, qChar)
}

// Expression implementation of SFunctionFieldBase for IQueryField
func (ff *SFunctionFieldBase) Expression() string {
	return ff.expression()
}

// Name implementation of SFunctionFieldBase for IQueryField
func (ff *SFunctionFieldBase) Name() string {
	if len(ff.alias) > 0 {
		return ff.alias
	}
	return ff.expression()
}

// Label implementation of SFunctionFieldBase for IQueryField
func (ff *SFunctionFieldBase) Label(label string) IQueryField {
	if len(label) > 0 {
		ff.alias = label
	}
	return ff
}

// Variables implementation of SFunctionFieldBase for IQueryField
func (ff *SFunctionFieldBase) Variables() []interface{} {
	return ff.variables()
}

// ConvertFromValue implementation of SFunctionFieldBase for IQueryField
func (ff *SFunctionFieldBase) ConvertFromValue(val interface{}) interface{} {
	if ff.convertFunc != nil {
		return ff.convertFunc(val)
	}
	return val
}

func (ff *SFunctionFieldBase) IsAggregate() bool {
	return ff.aggregate
}

type sExprFunction struct {
	fields   []IQueryField
	function string
}

func (ff *sExprFunction) expression() string {
	fieldRefs := make([]interface{}, 0)
	for _, f := range ff.fields {
		fieldRefs = append(fieldRefs, f.Expression())
	}
	return fmt.Sprintf(ff.function, fieldRefs...)
}

func (ff *sExprFunction) variables() []interface{} {
	vars := make([]interface{}, 0)
	for _, f := range ff.fields {
		fromVars := f.Variables()
		vars = append(vars, fromVars...)
	}
	return vars
}

func (ff *sExprFunction) database() *SDatabase {
	for i := range ff.fields {
		db := ff.fields[i].database()
		if db != nil {
			return db
		}
	}
	// if ff.function != "COUNT(*)" {
	// 	log.Debugf("no fields function? %s", ff.expression())
	// }
	return nil
}

func (ff *sExprFunction) queryFields() []IQueryField {
	return ff.fields
}

func NewFunctionField(name string, isAggr bool, funcexp string, fields ...IQueryField) IQueryField {
	return NewFunctionFieldWithConvert(name, isAggr, funcexp, nil, fields...)
}

// NewFunctionField returns an instance of query field by calling a SQL embedded function
func NewFunctionFieldWithConvert(name string, isAggr bool, funcexp string, convertFunc func(interface{}) interface{}, fields ...IQueryField) IQueryField {
	funcBase := &sExprFunction{
		fields:   fields,
		function: funcexp,
	}
	return &SFunctionFieldBase{
		IFunction:   funcBase,
		alias:       name,
		aggregate:   isAggr,
		convertFunc: convertFunc,
	}
}

// COUNT represents the SQL function COUNT
func COUNT(name string, field ...IQueryField) IQueryField {
	return getFieldBackend(field...).COUNT(name, field...)
}

// MAX represents the SQL function MAX
func MAX(name string, field IQueryField) IQueryField {
	return getFieldBackend(field).MAX(name, field)
}

// MIN represents the SQL function MIN
func MIN(name string, field IQueryField) IQueryField {
	return getFieldBackend(field).MIN(name, field)
}

// SUM represents the SQL function SUM
func SUM(name string, field IQueryField) IQueryField {
	return getFieldBackend(field).SUM(name, field)
}

// AVG represents the SQL function SUM
func AVG(name string, field IQueryField) IQueryField {
	return getFieldBackend(field).AVG(name, field)
}

// LOWER represents the SQL function SUM
func LOWER(name string, field IQueryField) IQueryField {
	return getFieldBackend(field).LOWER(name, field)
}

// UPPER represents the SQL function SUM
func UPPER(name string, field IQueryField) IQueryField {
	return getFieldBackend(field).UPPER(name, field)
}

// DISTINCT represents the SQL function DISTINCT
func DISTINCT(name string, field IQueryField) IQueryField {
	return getFieldBackend(field).DISTINCT(name, field)
}

// GROUP_CONCAT represents the SQL function GROUP_CONCAT
func GROUP_CONCAT(name string, field IQueryField) IQueryField {
	return GROUP_CONCAT2(name, ",", field)
}

// GROUP_CONCAT2 represents the SQL function GROUP_CONCAT
func GROUP_CONCAT2(name string, sep string, field IQueryField) IQueryField {
	// return NewFunctionField(name, "GROUP_CONCAT(%s)", field)
	return getFieldBackend(field).GROUP_CONCAT2(name, sep, field)
}

// REPLACE represents the SQL function REPLACE
func REPLACE(name string, field IQueryField, old string, new string) IQueryField {
	return getFieldBackend(field).REPLACE(name, field, old, new)
}

// SConstField is a query field of a constant
type SConstField struct {
	constVar interface{}
	alias    string
}

// Expression implementation of SConstField for IQueryField
func (s *SConstField) Expression() string {
	return s.Reference()
}

// Name implementation of SConstField for IQueryField
func (s *SConstField) Name() string {
	return s.alias
}

// Reference implementation of SConstField for IQueryField
func (s *SConstField) Reference() string {
	return getQuoteStringValue(s.constVar)
}

// Label implementation of SConstField for IQueryField
func (s *SConstField) Label(label string) IQueryField {
	if len(label) > 0 {
		s.alias = label
	}
	return s
}

// ConvertFromValue implementation of SConstField for IQueryField
func (s *SConstField) ConvertFromValue(val interface{}) interface{} {
	return val
}

// database implementation of SConstField for IQueryField
func (s *SConstField) database() *SDatabase {
	return nil
}

// Variables implementation of SConstField for IQueryField
func (s *SConstField) Variables() []interface{} {
	return nil
}

// IsAggregate implementation of SConstField for IFunctionQueryField
func (s *SConstField) IsAggregate() bool {
	return true
}

// NewConstField returns an instance of SConstField
func NewConstField(variable interface{}) *SConstField {
	return &SConstField{constVar: variable}
}

// SStringField is a query field of a string constant
type SStringField struct {
	strConst string
	alias    string
}

// Expression implementation of SStringField for IQueryField
func (s *SStringField) Expression() string {
	return s.Reference()
}

// Name implementation of SStringField for IQueryField
func (s *SStringField) Name() string {
	return s.alias
}

// Reference implementation of SStringField for IQueryField
func (s *SStringField) Reference() string {
	return getQuoteStringValue(s.strConst)
}

// Label implementation of SStringField for IQueryField
func (s *SStringField) Label(label string) IQueryField {
	if len(label) > 0 {
		s.alias = label
	}
	return s
}

// ConvertFromValue implementation of SStringField for IQueryField
func (s *SStringField) ConvertFromValue(val interface{}) interface{} {
	return val
}

// database implementation of SStringField for IQueryField
func (s *SStringField) database() *SDatabase {
	return nil
}

// Variables implementation of SStringField for IQueryField
func (s *SStringField) Variables() []interface{} {
	return nil
}

// IsAggregate implementation of SStringField for IFunctionQueryField
func (s *SStringField) IsAggregate() bool {
	return true
}

// NewStringField returns an instance of SStringField
func NewStringField(strConst string) *SStringField {
	return &SStringField{strConst: strConst}
}

// CONCAT represents a SQL function CONCAT
func CONCAT(name string, fields ...IQueryField) IQueryField {
	return getFieldBackend(fields...).CONCAT(name, fields...)
}

// SubStr represents a SQL function SUBSTR
// Deprecated
func SubStr(name string, field IQueryField, pos, length int) IQueryField {
	return SUBSTR(name, field, pos, length)
}

// SUBSTR represents a SQL function SUBSTR
func SUBSTR(name string, field IQueryField, pos, length int) IQueryField {
	return getFieldBackend(field).SUBSTR(name, field, pos, length)
}

// OR_Val represents a SQL function that does binary | operation on a field
func OR_Val(name string, field IQueryField, v interface{}) IQueryField {
	return getFieldBackend(field).OR_Val(name, field, v)
}

// AND_Val represents a SQL function that does binary & operation on a field
func AND_Val(name string, field IQueryField, v interface{}) IQueryField {
	return getFieldBackend(field).AND_Val(name, field, v)
}

// INET_ATON represents a SQL function INET_ATON
func INET_ATON(field IQueryField) IQueryField {
	return getFieldBackend(field).INET_ATON(field)
}

// TimestampAdd represents a SQL function TimestampAdd
func TimestampAdd(name string, field IQueryField, offsetSeconds int) IQueryField {
	return TIMESTAMPADD(name, field, offsetSeconds)
}

// TIMESTAMPADD represents a SQL function TimestampAdd
func TIMESTAMPADD(name string, field IQueryField, offsetSeconds int) IQueryField {
	return getFieldBackend(field).TIMESTAMPADD(name, field, offsetSeconds)
}

// DATE_FORMAT represents a SQL function DATE_FORMAT
func DATE_FORMAT(name string, field IQueryField, format string) IQueryField {
	return getFieldBackend(field).DATE_FORMAT(name, field, format)
}

// CAST represents a SQL function cast types
func CAST(field IQueryField, typeStr string, fieldname string) IQueryField {
	return getFieldBackend(field).CAST(field, typeStr, fieldname)
}

// CASTString represents a SQL function cast any type to String
func CASTString(field IQueryField, fieldname string) IQueryField {
	return getFieldBackend(field).CASTString(field, fieldname)
}

// CASTInt represents a SQL function cast any type to Integer
func CASTInt(field IQueryField, fieldname string) IQueryField {
	return getFieldBackend(field).CASTInt(field, fieldname)
}

// CASTFloat represents a SQL function cast any type to Float
func CASTFloat(field IQueryField, fieldname string) IQueryField {
	return getFieldBackend(field).CASTFloat(field, fieldname)
}

// LENGTH represents a SQL function of LENGTH
func LENGTH(name string, field IQueryField) IQueryField {
	return getFieldBackend(field).LENGTH(name, field)
}

func bc(name, op string, fields ...IQueryField) IQueryField {
	exps := []string{}
	for i := 0; i < len(fields); i++ {
		exps = append(exps, "%s")
	}
	return NewFunctionField(name, false, strings.Join(exps, fmt.Sprintf(" %s ", op)), fields...)
}

func ADD(name string, fields ...IQueryField) IQueryField {
	return bc(name, "+", fields...)
}

func SUB(name string, fields ...IQueryField) IQueryField {
	return bc(name, "-", fields...)
}

func MUL(name string, fields ...IQueryField) IQueryField {
	return bc(name, "*", fields...)
}

func DIV(name string, fields ...IQueryField) IQueryField {
	return bc(name, "/", fields...)
}

func DATEDIFF(unit string, field1, field2 IQueryField) IQueryField {
	return getFieldBackend(field1).DATEDIFF(unit, field1, field2)
}

func ABS(name string, field IQueryField) IQueryField {
	return NewFunctionField(name, false, "ABS(%s)", field)
}
