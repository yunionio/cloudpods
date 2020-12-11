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
	"database/sql"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
)

type IQuery interface {
	// queryString
	String(fields ...IQueryField) string
	// fields after select
	QueryFields() []IQueryField
	// variables in statement
	Variables() []interface{}
	// convert to a subquery
	SubQuery() *SSubQuery
	// reference to a field by name
	Field(name string) IQueryField
}

type IQuerySource interface {
	// string in select ... from (expresson here)
	Expression() string
	// alias in select ... from (express) as alias
	Alias() string
	// variables in statement
	Variables() []interface{}
	// reference to a field by name, optionally giving an alias name
	Field(id string, alias ...string) IQueryField
	// return all the fields that this source provides
	Fields() []IQueryField
}

type IQueryField interface {
	// the string after select
	Expression() string
	// the name of thie field
	Name() string
	// the string in where clause
	Reference() string
	// give this field an alias name
	Label(label string) IQueryField
}

func (tbl *STable) Expression() string {
	return tbl.spec.Expression()
}

func (tbl *STable) Alias() string {
	return tbl.alias
}

func (tbl *STable) Variables() []interface{} {
	return []interface{}{}
}

type QueryJoinType string

const (
	INNERJOIN QueryJoinType = "JOIN"
	LEFTJOIN  QueryJoinType = "LEFT JOIN"
	RIGHTJOIN QueryJoinType = "RIGHT JOIN"
	// FULLJOIN  QueryJoinType = "FULLJOIN"
)

type SQueryJoin struct {
	jointype  QueryJoinType
	from      IQuerySource
	condition ICondition
}

type SQuery struct {
	rawSql   string
	fields   []IQueryField
	distinct bool
	from     IQuerySource
	joins    []SQueryJoin
	where    ICondition
	groupBy  []IQueryField
	orderBy  []SQueryOrder
	having   ICondition
	limit    int
	offset   int

	fieldCache map[string]IQueryField

	snapshot string
}

type SSubQuery struct {
	query IQuery
	alias string

	referedFields map[string]IQueryField
}

type SSubQueryField struct {
	field IQueryField
	query *SSubQuery
	alias string
}

func (sqf *SSubQueryField) Expression() string {
	if len(sqf.alias) > 0 {
		return fmt.Sprintf("`%s`.`%s` AS `%s`", sqf.query.alias, sqf.field.Name(), sqf.alias)
	} else {
		return fmt.Sprintf("`%s`.`%s`", sqf.query.alias, sqf.field.Name())
	}
}

func (sqf *SSubQueryField) Name() string {
	if len(sqf.alias) > 0 {
		return sqf.alias
	} else {
		return sqf.field.Name()
	}
}

func (sqf *SSubQueryField) Reference() string {
	return fmt.Sprintf("`%s`.`%s`", sqf.query.alias, sqf.Name())
}

func (sqf *SSubQueryField) Label(label string) IQueryField {
	if len(label) > 0 && label != sqf.field.Name() {
		sqf.alias = label
	}
	return sqf
}

func (sq *SSubQuery) Expression() string {
	fields := make([]IQueryField, 0)
	for k := range sq.referedFields {
		fields = append(fields, sq.referedFields[k])
	}
	// Make sure the order of the fields
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Name() < fields[j].Name()
	})
	return fmt.Sprintf("(%s)", sq.query.String(fields...))
}

func (sq *SSubQuery) Alias() string {
	return sq.alias
}

func (sq *SSubQuery) Variables() []interface{} {
	return sq.query.Variables()
}

func (sq *SSubQuery) findField(id string) IQueryField {
	if sq.referedFields == nil {
		sq.referedFields = make(map[string]IQueryField)
	}
	if _, ok := sq.referedFields[id]; ok {
		return sq.referedFields[id]
	}
	queryFields := sq.query.QueryFields()
	for i := range queryFields {
		if queryFields[i].Name() == id {
			sq.referedFields[id] = sq.query.Field(queryFields[i].Name())
			return sq.referedFields[id]
		}
	}
	return nil
}

func (sq *SSubQuery) Field(id string, alias ...string) IQueryField {
	f := sq.findField(id)
	if f == nil {
		return nil
	}
	sqf := SSubQueryField{query: sq, field: f}
	if len(alias) > 0 {
		sqf.Label(alias[0])
	}
	return &sqf
}

func (sq *SSubQuery) Fields() []IQueryField {
	ret := make([]IQueryField, 0)
	for _, f := range sq.query.QueryFields() {
		sqf := SSubQueryField{query: sq, field: f}
		ret = append(ret, &sqf)
	}
	return ret
}

func DoQuery(from IQuerySource, f ...IQueryField) *SQuery {
	// if len(f) == 0 {
	// 	f = from.Fields()
	// }
	tq := SQuery{fields: f, from: from}
	return &tq
}

func (q *SQuery) AppendField(f ...IQueryField) *SQuery {
	q.fields = append(q.fields, f...)
	return q
}

func (table *SSubQuery) Query(f ...IQueryField) *SQuery {
	return DoQuery(table, f...)
}

func (tbl *STable) Query(f ...IQueryField) *SQuery {
	return DoQuery(tbl, f...)
}

func (ts *STableSpec) Query(f ...IQueryField) *SQuery {
	return ts.Instance().Query(f...)
}

type QueryOrderType string

const (
	SQL_ORDER_ASC  QueryOrderType = "ASC"
	SQL_ORDER_DESC QueryOrderType = "DESC"
)

func (qot QueryOrderType) Equals(orderType string) bool {
	if strings.ToUpper(orderType) == string(qot) {
		return true
	} else {
		return false
	}
}

type SQueryOrder struct {
	field IQueryField
	order QueryOrderType
}

func (tq *SQuery) _orderBy(order QueryOrderType, fields []IQueryField) *SQuery {
	if tq.orderBy == nil {
		tq.orderBy = make([]SQueryOrder, 0)
	}
	for i := range fields {
		tq.orderBy = append(tq.orderBy, SQueryOrder{field: fields[i], order: order})
	}
	return tq
}

func (tq *SQuery) Asc(fields ...interface{}) *SQuery {
	return tq._orderBy(SQL_ORDER_ASC, convertQueryField(tq, fields))
}

func (tq *SQuery) Desc(fields ...interface{}) *SQuery {
	return tq._orderBy(SQL_ORDER_DESC, convertQueryField(tq, fields))
}

func convertQueryField(tq IQuery, fields []interface{}) []IQueryField {
	nFields := make([]IQueryField, 0)
	for _, f := range fields {
		switch ff := f.(type) {
		case string:
			nFields = append(nFields, tq.Field(ff))
		case IQueryField:
			nFields = append(nFields, ff)
		default:
			log.Errorf("Invalid query field %s neither string nor IQueryField", f)
		}
	}
	return nFields
}

func (tq *SQuery) GroupBy(f ...interface{}) *SQuery {
	if tq.groupBy == nil {
		tq.groupBy = make([]IQueryField, 0)
	}
	qfs := convertQueryField(tq, f)
	tq.groupBy = append(tq.groupBy, qfs...)
	return tq
}

func (tq *SQuery) Limit(limit int) *SQuery {
	tq.limit = limit
	return tq
}

func (tq *SQuery) Offset(offset int) *SQuery {
	tq.offset = offset
	return tq
}

func (tq *SQuery) QueryFields() []IQueryField {
	if len(tq.fields) > 0 {
		return tq.fields
	} else {
		return tq.from.Fields()
	}
}

func (tq *SQuery) String(fields ...IQueryField) string {
	sql := queryString(tq, fields...)
	// log.Debugf("Query: %s", sql)
	return sql
}

func queryString(tq *SQuery, tmpFields ...IQueryField) string {
	if len(tq.rawSql) > 0 {
		return tq.rawSql
	}

	var buf bytes.Buffer
	buf.WriteString("SELECT ")
	if tq.distinct {
		buf.WriteString("DISTINCT ")
	}
	fields := tq.fields
	if len(fields) == 0 {
		fields = tmpFields
	}
	if len(fields) == 0 {
		fields = tq.QueryFields()
		for i := range fields {
			tq.from.Field(fields[i].Name())
		}
	}
	for i := range fields {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(fields[i].Expression())
	}
	buf.WriteString(" FROM ")
	buf.WriteString(fmt.Sprintf("%s AS `%s`", tq.from.Expression(), tq.from.Alias()))
	for _, join := range tq.joins {
		buf.WriteByte(' ')
		buf.WriteString(string(join.jointype))
		buf.WriteByte(' ')
		buf.WriteString(fmt.Sprintf("%s AS `%s`", join.from.Expression(), join.from.Alias()))
		buf.WriteString(" ON ")
		buf.WriteString(join.condition.WhereClause())
	}
	if tq.where != nil {
		buf.WriteString(" WHERE ")
		buf.WriteString(tq.where.WhereClause())
	}
	if tq.groupBy != nil && len(tq.groupBy) > 0 {
		buf.WriteString(" GROUP BY ")
		for i, f := range tq.groupBy {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(f.Reference())
		}
	}
	if tq.having != nil {
		buf.WriteString(" HAVING ")
		buf.WriteString(tq.having.WhereClause())
	}
	if tq.orderBy != nil && len(tq.orderBy) > 0 {
		buf.WriteString(" ORDER BY ")
		for i, f := range tq.orderBy {
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

func (tq *SQuery) Join(from IQuerySource, on ICondition) *SQuery {
	return tq._join(from, on, INNERJOIN)
}

func (tq *SQuery) LeftJoin(from IQuerySource, on ICondition) *SQuery {
	return tq._join(from, on, LEFTJOIN)
}

func (tq *SQuery) RightJoin(from IQuerySource, on ICondition) *SQuery {
	return tq._join(from, on, RIGHTJOIN)
}

/*func (tq *SQuery) FullJoin(from IQuerySource, on ICondition) *SQuery {
	return tq._join(from, on, FULLJOIN)
}*/

func (tq *SQuery) _join(from IQuerySource, on ICondition, joinType QueryJoinType) *SQuery {
	if tq.joins == nil {
		tq.joins = make([]SQueryJoin, 0)
	}
	qj := SQueryJoin{jointype: joinType, from: from, condition: on}
	tq.joins = append(tq.joins, qj)
	return tq
}

func (tq *SQuery) Variables() []interface{} {
	vars := make([]interface{}, 0)
	var fromvars []interface{}
	if tq.from != nil {
		fromvars = tq.from.Variables()
		vars = append(vars, fromvars...)
	}
	for _, join := range tq.joins {
		fromvars = join.from.Variables()
		vars = append(vars, fromvars...)
		fromvars = join.condition.Variables()
		vars = append(vars, fromvars...)
	}
	if tq.where != nil {
		fromvars = tq.where.Variables()
		vars = append(vars, fromvars...)
	}
	if tq.having != nil {
		fromvars = tq.having.Variables()
		vars = append(vars, fromvars...)
	}
	return vars
}

func (tq *SQuery) Distinct() *SQuery {
	tq.distinct = true
	return tq
}

func (tq *SQuery) SubQuery() *SSubQuery {
	sq := SSubQuery{query: tq, alias: getTableAliasName()}
	return &sq
}

func (tq *SQuery) Row() *sql.Row {
	sqlstr := tq.String()
	vars := tq.Variables()
	if DEBUG_SQLCHEMY {
		log.Debugf("SQuery %s with vars: %s", sqlstr, vars)
	}
	return _db.QueryRow(sqlstr, vars...)
}

func (tq *SQuery) Rows() (*sql.Rows, error) {
	sqlstr := tq.String()
	vars := tq.Variables()
	if DEBUG_SQLCHEMY {
		log.Debugf("SQuery %s with vars: %s", sqlstr, vars)
	}
	return _db.Query(sqlstr, vars...)
}

// deprecated
func (tq *SQuery) Count() int {
	cnt, _ := tq.CountWithError()
	return cnt
}

func (tq *SQuery) countQuery() *SQuery {
	tq2 := *tq
	tq2.limit = 0
	tq2.offset = 0
	cq := &SQuery{
		fields: []IQueryField{
			COUNT("count"),
		},
		from: tq2.SubQuery(),
	}
	return cq
}

func (tq *SQuery) CountWithError() (int, error) {
	cq := tq.countQuery()
	count := 0
	err := cq.Row().Scan(&count)
	if err == nil {
		return count, nil
	}
	log.Errorf("SQuery count %s failed: %s", cq.String(), err)
	return -1, err
}

func (tq *SQuery) Field(name string) IQueryField {
	f := tq.findField(name)
	if DEBUG_SQLCHEMY && f == nil {
		log.Debugf("cannot find field %s for query", name)
	}
	return f
}

func (tq *SQuery) findField(name string) IQueryField {
	if tq.fieldCache == nil {
		tq.fieldCache = make(map[string]IQueryField)
	}
	if _, ok := tq.fieldCache[name]; ok {
		return tq.fieldCache[name]
	}
	f := tq.internalFindField(name)
	if f != nil {
		tq.fieldCache[name] = f
	}
	return f
}

func (tq *SQuery) internalFindField(name string) IQueryField {
	for _, f := range tq.fields {
		if f.Name() == name {
			// switch f.(type) {
			// case *SFunctionFieldBase:
			// 	log.Warningf("cannot directly reference a function alias, should use Subquery() to enclose the query")
			// }
			return f
		}
	}
	f := tq.from.Field(name)
	if f != nil {
		return f
	}
	/* for _, f := range tq.from.Fields() {
		if f.Name() == name {
			return f
		}
	}*/
	for _, join := range tq.joins {
		f = join.from.Field(name)
		if f != nil {
			return f
		}
		/* for _, f := range join.from.Fields() {
			if f.Name() == name {
				return f
			}
		}*/
	}
	return nil
}

type IRowScanner interface {
	Scan(desc ...interface{}) error
}

func rowScan2StringMap(fields []string, row IRowScanner) (map[string]string, error) {
	targets := make([]interface{}, len(fields))
	for i := range fields {
		var recver interface{}
		targets[i] = &recver
	}
	if err := row.Scan(targets...); err != nil {
		return nil, err
	}
	results := make(map[string]string)
	for i, f := range fields {
		//log.Debugf("%d %s: %s", i, f, targets[i])
		rawValue := reflect.Indirect(reflect.ValueOf(targets[i]))
		if rawValue.Interface() == nil {
			results[f] = ""
		} else {
			value := rawValue.Interface()
			// log.Infof("%s %s", value, reflect.TypeOf(value))
			results[f] = getStringValue(value)
		}
	}
	return results, nil
}

func (q *SQuery) rowScan2StringMap(row IRowScanner) (map[string]string, error) {
	queryFields := q.QueryFields()
	fields := make([]string, len(queryFields))
	for i, f := range queryFields {
		fields[i] = f.Name()
	}
	return rowScan2StringMap(fields, row)
}

func (q *SQuery) FirstStringMap() (map[string]string, error) {
	return q.rowScan2StringMap(q.Row())
}

func (q *SQuery) AllStringMap() ([]map[string]string, error) {
	rows, err := q.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := make([]map[string]string, 0)
	for rows.Next() {
		result, err := q.rowScan2StringMap(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func mapString2Struct(mapResult map[string]string, destValue reflect.Value) error {
	destFields := reflectutils.FetchStructFieldValueSet(destValue)
	var err error
	for k, v := range mapResult {
		if len(v) > 0 {
			fieldValue, ok := destFields.GetValue(k)
			if ok {
				err = setValueBySQLString(fieldValue, v)
				if err != nil {
					log.Errorf("Set field %q value error %s", k, err)
				}
			}
		}
	}
	return err
}

func callAfterQuery(val reflect.Value) {
	afterQueryFunc := val.MethodByName("AfterQuery")
	if afterQueryFunc.IsValid() && !afterQueryFunc.IsNil() {
		afterQueryFunc.Call([]reflect.Value{})
	}
}

func (q *SQuery) First(dest interface{}) error {
	mapResult, err := q.FirstStringMap()
	if err != nil {
		return err
	}
	destPtrValue := reflect.ValueOf(dest)
	if destPtrValue.Kind() != reflect.Ptr {
		return errors.Wrap(ErrNeedsPointer, "input must be a pointer")
	}
	destValue := destPtrValue.Elem()
	err = mapString2Struct(mapResult, destValue)
	if err != nil {
		return err
	}
	callAfterQuery(destPtrValue)
	return nil
}

func (q *SQuery) All(dest interface{}) error {
	arrayType := reflect.TypeOf(dest).Elem()

	if arrayType.Kind() != reflect.Array && arrayType.Kind() != reflect.Slice {
		return errors.Wrap(ErrNeedsArray, "dest is not an array or slice")
	}
	elemType := arrayType.Elem()

	mapResults, err := q.AllStringMap()
	if err != nil {
		return err
	}

	arrayValue := reflect.ValueOf(dest).Elem()
	for _, mapV := range mapResults {
		elemPtrValue := reflect.New(elemType)
		elemValue := reflect.Indirect(elemPtrValue)
		err = mapString2Struct(mapV, elemValue)
		if err != nil {
			break
		}
		callAfterQuery(elemPtrValue)
		newArray := reflect.Append(arrayValue, elemValue)
		arrayValue.Set(newArray)
	}
	return err
}

func (q *SQuery) Row2Map(row IRowScanner) (map[string]string, error) {
	return q.rowScan2StringMap(row)
}

func (q *SQuery) RowMap2Struct(result map[string]string, dest interface{}) error {
	destPtrValue := reflect.ValueOf(dest)
	if destPtrValue.Kind() != reflect.Ptr {
		return errors.Wrap(ErrNeedsPointer, "input must be a pointer")
	}

	destValue := destPtrValue.Elem()
	err := mapString2Struct(result, destValue)
	if err != nil {
		return err
	}
	callAfterQuery(destPtrValue)
	return nil
}

func (q *SQuery) Row2Struct(row IRowScanner, dest interface{}) error {
	result, err := q.rowScan2StringMap(row)
	if err != nil {
		return err
	}
	return q.RowMap2Struct(result, dest)
}

func (q *SQuery) Snapshot() *SQuery {
	q.snapshot = q.String()
	return q
}

func (q *SQuery) IsAltered() bool {
	if len(q.snapshot) == 0 {
		panic(fmt.Sprintf("Query %s has never been snapshot when IsAltered called", q.String()))
	}
	return q.String() != q.snapshot
}
