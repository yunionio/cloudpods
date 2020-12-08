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

type IFunction interface {
	expression() string
}

type SFunctionFieldBase struct {
	IFunction
	alias string
}

func (ff *SFunctionFieldBase) Reference() string {
	if len(ff.alias) == 0 {
		log.Warningf("reference a function field without alias! %s", ff.expression())
		return ff.expression()
	} else {
		return fmt.Sprintf("`%s`", ff.alias)
	}
}

func (ff *SFunctionFieldBase) Expression() string {
	if len(ff.alias) > 0 {
		// add alias
		return fmt.Sprintf("%s AS `%s`", ff.expression(), ff.alias)
	} else {
		// no alias
		return ff.expression()
	}
}

func (ff *SFunctionFieldBase) Name() string {
	if len(ff.alias) > 0 {
		return ff.alias
	} else {
		return ff.expression()
	}
}

func (ff *SFunctionFieldBase) Label(label string) IQueryField {
	if len(label) > 0 && label != ff.alias {
		ff.alias = label
	}
	return ff
}

type SExprFunction struct {
	fields   []IQueryField
	function string
}

func (ff *SExprFunction) expression() string {
	fieldRefs := make([]interface{}, 0)
	for _, f := range ff.fields {
		fieldRefs = append(fieldRefs, f.Reference())
	}
	return fmt.Sprintf(ff.function, fieldRefs...)
}

func NewFunctionField(name string, funcexp string, fields ...IQueryField) IQueryField {
	funcBase := &SExprFunction{
		fields:   fields,
		function: funcexp,
	}
	return &SFunctionFieldBase{
		IFunction: funcBase,
		alias:     name,
	}
}

func COUNT(name string, field ...IQueryField) IQueryField {
	var expr string
	if len(field) == 0 {
		expr = "COUNT(*)"
	} else {
		expr = "COUNT(%s)"
	}
	return NewFunctionField(name, expr, field...)
}

func MAX(name string, field IQueryField) IQueryField {
	return NewFunctionField(name, "MAX(%s)", field)
}

func MIN(name string, field IQueryField) IQueryField {
	return NewFunctionField(name, "MIN(%s)", field)
}

func SUM(name string, field IQueryField) IQueryField {
	return NewFunctionField(name, "SUM(%s)", field)
}

func DISTINCT(name string, field IQueryField) IQueryField {
	return NewFunctionField(name, "DISTINCT(%s)", field)
}

func GROUP_CONCAT(name string, field IQueryField) IQueryField {
	return NewFunctionField(name, "GROUP_CONCAT(%s)", field)
}

func REPLACE(name string, field IQueryField, old string, new string) IQueryField {
	return NewFunctionField(name, fmt.Sprintf(`REPLACE(%s, "%s", "%s")`, "%s", old, new), field)
}

type SStringField struct {
	strConst string
	alias    string
}

func (s *SStringField) Expression() string {
	return fmt.Sprintf("%s AS `%s`", s.Reference(), s.Name())
}

func (s *SStringField) Name() string {
	return s.alias
}

func (s *SStringField) Reference() string {
	return strconv.Quote(s.strConst)
}

func (s *SStringField) Label(label string) IQueryField {
	if len(label) > 0 {
		s.alias = label
	}
	return s
}

func NewStringField(name string) *SStringField {
	return &SStringField{strConst: name}
}

func CONCAT(name string, fields ...IQueryField) IQueryField {
	params := []string{}
	for i := 0; i < len(fields); i++ {
		params = append(params, "%s")
	}
	return NewFunctionField(name, `CONCAT(`+strings.Join(params, ",")+`)`, fields...)
}

func SubStr(name string, field IQueryField, pos, length int) IQueryField {
	var rightStr string
	if length <= 0 {
		rightStr = fmt.Sprintf("%d)", pos)
	} else {
		rightStr = fmt.Sprintf("%d, %d)", pos, length)
	}
	return NewFunctionField(name, `SUBSTR(%s, `+rightStr, field)
}

func OR_Val(name string, field IQueryField, v interface{}) IQueryField {
	rightStr := fmt.Sprintf("|%v", v)
	return NewFunctionField(name, "%s"+rightStr, field)
}

func AND_Val(name string, field IQueryField, v interface{}) IQueryField {
	rightStr := fmt.Sprintf("&%v", v)
	return NewFunctionField(name, "%s"+rightStr, field)
}

func INET_ATON(field IQueryField) IQueryField {
	return NewFunctionField("", `INET_ATON(%s)`, field)
}

func TimestampAdd(name string, field IQueryField, offsetSeconds int) IQueryField {
	return NewFunctionField(name, `TIMESTAMPADD(SECOND, `+fmt.Sprintf("%d", offsetSeconds)+`, %s)`, field)
}
