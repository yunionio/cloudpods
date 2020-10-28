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

func (tq *SQuery) FilterByTrue() *SQuery {
	return tq.Filter(&STrueCondition{})
}

func (tq *SQuery) FilterByFalse() *SQuery {
	return tq.Filter(&SFalseCondition{})
}

func (q *SQuery) Like(f string, v string) *SQuery {
	cond := Like(q.Field(f), v)
	return q.Filter(cond)
}

func (q *SQuery) Contains(f string, v string) *SQuery {
	cond := Contains(q.Field(f), v)
	return q.Filter(cond)
}

func (q *SQuery) Startswith(f string, v string) *SQuery {
	cond := Startswith(q.Field(f), v)
	return q.Filter(cond)
}

func (q *SQuery) Endswith(f string, v string) *SQuery {
	cond := Endswith(q.Field(f), v)
	return q.Filter(cond)
}

func (q *SQuery) NotLike(f string, v string) *SQuery {
	cond := Like(q.Field(f), v)
	return q.Filter(NOT(cond))
}

func (q *SQuery) In(f string, v interface{}) *SQuery {
	cond := In(q.Field(f), v)
	return q.Filter(cond)
}

func (q *SQuery) NotIn(f string, v interface{}) *SQuery {
	cond := In(q.Field(f), v)
	return q.Filter(NOT(cond))
}

func (q *SQuery) Between(f string, v1, v2 interface{}) *SQuery {
	cond := Between(q.Field(f), v1, v2)
	return q.Filter(cond)
}

func (q *SQuery) NotBetween(f string, v1, v2 interface{}) *SQuery {
	cond := Between(q.Field(f), v1, v2)
	return q.Filter(NOT(cond))
}

func (q *SQuery) Equals(f string, v interface{}) *SQuery {
	cond := Equals(q.Field(f), v)
	return q.Filter(cond)
}

func (q *SQuery) NotEquals(f string, v interface{}) *SQuery {
	cond := NotEquals(q.Field(f), v)
	return q.Filter(cond)
}

func (q *SQuery) GE(f string, v interface{}) *SQuery {
	cond := GE(q.Field(f), v)
	return q.Filter(cond)
}

func (q *SQuery) LE(f string, v interface{}) *SQuery {
	cond := LE(q.Field(f), v)
	return q.Filter(cond)
}

func (q *SQuery) GT(f string, v interface{}) *SQuery {
	cond := GT(q.Field(f), v)
	return q.Filter(cond)
}

func (q *SQuery) LT(f string, v interface{}) *SQuery {
	cond := LT(q.Field(f), v)
	return q.Filter(cond)
}

func (q *SQuery) IsNull(f string) *SQuery {
	cond := IsNull(q.Field(f))
	return q.Filter(cond)
}

func (q *SQuery) IsNotNull(f string) *SQuery {
	cond := IsNotNull(q.Field(f))
	return q.Filter(cond)
}

func (q *SQuery) IsEmpty(f string) *SQuery {
	cond := IsEmpty(q.Field(f))
	return q.Filter(cond)
}

func (q *SQuery) IsNullOrEmpty(f string) *SQuery {
	cond := IsNullOrEmpty(q.Field(f))
	return q.Filter(cond)
}

func (q *SQuery) IsNotEmpty(f string) *SQuery {
	cond := IsNotEmpty(q.Field(f))
	return q.Filter(cond)
}

func (q *SQuery) IsTrue(f string) *SQuery {
	cond := IsTrue(q.Field(f))
	return q.Filter(cond)
}

func (q *SQuery) IsFalse(f string) *SQuery {
	cond := IsFalse(q.Field(f))
	return q.Filter(cond)
}
