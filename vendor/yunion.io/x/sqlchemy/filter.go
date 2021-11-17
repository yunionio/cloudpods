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

// Filter method filters a SQL query with given ICondition
// equivalent to add a clause in where conditions
func (tq *SQuery) Filter(cond ICondition) *SQuery {
	if tq.groupBy != nil && len(tq.groupBy) > 0 {
		if tq.having == nil {
			tq.having = cond
		} else {
			tq.having = AND(tq.having, cond)
		}
	} else {
		if tq.where == nil {
			tq.where = cond
		} else {
			tq.where = AND(tq.where, cond)
		}
	}
	return tq
}

// FilterByTrue filters query with a true condition
func (tq *SQuery) FilterByTrue() *SQuery {
	return tq.Filter(&STrueCondition{})
}

// FilterByFalse filters query with a false condition
func (tq *SQuery) FilterByFalse() *SQuery {
	return tq.Filter(&SFalseCondition{})
}

// Like filters query with a like condition
func (tq *SQuery) Like(f string, v string) *SQuery {
	cond := Like(tq.Field(f), v)
	return tq.Filter(cond)
}

// Contains filters query with a contains condition
func (tq *SQuery) Contains(f string, v string) *SQuery {
	cond := Contains(tq.Field(f), v)
	return tq.Filter(cond)
}

// Startswith filters query with a startswith condition
func (tq *SQuery) Startswith(f string, v string) *SQuery {
	cond := Startswith(tq.Field(f), v)
	return tq.Filter(cond)
}

// Endswith filters query with a endswith condition
func (tq *SQuery) Endswith(f string, v string) *SQuery {
	cond := Endswith(tq.Field(f), v)
	return tq.Filter(cond)
}

// NotLike filters query with a not like condition
func (tq *SQuery) NotLike(f string, v string) *SQuery {
	cond := Like(tq.Field(f), v)
	return tq.Filter(NOT(cond))
}

// In filters query with a in condition
func (tq *SQuery) In(f string, v interface{}) *SQuery {
	cond := In(tq.Field(f), v)
	return tq.Filter(cond)
}

// NotIn filters query with a not in condition
func (tq *SQuery) NotIn(f string, v interface{}) *SQuery {
	cond := In(tq.Field(f), v)
	return tq.Filter(NOT(cond))
}

// Between filters query with a between condition
func (tq *SQuery) Between(f string, v1, v2 interface{}) *SQuery {
	cond := Between(tq.Field(f), v1, v2)
	return tq.Filter(cond)
}

// NotBetween filters query with a not between condition
func (tq *SQuery) NotBetween(f string, v1, v2 interface{}) *SQuery {
	cond := Between(tq.Field(f), v1, v2)
	return tq.Filter(NOT(cond))
}

// Equals filters query with a equals condition
func (tq *SQuery) Equals(f string, v interface{}) *SQuery {
	cond := Equals(tq.Field(f), v)
	return tq.Filter(cond)
}

// NotEquals filters the query with a not equals condition
func (tq *SQuery) NotEquals(f string, v interface{}) *SQuery {
	cond := NotEquals(tq.Field(f), v)
	return tq.Filter(cond)
}

// GE filters the query with a >= condition
func (tq *SQuery) GE(f string, v interface{}) *SQuery {
	cond := GE(tq.Field(f), v)
	return tq.Filter(cond)
}

// LE filters the query with a <= condition
func (tq *SQuery) LE(f string, v interface{}) *SQuery {
	cond := LE(tq.Field(f), v)
	return tq.Filter(cond)
}

// GT filters the query with a > condition
func (tq *SQuery) GT(f string, v interface{}) *SQuery {
	cond := GT(tq.Field(f), v)
	return tq.Filter(cond)
}

// LT filters the query with a < condition
func (tq *SQuery) LT(f string, v interface{}) *SQuery {
	cond := LT(tq.Field(f), v)
	return tq.Filter(cond)
}

// IsNull filters the query with a is null condition
func (tq *SQuery) IsNull(f string) *SQuery {
	cond := IsNull(tq.Field(f))
	return tq.Filter(cond)
}

// IsNotNull filters the query with a is not null condition
func (tq *SQuery) IsNotNull(f string) *SQuery {
	cond := IsNotNull(tq.Field(f))
	return tq.Filter(cond)
}

// IsEmpty filters the query with a is_empty condition
func (tq *SQuery) IsEmpty(f string) *SQuery {
	cond := IsEmpty(tq.Field(f))
	return tq.Filter(cond)
}

// IsNullOrEmpty filters the query with a is null or empty condition
func (tq *SQuery) IsNullOrEmpty(f string) *SQuery {
	cond := IsNullOrEmpty(tq.Field(f))
	return tq.Filter(cond)
}

// IsNotEmpty filters the query with a is not empty condition
func (tq *SQuery) IsNotEmpty(f string) *SQuery {
	cond := IsNotEmpty(tq.Field(f))
	return tq.Filter(cond)
}

// IsTrue filters the query with a is true condition
func (tq *SQuery) IsTrue(f string) *SQuery {
	cond := IsTrue(tq.Field(f))
	return tq.Filter(cond)
}

// IsFalse filters the query with a is false condition
func (tq *SQuery) IsFalse(f string) *SQuery {
	cond := IsFalse(tq.Field(f))
	return tq.Filter(cond)
}
