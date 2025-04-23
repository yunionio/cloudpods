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
	"sort"
)

// IQuery is an interface that reprsents a SQL query, e.g.
// SELECT ... FROM ... WHERE ...
type IQuery interface {
	// String returns the queryString
	String(fields ...IQueryField) string

	// QueryFields returns fields in the select clause
	QueryFields() []IQueryField

	// Variables returns variables in statement
	Variables() []interface{}

	// SubQuery convert this SQL to a subquery
	SubQuery() *SSubQuery

	// Field reference to a field by name
	Field(name string) IQueryField

	// Database returns the database for this query
	database() *SDatabase
}

// IQuerySource is an interface that represents a data source of a SQL query. the source can be a table or a subquery
// e.g. SELECT ... FROM (SELECT * FROM tbl) AS A
type IQuerySource interface {
	// Expression string in select ... from (expresson here)
	Expression() string

	// Alias is the alias in select ... from (express) as alias
	Alias() string

	// variables in statement
	Variables() []interface{}

	// Field reference to a field by name, optionally giving an alias name
	Field(id string, alias ...string) IQueryField

	// Fields return all the fields that this source provides
	Fields() []IQueryField

	// Database returns the database of this IQuerySource
	database() *SDatabase
}

// IQueryField is an interface that represents a select field in a SQL query
type IQueryField interface {
	// the string after select
	Expression() string

	// the name of thie field
	Name() string

	// the reference string in where clause
	Reference() string

	// give this field an alias name
	Label(label string) IQueryField

	// return variables
	Variables() []interface{}

	// ConvertFromValue returns the SQL representation of a value for this
	ConvertFromValue(val interface{}) interface{}

	// Database returns the database of this IQuerySource
	database() *SDatabase
}

// DoQuery returns a SQuery instance that query specified fields from a query source
func DoQuery(from IQuerySource, f ...IQueryField) *SQuery {
	if from.database() == nil {
		panic("DoQuery IQuerySource with empty database")
	}
	// if len(f) == 0 {
	// 	f = from.Fields()
	// }
	tq := SQuery{fields: f, from: from, db: from.database()}
	return &tq
}

func queryString(tq *SQuery, tmpFields ...IQueryField) string {
	if len(tq.rawSql) > 0 {
		return tq.rawSql
	}

	qChar := tq.database().backend.QuoteChar()

	var buf bytes.Buffer
	buf.WriteString("SELECT ")
	if tq.distinct {
		buf.WriteString("DISTINCT ")
	}

	fields := tmpFields
	if len(fields) == 0 {
		if len(tq.fields) > 0 {
			fields = append(fields, tq.fields...)
		} else {
			fields = append(fields, tq.from.Fields()...)
			for i := range fields {
				tq.from.Field(fields[i].Name())
			}
		}
	}
	{
		// add reference query fields
		queryFields := make(map[string]IQueryField)
		for i, f := range fields {
			queryFields[f.Name()] = fields[i]
		}
		refFields := make([]IQueryField, 0)
		for _, f := range tq.refFieldMap {
			if _, ok := queryFields[f.Name()]; !ok {
				queryFields[f.Name()] = f
				refFields = append(refFields, f)
			}
		}
		sort.Sort(queryFieldList(refFields))
		fields = append(fields, refFields...)
	}

	groupFields := make(map[string]IQueryField)
	if len(tq.groupBy) > 0 {
		for i := range tq.groupBy {
			f := tq.groupBy[i]
			groupFields[f.Name()] = f
		}
	}

	for i := range fields {
		if i > 0 {
			buf.WriteString(", ")
		}
		f := fields[i]
		if len(tq.groupBy) > 0 {
			if gf, ok := groupFields[f.Name()]; ok && f.Reference() == gf.Reference() {
				// in group by, normal
				buf.WriteString(f.Expression())
			} else {
				// not in group by, check if the field is a group aggregate function
				if gf, ok := f.(IFunctionQueryField); ok && gf.IsAggregate() {
					// is a aggregate function field
					buf.WriteString(f.Expression())
				} else {
					f = MAX(f.Name(), f)
					buf.WriteString(f.Expression())
				}
			}
		} else {
			// normal
			buf.WriteString(f.Expression())
		}
		if f.Name() != "" {
			buf.WriteString(fmt.Sprintf(" AS %s%s%s", qChar, f.Name(), qChar))
		}
	}
	buf.WriteString(" FROM ")
	buf.WriteString(fmt.Sprintf("%s AS %s%s%s", tq.from.Expression(), qChar, tq.from.Alias(), qChar))
	for _, join := range tq.joins {
		buf.WriteByte(' ')
		buf.WriteString(string(join.jointype))
		buf.WriteByte(' ')
		buf.WriteString(fmt.Sprintf("%s AS %s%s%s", join.from.Expression(), qChar, join.from.Alias(), qChar))
		whereCls := join.condition.WhereClause()
		if len(whereCls) > 0 {
			buf.WriteString(" ON ")
			buf.WriteString(whereCls)
		}
	}
	if tq.where != nil {
		whereCls := tq.where.WhereClause()
		if len(whereCls) > 0 {
			buf.WriteString(" WHERE ")
			buf.WriteString(whereCls)
		}
	}
	if tq.groupBy != nil && len(tq.groupBy) > 0 {
		buf.WriteString(" GROUP BY ")
		groupByFields := make(map[string]IQueryField)
		for i := range tq.groupBy {
			f := tq.groupBy[i]
			if _, ok := groupByFields[f.Reference()]; ok {
				continue
			}
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(f.Reference())
			groupByFields[f.Reference()] = f
		}
		// DAMENG SQL Compatibility, all order by fields should be in group by
		for i := range tq.orderBy {
			f := tq.orderBy[i]
			if _, ok := groupByFields[f.field.Reference()]; ok {
				continue
			}
			if ff, ok := f.field.(IFunctionQueryField); ok && ff.IsAggregate() {
				continue
			}
			buf.WriteString(", ")
			buf.WriteString(f.field.Reference())
			groupByFields[f.field.Reference()] = f.field
		}
	}
	/*if tq.having != nil {
		buf.WriteString(" HAVING ")
		buf.WriteString(tq.having.WhereClause())
	}*/
	if tq.orderBy != nil && len(tq.orderBy) > 0 {
		buf.WriteString(" ORDER BY ")
		for i := range tq.orderBy {
			f := tq.orderBy[i]
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(fmt.Sprintf("%s %s", f.field.Reference(), f.order))
		}
	}
	if tq.limit > 0 {
		buf.WriteString(fmt.Sprintf(" LIMIT %d", tq.limit))
	}
	if tq.offset > 0 {
		buf.WriteString(fmt.Sprintf(" OFFSET %d", tq.offset))
	}
	return buf.String()
}

func getFieldBackend(fields ...IQueryField) IBackend {
	for _, f := range fields {
		db := f.database()
		if db != nil {
			return db.backend
		}
	}
	return defaultBackend
}
