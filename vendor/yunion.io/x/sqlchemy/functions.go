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
}

func (s *SStringField) Expression() string {
	return ""
}

func (s *SStringField) Name() string {
	return ""
}

func (s *SStringField) Reference() string {
	return strconv.Quote(s.strConst)
}

func (s *SStringField) Label(label string) IQueryField {
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
