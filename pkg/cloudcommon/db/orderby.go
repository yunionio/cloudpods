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

package db

import "yunion.io/x/sqlchemy"

func NeedOrderQuery(fieldOrders []string) bool {
	for _, field := range fieldOrders {
		if sqlchemy.SQL_ORDER_ASC.Equals(field) || sqlchemy.SQL_ORDER_DESC.Equals(field) {
			return true
		}
	}
	return false
}

func OrderByFields(q *sqlchemy.SQuery, fieldOrders []string, fields []sqlchemy.IQueryField) *sqlchemy.SQuery {
	for i := range fields {
		if sqlchemy.SQL_ORDER_ASC.Equals(fieldOrders[i]) {
			q = q.Asc(fields[i])
		} else if sqlchemy.SQL_ORDER_DESC.Equals(fieldOrders[i]) {
			q = q.Desc(fields[i])
		}
	}
	return q
}

func OrderByStandaloneResourceName(q *sqlchemy.SQuery, modelManager IStandaloneModelManager, fieldName string, orderBy string) *sqlchemy.SQuery {
	subq := modelManager.Query("id", "name").SubQuery()
	orders := []string{orderBy}
	if NeedOrderQuery(orders) {
		q = q.LeftJoin(subq, sqlchemy.Equals(q.Field(fieldName), subq.Field("id")))
		q = OrderByFields(q, orders, []sqlchemy.IQueryField{subq.Field("name")})
	}
	return q
}
