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
)

type SFunctionField struct {
	fields   []IQueryField
	function string
	alias    string
}

func (ff *SFunctionField) Expression() string {
	fieldRefs := make([]interface{}, 0)
	for _, f := range ff.fields {
		fieldRefs = append(fieldRefs, f.Reference())
	}
	return fmt.Sprintf("%s AS %s", fmt.Sprintf(ff.function, fieldRefs...), ff.Name())
}

func (ff *SFunctionField) Name() string {
	return ff.alias
}

func (ff *SFunctionField) Reference() string {
	return ff.alias
}

func (ff *SFunctionField) Label(label string) IQueryField {
	if len(label) > 0 && label != ff.alias {
		ff.alias = label
	}
	return ff
}

func NewFunctionField(name string, funcexp string, fields ...IQueryField) SFunctionField {
	ff := SFunctionField{function: funcexp, alias: name, fields: fields}
	return ff
}

func COUNT(name string, field ...IQueryField) IQueryField {
	var expr string
	if len(field) == 0 {
		expr = "COUNT(*)"
	} else {
		expr = "COUNT(%s)"
	}
	ff := NewFunctionField(name, expr, field...)
	return &ff
}

func MAX(name string, field IQueryField) IQueryField {
	ff := NewFunctionField(name, "MAX(%s)", field)
	return &ff
}

func SUM(name string, field IQueryField) IQueryField {
	ff := NewFunctionField(name, "SUM(%s)", field)
	return &ff
}

func DISTINCT(name string, field IQueryField) IQueryField {
	ff := NewFunctionField(name, "DISTINCT(%s)", field)
	return &ff
}

func GROUP_CONCAT(name string, field IQueryField) IQueryField {
	ff := NewFunctionField(name, "GROUP_CONCAT(%s)", field)
	return &ff
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
	ff := NewFunctionField(name, `CONCAT(`+strings.Join(params, ",")+`)`, fields...)
	return &ff
}
