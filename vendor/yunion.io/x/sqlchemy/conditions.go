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

// ICondition is the interface representing a condition for SQL query
// e.g. WHERE a1 = b1 is a condition of equal
// the condition support nested condition, with AND, OR and NOT boolean operators
type ICondition interface {
	WhereClause() string
	Variables() []interface{}
}

// SCompoundConditions is a Compound condition represents AND or OR boolean operation
// Compound condition also follows the ICondition interface
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

// WhereClause implementation of SCompoundConditions for ICondition
func (c *SCompoundConditions) WhereClause() string {
	return ""
}

// Variables implementation of SCompoundConditions for ICondition
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

// SAndConditions represents the AND condition, which is a SCompoundConditions
type SAndConditions struct {
	SCompoundConditions
}

// WhereClause implementation of SAndConditions for IConditionq
func (c *SAndConditions) WhereClause() string {
	return compoundWhereClause(&c.SCompoundConditions, SQL_OP_AND)
}

// SOrConditions represents the OR condition, which is a SCompoundConditions
type SOrConditions struct {
	SCompoundConditions
}

// WhereClause implementation of SOrConditions for ICondition
func (c *SOrConditions) WhereClause() string {
	return compoundWhereClause(&c.SCompoundConditions, SQL_OP_OR)
}

// AND method that combines many conditions with AND operator
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

// OR method that combines many conditions with OR operator
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

// SNotCondition represents the NOT condition, which is a boolean operator
type SNotCondition struct {
	condition ICondition
}

// WhereClause implementationq of SNotCondition for ICondition
func (c *SNotCondition) WhereClause() string {
	return fmt.Sprintf("%s (%s)", SQL_OP_NOT, c.condition.WhereClause())
}

// Variables implementation of SNotCondition for ICondition
func (c *SNotCondition) Variables() []interface{} {
	return c.condition.Variables()
}

// NOT method that makes negative operator on a condition
func NOT(cond ICondition) ICondition {
	cc := SNotCondition{condition: cond}
	return &cc
}

// SSingleCondition represents a kind of condition that composed of one query field
type SSingleCondition struct {
	field IQueryField
}

// Variables implementation of SSingleCondition for ICondition
func (c *SSingleCondition) Variables() []interface{} {
	return []interface{}{}
}

// NewSingleCondition returns an instance of SSingleCondition
func NewSingleCondition(field IQueryField) SSingleCondition {
	return SSingleCondition{field: field}
}

// SIsNullCondition is a condition representing a comparison with null, e.g. a is null
type SIsNullCondition struct {
	SSingleCondition
}

// WhereClause implementation for SIsNullCondition for ICondition
func (c *SIsNullCondition) WhereClause() string {
	return fmt.Sprintf("%s IS NULL", c.field.Reference())
}

// IsNull methods that justifies a field is null
func IsNull(f IQueryField) ICondition {
	c := SIsNullCondition{NewSingleCondition(f)}
	return &c
}

// SIsNotNullCondition is a condition represents a comparison with not null, e.g. a is not null
type SIsNotNullCondition struct {
	SSingleCondition
}

// WhereClause implementation of SIsNotNullCondition for ICondition
func (c *SIsNotNullCondition) WhereClause() string {
	return fmt.Sprintf("%s IS NOT NULL", c.field.Reference())
}

// IsNotNull methods that justifies a field is not null
func IsNotNull(f IQueryField) ICondition {
	c := SIsNotNullCondition{NewSingleCondition(f)}
	return &c
}

// SIsEmptyCondition is a condition representing the empty status of a field
type SIsEmptyCondition struct {
	SSingleCondition
}

// WhereClause implementation of SIsEmptyCondition for ICondition
func (c *SIsEmptyCondition) WhereClause() string {
	return fmt.Sprintf("LENGTH(%s) = 0", c.field.Reference())
}

// IsEmpty method that justifies where a text field is empty, e.g. length is zero
func IsEmpty(f IQueryField) ICondition {
	c := SIsEmptyCondition{NewSingleCondition(f)}
	return &c
}

// SIsNullOrEmptyCondition is a condition that justifies a field is null or empty
type SIsNullOrEmptyCondition struct {
	SSingleCondition
}

// WhereClause implementation of SIsNullOrEmptyCondition for ICondition
func (c *SIsNullOrEmptyCondition) WhereClause() string {
	return fmt.Sprintf("%s IS NULL OR LENGTH(%s) = 0", c.field.Reference(), c.field.Reference())
}

// IsNullOrEmpty is the ethod justifies a field is null or empty, e.g. a is null or length(a) == 0
func IsNullOrEmpty(f IQueryField) ICondition {
	c := SIsNullOrEmptyCondition{NewSingleCondition(f)}
	return &c
}

// SIsNotEmptyCondition represents a condition that represents a field is not empty
type SIsNotEmptyCondition struct {
	SSingleCondition
}

// WhereClause implementation of SIsNotEmptyCondition for ICondition
func (c *SIsNotEmptyCondition) WhereClause() string {
	return fmt.Sprintf("%s IS NOT NULL AND LENGTH(%s) > 0", c.field.Reference(), c.field.Reference())
}

// IsNotEmpty method justifies a field is not empty
func IsNotEmpty(f IQueryField) ICondition {
	c := SIsNotEmptyCondition{NewSingleCondition(f)}
	return &c
}

// SIsTrueCondition represents a boolean field (TINYINT) is true, e.g. a == 1
type SIsTrueCondition struct {
	SSingleCondition
}

// WhereClause implementation of SIsTrueCondition for ICondition
func (c *SIsTrueCondition) WhereClause() string {
	return fmt.Sprintf("%s = 1", c.field.Reference())
}

// IsTrue method that justifies a field is true, e.g. field == 1
func IsTrue(f IQueryField) ICondition {
	c := SIsTrueCondition{NewSingleCondition(f)}
	return &c
}

// SIsFalseCondition represents a boolean is false
type SIsFalseCondition struct {
	SSingleCondition
}

// WhereClause implementation of SIsFalseCondition for ICondition
func (c *SIsFalseCondition) WhereClause() string {
	return fmt.Sprintf("%s = 0", c.field.Reference())
}

// IsFalse method justifies a boolean is false
func IsFalse(f IQueryField) ICondition {
	c := SIsFalseCondition{NewSingleCondition(f)}
	return &c
}

// SNoLaterThanCondition coompares a DATETIME field with current time and ensure the field is no later than now, e.g. a <= NOW()
type SNoLaterThanCondition struct {
	SSingleCondition
}

// WhereClause implementation of SNoLaterThanCondition for ICondition
func (c *SNoLaterThanCondition) WhereClause() string {
	return fmt.Sprintf("%s <= NOW()", c.field.Reference())
}

// NoLaterThan method justifies a DATETIME field is before current time
func NoLaterThan(f IQueryField) ICondition {
	c := SNoLaterThanCondition{NewSingleCondition(f)}
	return &c
}

// SNoEarlierThanCondition compares a field with current time and ensure the field is no earlier than NOW, e.g. a >= NOW()
type SNoEarlierThanCondition struct {
	SSingleCondition
}

// WhereClause implementation of SNoEarlierThanCondition for ICondition
func (c *SNoEarlierThanCondition) WhereClause() string {
	return fmt.Sprintf("%s >= NOW()", c.field.Reference())
}

// NoEarlierThan justifies a field is no earlier than current time
func NoEarlierThan(f IQueryField) ICondition {
	c := SNoEarlierThanCondition{NewSingleCondition(f)}
	return &c
}

// STupleCondition is a base condition that composed of two fields
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
		for i := 0; i < count; i++ {
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

// NewTupleCondition returns an instance of tuple condition
func NewTupleCondition(l IQueryField, r interface{}) STupleCondition {
	return STupleCondition{left: l, right: r}
}

// Variables implementation of STupleCondition for ICondition
func (t *STupleCondition) Variables() []interface{} {
	return varConditionVariables(t.right)
}

// SInCondition represents a IN operation in SQL query
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

// WhereClause implementation of SInCondition for ICondition
func (t *SInCondition) WhereClause() string {
	return inConditionWhereClause(&t.STupleCondition, t.op)
}

// In SQL operator
func In(f IQueryField, v interface{}) ICondition {
	c := SInCondition{
		NewTupleCondition(f, v),
		SQL_OP_IN,
	}
	return &c
}

// NotIn SQL operator
func NotIn(f IQueryField, v interface{}) ICondition {
	c := SInCondition{
		NewTupleCondition(f, v),
		SQL_OP_NOTIN,
	}
	return &c
}

// SLikeCondition represents LIKE operation in a SQL query
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

// WhereClause implementation for SLikeCondition for ICondition
func (t *SLikeCondition) WhereClause() string {
	return tupleConditionWhereClause(&t.STupleCondition, SQL_OP_LIKE)
}

// Like SQL operator
func Like(f IQueryField, v string) ICondition {
	c := SLikeCondition{NewTupleCondition(f, v)}
	return &c
}

// ContainsAny is a OR combination of serveral Contains conditions
func ContainsAny(f IQueryField, v []string) ICondition {
	conds := make([]ICondition, len(v))
	for i := range v {
		conds[i] = Contains(f, v[i])
	}
	return OR(conds...)
}

// Contains method is a shortcut of LIKE method, Contains represents the condtion that a field contains a substring
func Contains(f IQueryField, v string) ICondition {
	v = likeEscape(v)
	nv := fmt.Sprintf("%%%s%%", v)
	return Like(f, nv)
}

// Startswith method is a shortcut of LIKE method, Startswith represents the condition that field starts with a substring
func Startswith(f IQueryField, v string) ICondition {
	v = likeEscape(v)
	nv := fmt.Sprintf("%s%%", v)
	return Like(f, nv)
}

// Endswith method is a shortcut of LIKE condition, Endswith represents that condition that field endswith a substring
func Endswith(f IQueryField, v string) ICondition {
	v = likeEscape(v)
	nv := fmt.Sprintf("%%%s", v)
	return Like(f, nv)
}

// SEqualsCondition represents equal operation between two fields
type SEqualsCondition struct {
	STupleCondition
}

// WhereClause implementation of SEqualsCondition for ICondition
func (t *SEqualsCondition) WhereClause() string {
	return tupleConditionWhereClause(&t.STupleCondition, SQL_OP_EQUAL)
}

// Equals method represents equal of two fields
func Equals(f IQueryField, v interface{}) ICondition {
	c := SEqualsCondition{NewTupleCondition(f, v)}
	return &c
}

// SNotEqualsCondition is the opposite of equal condition
type SNotEqualsCondition struct {
	STupleCondition
}

// WhereClause implementation of SNotEqualsCondition for ICondition
func (t *SNotEqualsCondition) WhereClause() string {
	return tupleConditionWhereClause(&t.STupleCondition, SQL_OP_NOTEQUAL)
}

// NotEquals method represents not equal of two fields
func NotEquals(f IQueryField, v interface{}) ICondition {
	c := SNotEqualsCondition{NewTupleCondition(f, v)}
	return &c
}

// SGreatEqualCondition represents >= operation on two fields
type SGreatEqualCondition struct {
	STupleCondition
}

// WhereClause implementation of SGreatEqualCondition for ICondition
func (t *SGreatEqualCondition) WhereClause() string {
	return tupleConditionWhereClause(&t.STupleCondition, SQL_OP_GE)
}

// GE method represetns operation of Greate Than Or Equal to, e.g. a >= b
func GE(f IQueryField, v interface{}) ICondition {
	c := SGreatEqualCondition{NewTupleCondition(f, v)}
	return &c
}

// SGreatThanCondition represetns > operation on two fields
type SGreatThanCondition struct {
	STupleCondition
}

// WhereClause implementation of SGreatThanCondition for ICondition
func (t *SGreatThanCondition) WhereClause() string {
	return tupleConditionWhereClause(&t.STupleCondition, SQL_OP_GT)
}

// GT method represents operation of Great Than, e.g. a > b
func GT(f IQueryField, v interface{}) ICondition {
	c := SGreatThanCondition{NewTupleCondition(f, v)}
	return &c
}

// SLessEqualCondition represents <= operation on two fields
type SLessEqualCondition struct {
	STupleCondition
}

// WhereClause implementation of SLessEqualCondition for ICondition
func (t *SLessEqualCondition) WhereClause() string {
	return tupleConditionWhereClause(&t.STupleCondition, SQL_OP_LE)
}

// LE method represents operation of Less Than Or Equal to, e.q. a <= b
func LE(f IQueryField, v interface{}) ICondition {
	c := SLessEqualCondition{NewTupleCondition(f, v)}
	return &c
}

// SLessThanCondition represents < operation on two fields
type SLessThanCondition struct {
	STupleCondition
}

// WhereClause implementation of SLessThanCondition for ICondition
func (t *SLessThanCondition) WhereClause() string {
	return tupleConditionWhereClause(&t.STupleCondition, SQL_OP_LT)
}

// LT method represents operation of Less Than, e.g. a < b
func LT(f IQueryField, v interface{}) ICondition {
	c := SLessThanCondition{NewTupleCondition(f, v)}
	return &c
}

// STripleCondition represents a base condition that composed of THREE fields
type STripleCondition struct {
	STupleCondition
	right2 interface{}
}

// Variables implementation of STripleCondition for ICondition
func (t *STripleCondition) Variables() []interface{} {
	ret := make([]interface{}, 0)
	vars := varConditionVariables(t.right)
	ret = append(ret, vars...)
	vars = varConditionVariables(t.right2)
	ret = append(ret, vars...)
	return ret
}

// NewTripleCondition return an instance of STripleCondition
func NewTripleCondition(l IQueryField, r interface{}, r2 interface{}) STripleCondition {
	return STripleCondition{STupleCondition: NewTupleCondition(l, r),
		right2: r2}
}

// SBetweenCondition represents BETWEEN operator, e.g. c between a and b
type SBetweenCondition struct {
	STripleCondition
}

// WhereClause implementation of SBetweenCondition for ICondition
func (t *SBetweenCondition) WhereClause() string {
	ret := tupleConditionWhereClause(&t.STupleCondition, SQL_OP_BETWEEN)
	return fmt.Sprintf("%s AND %s", ret, varConditionWhereClause(t.right2))
}

// Between SQL operator
func Between(f IQueryField, r1, r2 interface{}) ICondition {
	c := SBetweenCondition{NewTripleCondition(f, r1, r2)}
	return &c
}

// STrueCondition represents a dummy condition that is always true
type STrueCondition struct{}

// WhereClause implementation of STrueCondition for ICondition
func (t *STrueCondition) WhereClause() string {
	return "1"
}

// Variables implementation of STrueCondition for ICondition
func (t *STrueCondition) Variables() []interface{} {
	return nil
}

// SFalseCondition is a dummy condition that is always false
type SFalseCondition struct{}

// WhereClause implementation of SFalseCondition for ICondition
func (t *SFalseCondition) WhereClause() string {
	return "0"
}

// Variables implementation of SFalseCondition for ICondition
func (t *SFalseCondition) Variables() []interface{} {
	return nil
}
