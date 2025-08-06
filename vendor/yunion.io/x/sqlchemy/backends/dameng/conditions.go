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

package dameng

import (
	"yunion.io/x/sqlchemy"
)

// SEqualsCondition represents equal operation between two fields
type SDamengEqualsCondition struct {
	sqlchemy.STupleCondition
}

// WhereClause implementation of SDamengEqualsCondition for ICondition
// 修复达梦数据库报错：数据类型不匹配
func (t *SDamengEqualsCondition) WhereClause() string {
	return sqlchemy.TupleConditionWhereClauseWithFuncname(&t.STupleCondition, "TEXT_EQUAL")
}

// Equals filter conditions
func (dameng *SDamengBackend) Equals(f sqlchemy.IQueryField, v interface{}) sqlchemy.ICondition {
	// log.Debugf("field %s isFieldText: %v %#v", f.Name(), sqlchemy.IsFieldText(f), f)
	if sqlchemy.IsFieldText(f) {
		c := SDamengEqualsCondition{sqlchemy.NewTupleCondition(f, v)}
		return &c
	} else {
		return dameng.SBaseBackend.Equals(f, v)
	}
}
