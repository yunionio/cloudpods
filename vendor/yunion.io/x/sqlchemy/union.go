package sqlchemy

import (
	"fmt"
	"strings"

	"yunion.io/x/log"
)

type SUnionQueryField struct {
	name string
}

func (sqf *SUnionQueryField) Expression() string {
	return sqf.name
}

func (sqf *SUnionQueryField) Name() string {
	return sqf.name
}

func (sqf *SUnionQueryField) Reference() string {
	return sqf.name
}

func (sqf *SUnionQueryField) Label(label string) IQueryField {
	return sqf
}

type SUnionQuery struct {
	queries []IQuery
	fields  []IQueryField
	orderBy []SQueryOrder
	limit   int
	offset  int
}

func (uq *SUnionQuery) String() string {
	var buf strings.Builder
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
	return buf.String()
}

func (tq *SUnionQuery) _orderBy(order QueryOrderType, fields []IQueryField) *SUnionQuery {
	if tq.orderBy == nil {
		tq.orderBy = make([]SQueryOrder, 0)
	}
	for _, f := range fields {
		tq.orderBy = append(tq.orderBy, SQueryOrder{field: f, order: order})
	}
	return tq
}

func (tq *SUnionQuery) Asc(fields ...interface{}) *SUnionQuery {
	return tq._orderBy(SQL_ORDER_ASC, convertQueryField(tq, fields))
}

func (tq *SUnionQuery) Desc(fields ...interface{}) *SUnionQuery {
	return tq._orderBy(SQL_ORDER_DESC, convertQueryField(tq, fields))
}

func (uq *SUnionQuery) Limit(limit int) *SUnionQuery {
	uq.limit = limit
	return uq
}

func (uq *SUnionQuery) Offset(offset int) *SUnionQuery {
	uq.offset = offset
	return uq
}

func (uq *SUnionQuery) QueryFields() []IQueryField {
	return uq.fields
}

func (uq *SUnionQuery) Field(name string) IQueryField {
	for i := range uq.fields {
		if name == uq.fields[i].Name() {
			return uq.fields[i]
		}
	}
	return nil
}

func (uq *SUnionQuery) Variables() []interface{} {
	ret := make([]interface{}, 0)
	for i := range uq.queries {
		ret = append(ret, uq.queries[i].Variables()...)
	}
	return ret
}

func (uq *SUnionQuery) SubQuery() *SSubQuery {
	sq := SSubQuery{query: uq, alias: getTableAliasName()}
	return &sq
}

func Union(query ...IQuery) *SUnionQuery {
	fieldNames := make([]string, 0)
	for _, f := range query[0].QueryFields() {
		fieldNames = append(fieldNames, f.Name())
	}

	for i := 1; i < len(query); i += 1 {
		qfields := query[i].QueryFields()
		if len(fieldNames) != len(qfields) {
			log.Fatalf("cannot union, number of fields not match!")
		}
		for i := range qfields {
			if fieldNames[i] != qfields[i].Name() {
				log.Fatalf("cannot union, name of fields not match!")
			}
		}
	}

	fields := make([]IQueryField, len(fieldNames))
	for i := range fieldNames {
		fields[i] = &SUnionQueryField{name: fieldNames[i]}
	}

	uq := &SUnionQuery{
		queries: query,
		fields:  fields,
	}

	return uq
}
