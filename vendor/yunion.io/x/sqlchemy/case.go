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
)

type sCaseFieldBranch struct {
	whenCondition ICondition
	thenField     IQueryField
}

// SCaseFunction represents function of case ... when ... branch
type SCaseFunction struct {
	branches  []sCaseFieldBranch
	elseField IQueryField
}

// NewFunction creates a field with SQL function
// for example: SUM(count) as total
func NewFunction(ifunc IFunction, name string) IQueryField {
	return &SFunctionFieldBase{
		IFunction: ifunc,
		alias:     name,
	}
}

// Else adds else clause for case when function
func (cf *SCaseFunction) Else(field IQueryField) *SCaseFunction {
	cf.elseField = field
	return cf
}

// When adds when clause for case when function
func (cf *SCaseFunction) When(when ICondition, then IQueryField) *SCaseFunction {
	cf.branches = append(cf.branches, sCaseFieldBranch{
		whenCondition: when,
		thenField:     then,
	})
	return cf
}

// NewCase creates a case... when...else... representation instance
func NewCase() *SCaseFunction {
	return &SCaseFunction{}
}

func (cf *SCaseFunction) expression() string {
	var buf bytes.Buffer
	buf.WriteString("CASE ")
	for i := range cf.branches {
		buf.WriteString("WHEN ")
		buf.WriteString(cf.branches[i].whenCondition.WhereClause())
		buf.WriteString(" THEN ")
		buf.WriteString(cf.branches[i].thenField.Reference())
	}
	buf.WriteString(" ELSE ")
	buf.WriteString(cf.elseField.Reference())
	buf.WriteString(" END")
	return buf.String()
}

func (cf *SCaseFunction) variables() []interface{} {
	vars := make([]interface{}, 0)
	for i := range cf.branches {
		fromvars := cf.branches[i].whenCondition.Variables()
		vars = append(vars, fromvars...)
		fromvars = cf.branches[i].thenField.Variables()
		vars = append(vars, fromvars...)
	}
	fromvars := cf.elseField.Variables()
	vars = append(vars, fromvars...)
	return vars
}
