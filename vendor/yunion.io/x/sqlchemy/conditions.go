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

	"yunion.io/x/pkg/util/reflectutils"
)

type ICondition interface {
	WhereClause() string
	Variables() []interface{}
}

type SCompoundConditions struct {
	conditions []ICondition
}

func compoundWhereClause(c *SCompoundConditions, op string) string {
	var buf bytes.Buffer
	for _, cond := range c.conditions {
		if buf.Len() > 0 {
			buf.WriteByte(' ')
			buf.WriteString(op)
			buf.WriteByte(' ')
		}
		buf.WriteByte('(')
		buf.WriteString(cond.WhereClause())
		buf.WriteByte(')')
	}
	return buf.String()
}

func (c *SCompoundConditions) WhereClause() string {
	return ""
}

func (c *SCompoundConditions) Variables() []interface{} {
	vars := make([]interface{}, 0)
	for _, cond := range c.conditions {
		nvars := cond.Variables()
		if len(nvars) > 0 {
			vars = append(vars, nvars...)
		}
	}
	return vars
}

type SAndConditions struct {
	SCompoundConditions
}

func (c *SAndConditions) WhereClause() string {
	return compoundWhereClause(&c.SCompoundConditions, SQL_OP_AND)
}

type SOrConditions struct {
	SCompoundConditions
}

func (c *SOrConditions) WhereClause() string {
	return compoundWhereClause(&c.SCompoundConditions, SQL_OP_OR)
}

func AND(cond ...ICondition) ICondition {
	conds := make([]ICondition, 0)
	for _, c := range cond {
		andCond, ok := c.(*SAndConditions)
		if ok {
			conds = append(conds, andCond.conditions...)
		} else {
			conds = append(conds, c)
		}
	}
	cc := SAndConditions{SCompoundConditions{conditions: conds}}
	return &cc
}

func OR(cond ...ICondition) ICondition {
	conds := make([]ICondition, 0)
	for _, c := range cond {
		orCond, ok := c.(*SOrConditions)
		if ok {
			conds = append(conds, orCond.conditions...)
		} else {
			conds = append(conds, c)
		}
	}
	cc := SOrConditions{SCompoundConditions{conditions: conds}}
	return &cc
}

type SNotCondition struct {
	condition ICondition
}

func (c *SNotCondition) WhereClause() string {
	return fmt.Sprintf("%s (%s)", SQL_OP_NOT, c.condition.WhereClause())
}

func (c *SNotCondition) Variables() []interface{} {
	return c.condition.Variables()
}

func NOT(cond ICondition) ICondition {
	cc := SNotCondition{condition: cond}
	return &cc
}

type SSingleCondition struct {
	field IQueryField
}

func (c *SSingleCondition) Variables() []interface{} {
	return []interface{}{}
}

func NewSingleCondition(field IQueryField) SSingleCondition {
	return SSingleCondition{field: field}
}

type SIsNullCondition struct {
	SSingleCondition
}

func (c *SIsNullCondition) WhereClause() string {
	return fmt.Sprintf("%s IS NULL", c.field.Reference())
}

func IsNull(f IQueryField) ICondition {
	c := SIsNullCondition{NewSingleCondition(f)}
	return &c
}

type SIsNotNullCondition struct {
	SSingleCondition
}

func (c *SIsNotNullCondition) WhereClause() string {
	return fmt.Sprintf("%s IS NOT NULL", c.field.Reference())
}

func IsNotNull(f IQueryField) ICondition {
	c := SIsNotNullCondition{NewSingleCondition(f)}
	return &c
}

type SIsEmptyCondition struct {
	SSingleCondition
}

func (c *SIsEmptyCondition) WhereClause() string {
	return fmt.Sprintf("LENGTH(%s) = 0", c.field.Reference())
}

func IsEmpty(f IQueryField) ICondition {
	c := SIsEmptyCondition{NewSingleCondition(f)}
	return &c
}

type SIsNullOrEmptyCondition struct {
	SSingleCondition
}

func (c *SIsNullOrEmptyCondition) WhereClause() string {
	return fmt.Sprintf("%s IS NULL OR LENGTH(%s) = 0", c.field.Reference(), c.field.Reference())
}

func IsNullOrEmpty(f IQueryField) ICondition {
	c := SIsNullOrEmptyCondition{NewSingleCondition(f)}
	return &c
}

type SIsNotEmptyCondition struct {
	SSingleCondition
}

func (c *SIsNotEmptyCondition) WhereClause() string {
	return fmt.Sprintf("%s IS NOT NULL AND LENGTH(%s) > 0", c.field.Reference(), c.field.Reference())
}

func IsNotEmpty(f IQueryField) ICondition {
	c := SIsNotEmptyCondition{NewSingleCondition(f)}
	return &c
}

type SIsTrueCondition struct {
	SSingleCondition
}

func (c *SIsTrueCondition) WhereClause() string {
	return fmt.Sprintf("%s = 1", c.field.Reference())
}

func IsTrue(f IQueryField) ICondition {
	c := SIsTrueCondition{NewSingleCondition(f)}
	return &c
}

type SIsFalseCondition struct {
	SSingleCondition
}

func (c *SIsFalseCondition) WhereClause() string {
	return fmt.Sprintf("%s = 0", c.field.Reference())
}

func IsFalse(f IQueryField) ICondition {
	c := SIsFalseCondition{NewSingleCondition(f)}
	return &c
}

type SNoLaterThanCondition struct {
	SSingleCondition
}

func (c *SNoLaterThanCondition) WhereClause() string {
	return fmt.Sprintf("%s >= NOW()", c.field.Reference())
}

func NoLaterThan(f IQueryField) ICondition {
	c := SNoLaterThanCondition{NewSingleCondition(f)}
	return &c
}

type SNoEarlierThanCondition struct {
	SSingleCondition
}

func (c *SNoEarlierThanCondition) WhereClause() string {
	return fmt.Sprintf("%s <= NOW()", c.field.Reference())
}

func NoEarlierThan(f IQueryField) ICondition {
	c := SNoEarlierThanCondition{NewSingleCondition(f)}
	return &c
}

type STupleCondition struct {
	left  IQueryField
	right interface{}
}

func tupleConditionWhereClause(t *STupleCondition, op string) string {
	var buf bytes.Buffer
	buf.WriteString(t.left.Reference())
	buf.WriteByte(' ')
	buf.WriteString(op)
	buf.WriteByte(' ')
	buf.WriteString(varConditionWhereClause(t.right))
	return buf.String()
}

func questionMark(count int) string {
	if count == 0 {
		return ""
	} else if count == 1 {
		return "( ? )"
	} else {
		var buf bytes.Buffer
		buf.WriteString("( ")
		for i := 0; i < count; i += 1 {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString("?")
		}
		buf.WriteString(" )")
		return buf.String()
	}
}

func varConditionWhereClause(v interface{}) string {
	switch q := v.(type) {
	case IQueryField:
		return q.Reference()
	case *SQuery:
		return fmt.Sprintf("(%s)", q.String())
	case *SSubQuery:
		return q.Expression()
	default:
		expandV := reflectutils.ExpandInterface(v)
		return questionMark(len(expandV))
	}
}

func varConditionVariables(v interface{}) []interface{} {
	switch v.(type) {
	case IQueryField:
		return []interface{}{}
	case *SQuery:
		q := v.(*SQuery)
		return q.Variables()
	case *SSubQuery:
		q := v.(*SSubQuery)
		return q.query.Variables()
	default:
		return reflectutils.ExpandInterface(v)
	}
}

func NewTupleCondition(l IQueryField, r interface{}) STupleCondition {
	return STupleCondition{left: l, right: r}
}

func (t *STupleCondition) Variables() []interface{} {
	return varConditionVariables(t.right)
}

type SInCondition struct {
	STupleCondition
	op string
}

func inConditionWhereClause(t *STupleCondition, op string) string {
	v := varConditionWhereClause(t.right)
	if len(v) != 0 {
		return tupleConditionWhereClause(t, op)
	}
	return "0"
}

func (t *SInCondition) WhereClause() string {
	return inConditionWhereClause(&t.STupleCondition, t.op)
}

func In(f IQueryField, v interface{}) ICondition {
	c := SInCondition{
		NewTupleCondition(f, v),
		SQL_OP_IN,
	}
	return &c
}

func NotIn(f IQueryField, v interface{}) ICondition {
	c := SInCondition{
		NewTupleCondition(f, v),
		SQL_OP_NOTIN,
	}
	return &c
}

type SLikeCondition struct {
	STupleCondition
}

func likeEscape(s string) string {
	var res bytes.Buffer
	for i := 0; i < len(s); i++ {
		if s[i] == '_' || s[i] == '%' {
			res.WriteByte('\\')
		}
		res.WriteByte(s[i])
	}
	return res.String()
}

func (t *SLikeCondition) WhereClause() string {
	return tupleConditionWhereClause(&t.STupleCondition, SQL_OP_LIKE)
}

func Like(f IQueryField, v string) ICondition {
	c := SLikeCondition{NewTupleCondition(f, v)}
	return &c
}

func ContainsAny(f IQueryField, v []string) ICondition {
	conds := make([]ICondition, len(v))
	for i := range v {
		conds[i] = Contains(f, v[i])
	}
	return OR(conds...)
}

func Contains(f IQueryField, v string) ICondition {
	v = likeEscape(v)
	nv := fmt.Sprintf("%%%s%%", v)
	return Like(f, nv)
}

func Startswith(f IQueryField, v string) ICondition {
	v = likeEscape(v)
	nv := fmt.Sprintf("%s%%", v)
	return Like(f, nv)
}

func Endswith(f IQueryField, v string) ICondition {
	v = likeEscape(v)
	nv := fmt.Sprintf("%%%s", v)
	return Like(f, nv)
}

type SEqualsCondition struct {
	STupleCondition
}

func (t *SEqualsCondition) WhereClause() string {
	return tupleConditionWhereClause(&t.STupleCondition, SQL_OP_EQUAL)
}

func Equals(f IQueryField, v interface{}) ICondition {
	c := SEqualsCondition{NewTupleCondition(f, v)}
	return &c
}

type SNotEqualsCondition struct {
	STupleCondition
}

func (t *SNotEqualsCondition) WhereClause() string {
	return tupleConditionWhereClause(&t.STupleCondition, SQL_OP_NOTEQUAL)
}

func NotEquals(f IQueryField, v interface{}) ICondition {
	c := SNotEqualsCondition{NewTupleCondition(f, v)}
	return &c
}

type SGreatEqualCondition struct {
	STupleCondition
}

func (t *SGreatEqualCondition) WhereClause() string {
	return tupleConditionWhereClause(&t.STupleCondition, SQL_OP_GE)
}

func GE(f IQueryField, v interface{}) ICondition {
	c := SGreatEqualCondition{NewTupleCondition(f, v)}
	return &c
}

type SGreatThanCondition struct {
	STupleCondition
}

func (t *SGreatThanCondition) WhereClause() string {
	return tupleConditionWhereClause(&t.STupleCondition, SQL_OP_GT)
}

func GT(f IQueryField, v interface{}) ICondition {
	c := SGreatThanCondition{NewTupleCondition(f, v)}
	return &c
}

type SLessEqualCondition struct {
	STupleCondition
}

func (t *SLessEqualCondition) WhereClause() string {
	return tupleConditionWhereClause(&t.STupleCondition, SQL_OP_LE)
}

func LE(f IQueryField, v interface{}) ICondition {
	c := SLessEqualCondition{NewTupleCondition(f, v)}
	return &c
}

type SLessThanCondition struct {
	STupleCondition
}

func (t *SLessThanCondition) WhereClause() string {
	return tupleConditionWhereClause(&t.STupleCondition, SQL_OP_LT)
}

func LT(f IQueryField, v interface{}) ICondition {
	c := SLessThanCondition{NewTupleCondition(f, v)}
	return &c
}

type STripleCondition struct {
	STupleCondition
	right2 interface{}
}

func (t *STripleCondition) Variables() []interface{} {
	ret := make([]interface{}, 0)
	vars := varConditionVariables(t.right)
	ret = append(ret, vars...)
	vars = varConditionVariables(t.right2)
	ret = append(ret, vars...)
	return ret
}

func NewTripleCondition(l IQueryField, r interface{}, r2 interface{}) STripleCondition {
	return STripleCondition{STupleCondition: NewTupleCondition(l, r),
		right2: r2}
}

type SBetweenCondition struct {
	STripleCondition
}

func (t *SBetweenCondition) WhereClause() string {
	ret := tupleConditionWhereClause(&t.STupleCondition, SQL_OP_BETWEEN)
	return fmt.Sprintf("%s AND %s", ret, varConditionWhereClause(t.right2))
}

func Between(f IQueryField, r1, r2 interface{}) ICondition {
	c := SBetweenCondition{NewTripleCondition(f, r1, r2)}
	return &c
}

type STrueCondition struct{}

func (t *STrueCondition) WhereClause() string {
	return "1"
}

func (t *STrueCondition) Variables() []interface{} {
	return nil
}

type SFalseCondition struct{}

func (t *SFalseCondition) WhereClause() string {
	return "0"
}

func (t *SFalseCondition) Variables() []interface{} {
	return nil
}
