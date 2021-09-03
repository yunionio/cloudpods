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
	"strings"

	"yunion.io/x/log"
)

// IFunction is the interface for a SQL embedded function, such as MIN, MAX, NOW, etc.
type IFunction interface {
	expression() string
	variables() []interface{}
}

// SFunctionFieldBase is a query field that is the result of a SQL embedded function, e.g. COUNT(*) as count
type SFunctionFieldBase struct {
	IFunction
	alias string
}

// Reference implementation of SFunctionFieldBase for IQueryField
func (ff *SFunctionFieldBase) Reference() string {
	if len(ff.alias) == 0 {
		log.Warningf("reference a function field without alias! %s", ff.expression())
		return ff.expression()
	}
	return fmt.Sprintf("`%s`", ff.alias)
}

// Expression implementation of SFunctionFieldBase for IQueryField
func (ff *SFunctionFieldBase) Expression() string {
	if len(ff.alias) > 0 {
		// add alias
		return fmt.Sprintf("%s AS `%s`", ff.expression(), ff.alias)
	}
	// no alias
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
	if len(label) > 0 && label != ff.alias {
		ff.alias = label
	}
	return ff
}

// Variables implementation of SFunctionFieldBase for IQueryField
func (ff *SFunctionFieldBase) Variables() []interface{} {
	return ff.variables()
}

type sExprFunction struct {
	fields   []IQueryField
	function string
}

func (ff *sExprFunction) expression() string {
	fieldRefs := make([]interface{}, 0)
	for _, f := range ff.fields {
		fieldRefs = append(fieldRefs, f.Reference())
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

// NewFunctionField returns an instance of query field by calling a SQL embedded function
func NewFunctionField(name string, funcexp string, fields ...IQueryField) IQueryField {
	funcBase := &sExprFunction{
		fields:   fields,
		function: funcexp,
	}
	return &SFunctionFieldBase{
		IFunction: funcBase,
		alias:     name,
	}
}

// COUNT represents the SQL function COUNT
func COUNT(name string, field ...IQueryField) IQueryField {
	var expr string
	if len(field) == 0 {
		expr = "COUNT(*)"
	} else {
		expr = "COUNT(%s)"
	}
	return NewFunctionField(name, expr, field...)
}

// MAX represents the SQL function MAX
func MAX(name string, field IQueryField) IQueryField {
	return NewFunctionField(name, "MAX(%s)", field)
}

// MIN represents the SQL function MIN
func MIN(name string, field IQueryField) IQueryField {
	return NewFunctionField(name, "MIN(%s)", field)
}

// SUM represents the SQL function SUM
func SUM(name string, field IQueryField) IQueryField {
	return NewFunctionField(name, "SUM(%s)", field)
}

// DISTINCT represents the SQL function DISTINCT
func DISTINCT(name string, field IQueryField) IQueryField {
	return NewFunctionField(name, "DISTINCT(%s)", field)
}

// GROUP_CONCAT represents the SQL function GROUP_CONCAT
func GROUP_CONCAT(name string, field IQueryField) IQueryField {
	return NewFunctionField(name, "GROUP_CONCAT(%s)", field)
}

// REPLACE represents the SQL function REPLACE
func REPLACE(name string, field IQueryField, old string, new string) IQueryField {
	return NewFunctionField(name, fmt.Sprintf(`REPLACE(%s, "%s", "%s")`, "%s", old, new), field)
}

// SConstField is a query field of a constant
type SConstField struct {
	constVar interface{}
	alias    string
}

// Expression implementation of SConstField for IQueryField
func (s *SConstField) Expression() string {
	return fmt.Sprintf("%s AS `%s`", s.Reference(), s.Name())
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

// Variables implementation of SConstField for IQueryField
func (s *SConstField) Variables() []interface{} {
	return nil
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
	return fmt.Sprintf("%s AS `%s`", s.Reference(), s.Name())
}

// Name implementation of SStringField for IQueryField
func (s *SStringField) Name() string {
	return s.alias
}

// Reference implementation of SStringField for IQueryField
func (s *SStringField) Reference() string {
	return strconv.Quote(s.strConst)
}

// Label implementation of SStringField for IQueryField
func (s *SStringField) Label(label string) IQueryField {
	if len(label) > 0 {
		s.alias = label
	}
	return s
}

// Variables implementation of SStringField for IQueryField
func (s *SStringField) Variables() []interface{} {
	return nil
}

// NewStringField returns an instance of SStringField
func NewStringField(name string) *SStringField {
	return &SStringField{strConst: name}
}

// CONCAT represents a SQL function CONCAT
func CONCAT(name string, fields ...IQueryField) IQueryField {
	params := []string{}
	for i := 0; i < len(fields); i++ {
		params = append(params, "%s")
	}
	return NewFunctionField(name, `CONCAT(`+strings.Join(params, ",")+`)`, fields...)
}

// SubStr represents a SQL function SUBSTR
func SubStr(name string, field IQueryField, pos, length int) IQueryField {
	var rightStr string
	if length <= 0 {
		rightStr = fmt.Sprintf("%d)", pos)
	} else {
		rightStr = fmt.Sprintf("%d, %d)", pos, length)
	}
	return NewFunctionField(name, `SUBSTR(%s, `+rightStr, field)
}

// OR_Val represents a SQL function that does binary | operation on a field
func OR_Val(name string, field IQueryField, v interface{}) IQueryField {
	rightStr := fmt.Sprintf("|%v", v)
	return NewFunctionField(name, "%s"+rightStr, field)
}

// AND_Val represents a SQL function that does binary & operation on a field
func AND_Val(name string, field IQueryField, v interface{}) IQueryField {
	rightStr := fmt.Sprintf("&%v", v)
	return NewFunctionField(name, "%s"+rightStr, field)
}

// INET_ATON represents a SQL function INET_ATON
func INET_ATON(field IQueryField) IQueryField {
	return NewFunctionField("", `INET_ATON(%s)`, field)
}

// TimestampAdd represents a SQL function TimestampAdd
func TimestampAdd(name string, field IQueryField, offsetSeconds int) IQueryField {
	return NewFunctionField(name, `TIMESTAMPADD(SECOND, `+fmt.Sprintf("%d", offsetSeconds)+`, %s)`, field)
}
