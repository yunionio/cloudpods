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
	"database/sql"
	"fmt"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type SUnionQueryField struct {
	union *SUnion
	name  string
	alias string
}

func (sqf *SUnionQueryField) Expression() string {
	if len(sqf.alias) > 0 {
		return fmt.Sprintf("`%s`.`%s` as `%s`", sqf.union.Alias(), sqf.name, sqf.alias)
	} else {
		return fmt.Sprintf("`%s`.`%s`", sqf.union.Alias(), sqf.name)
	}
}

func (sqf *SUnionQueryField) Name() string {
	if len(sqf.alias) > 0 {
		return sqf.alias
	} else {
		return sqf.name
	}
}

func (sqf *SUnionQueryField) Reference() string {
	return fmt.Sprintf("`%s`.`%s`", sqf.union.Alias(), sqf.Name())
}

func (sqf *SUnionQueryField) Label(label string) IQueryField {
	if len(label) > 0 && label != sqf.name {
		sqf.alias = label
	}
	return sqf
}

func (sqf *SUnionQueryField) Variables() []interface{} {
	return nil
}

type SUnion struct {
	alias   string
	queries []IQuery
	fields  []IQueryField
	orderBy []SQueryOrder
	limit   int
	offset  int
}

func (uq *SUnion) Alias() string {
	return uq.alias
}

func (uq *SUnion) Expression() string {
	var buf strings.Builder
	buf.WriteByte('(')
	for i := range uq.queries {
		if i != 0 {
			buf.WriteString(" UNION ")
		}
		buf.WriteByte('(')
		buf.WriteString(uq.queries[i].String())
		buf.WriteByte(')')
	}
	if uq.orderBy != nil && len(uq.orderBy) > 0 {
		buf.WriteString(" ORDER BY ")
		for i, f := range uq.orderBy {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(fmt.Sprintf("%s %s", f.field.Reference(), f.order))
		}
	}
	if uq.limit > 0 {
		buf.WriteString(fmt.Sprintf(" LIMIT %d", uq.limit))
	}
	if uq.offset > 0 {
		buf.WriteString(fmt.Sprintf(" OFFSET %d", uq.offset))
	}
	buf.WriteByte(')')
	return buf.String()
}

/*func (tq *SUnion) _orderBy(order QueryOrderType, fields []IQueryField) *SUnion {
	if tq.orderBy == nil {
		tq.orderBy = make([]SQueryOrder, 0)
	}
	for _, f := range fields {
		tq.orderBy = append(tq.orderBy, SQueryOrder{field: f, order: order})
	}
	return tq
}

func (tq *SUnion) Asc(fields ...interface{}) *SUnion {
	return tq._orderBy(SQL_ORDER_ASC, convertQueryField(tq, fields))
}

func (tq *SUnion) Desc(fields ...interface{}) *SUnion {
	return tq._orderBy(SQL_ORDER_DESC, convertQueryField(tq, fields))
}
*/

func (uq *SUnion) Limit(limit int) *SUnion {
	uq.limit = limit
	return uq
}

func (uq *SUnion) Offset(offset int) *SUnion {
	uq.offset = offset
	return uq
}

func (uq *SUnion) Fields() []IQueryField {
	return uq.fields
}

func (uq *SUnion) Field(name string, alias ...string) IQueryField {
	for i := range uq.fields {
		if name == uq.fields[i].Name() {
			if len(alias) > 0 {
				uq.fields[i].Label(alias[0])
			}
			return uq.fields[i]
		}
	}
	return nil
}

func (uq *SUnion) Variables() []interface{} {
	ret := make([]interface{}, 0)
	for i := range uq.queries {
		ret = append(ret, uq.queries[i].Variables()...)
	}
	return ret
}

// deprecated
func Union(query ...IQuery) *SUnion {
	u, err := UnionWithError(query...)
	if err != nil {
		log.Fatalf("Fatal: %s", err.Error())
	}
	return u
}

func UnionWithError(query ...IQuery) (*SUnion, error) {
	if len(query) == 0 {
		return nil, errors.Wrap(sql.ErrNoRows, "empty union query")
	}

	fieldNames := make([]string, 0)
	for _, f := range query[0].QueryFields() {
		fieldNames = append(fieldNames, f.Name())
	}

	for i := 1; i < len(query); i += 1 {
		qfields := query[i].QueryFields()
		if len(fieldNames) != len(qfields) {
			return nil, fmt.Errorf("cannot union, number of fields not match!")
		}
		for i := range qfields {
			if fieldNames[i] != qfields[i].Name() {
				return nil, fmt.Errorf("cannot union, name of fields not match!")
			}
		}
	}

	fields := make([]IQueryField, len(fieldNames))

	uq := &SUnion{
		alias:   getTableAliasName(),
		queries: query,
		fields:  fields,
	}

	for i := range fieldNames {
		fields[i] = &SUnionQueryField{name: fieldNames[i], union: uq}
	}

	return uq, nil
}

func (uq *SUnion) Query(f ...IQueryField) *SQuery {
	return DoQuery(uq, f...)
}
