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
	"reflect"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
)

// QueryJoinType is the Join type of SQL query, namely, innerjoin, leftjoin and rightjoin
type QueryJoinType string

const (
	// INNERJOIN represents innerjoin
	INNERJOIN QueryJoinType = "JOIN"

	// LEFTJOIN represents left join
	LEFTJOIN QueryJoinType = "LEFT JOIN"

	// RIGHTJOIN represents right-join
	RIGHTJOIN QueryJoinType = "RIGHT JOIN"

	// FULLJOIN  QueryJoinType = "FULLJOIN"
)

// sQueryJoin represents the state of a Join Query
type sQueryJoin struct {
	jointype  QueryJoinType
	from      IQuerySource
	condition ICondition
}

// SQuery is a data structure represents a SQL query in the form of
//     SELECT ... FROM ... JOIN ... ON ... WHERE ... GROUP BY ... ORDER BY ... HAVING ...
type SQuery struct {
	rawSql   string
	fields   []IQueryField
	distinct bool
	from     IQuerySource
	joins    []sQueryJoin
	where    ICondition
	groupBy  []IQueryField
	orderBy  []sQueryOrder
	// having   ICondition
	limit  int
	offset int

	fieldCache map[string]IQueryField

	snapshot string

	db *SDatabase
}

// IsGroupBy returns wether the query contains group by clauses
func (tq *SQuery) IsGroupBy() bool {
	return len(tq.groupBy) > 0
}

func (tq *SQuery) HasField(f IQueryField) bool {
	if len(tq.fields) == 0 {
		return false
	}
	for i := range tq.fields {
		fi := tq.fields[i]
		// log.Debugf("field at %d: %s", i, fi.Name())
		if fi.Name() == f.Name() {
			return true
		}
	}
	return false
}

// AppendField appends query field to a query
func (tq *SQuery) AppendField(f ...IQueryField) *SQuery {
	// log.Debugf("AppendField tq has fields %d", len(tq.fields))
	for i := range f {
		if !tq.HasField(f[i]) {
			tq.fields = append(tq.fields, f[i])
		}
	}
	return tq
}

func (tq *SQuery) ResetFields() *SQuery {
	tq.fields = nil
	return tq
}

// Query of SSubQuery generates a new query from a subquery
func (sq *SSubQuery) Query(f ...IQueryField) *SQuery {
	return DoQuery(sq, f...)
}

// Query of STable generates a new query from a table
func (tbl *STable) Query(f ...IQueryField) *SQuery {
	return DoQuery(tbl, f...)
}

// Query of STableSpec generates a new query from a STableSpec instance
func (ts *STableSpec) Query(f ...IQueryField) *SQuery {
	return ts.Instance().Query(f...)
}

// QueryOrderType indicates the query order type, either ASC or DESC
type QueryOrderType string

const (
	// SQL_ORDER_ASC represents Ascending order
	SQL_ORDER_ASC QueryOrderType = "ASC"

	// SQL_ORDER_DESC represents Descending order
	SQL_ORDER_DESC QueryOrderType = "DESC"
)

// Equals of QueryOrderType determines whether two order type identical
func (qot QueryOrderType) Equals(orderType string) bool {
	if strings.ToUpper(orderType) == string(qot) {
		return true
	}
	return false
}

// internal structure to store state of query order
type sQueryOrder struct {
	field IQueryField
	order QueryOrderType
}

func (tq *SQuery) _orderBy(order QueryOrderType, fields []IQueryField) *SQuery {
	if tq.orderBy == nil {
		tq.orderBy = make([]sQueryOrder, 0)
	}
	for i := range fields {
		tq.orderBy = append(tq.orderBy, sQueryOrder{field: fields[i], order: order})
	}
	return tq
}

// Asc of SQuery does query in ascending order of specified fields
func (tq *SQuery) Asc(fields ...interface{}) *SQuery {
	return tq._orderBy(SQL_ORDER_ASC, convertQueryField(tq, fields))
}

// Desc of SQuery does query in descending order of specified fields
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

// GroupBy of SQuery does query group by specified fields
func (tq *SQuery) GroupBy(f ...interface{}) *SQuery {
	if tq.groupBy == nil {
		tq.groupBy = make([]IQueryField, 0)
	}
	qfs := convertQueryField(tq, f)
	tq.groupBy = append(tq.groupBy, qfs...)
	return tq
}

// Limit of SQuery adds limit to a query
func (tq *SQuery) Limit(limit int) *SQuery {
	tq.limit = limit
	return tq
}

// Offset of SQuery adds offset to a query
func (tq *SQuery) Offset(offset int) *SQuery {
	tq.offset = offset
	return tq
}

// QueryFields of SQuery returns fields in SELECT clause of a query
func (tq *SQuery) QueryFields() []IQueryField {
	if len(tq.fields) > 0 {
		return tq.fields
	}
	return tq.from.Fields()
}

// String of SQuery implemetation of SQuery for IQuery
func (tq *SQuery) String(fields ...IQueryField) string {
	sql := queryString(tq, fields...)
	// log.Debugf("Query: %s", sql)
	return sql
}

// Join of SQuery joins query with another IQuerySource on specified condition
func (tq *SQuery) Join(from IQuerySource, on ICondition) *SQuery {
	return tq._join(from, on, INNERJOIN)
}

// LeftJoin of SQuery left-joins query with another IQuerySource on specified condition
func (tq *SQuery) LeftJoin(from IQuerySource, on ICondition) *SQuery {
	return tq._join(from, on, LEFTJOIN)
}

// RightJoin of SQuery right-joins query with another IQuerySource on specified condition
func (tq *SQuery) RightJoin(from IQuerySource, on ICondition) *SQuery {
	return tq._join(from, on, RIGHTJOIN)
}

/*func (tq *SQuery) FullJoin(from IQuerySource, on ICondition) *SQuery {
	return tq._join(from, on, FULLJOIN)
}*/

func (tq *SQuery) _join(from IQuerySource, on ICondition, joinType QueryJoinType) *SQuery {
	if from.database() != tq.db {
		panic(fmt.Sprintf("Cannot join across databases %s!=%s", tq.db.name, from.database().name))
	}
	if tq.joins == nil {
		tq.joins = make([]sQueryJoin, 0)
	}
	qj := sQueryJoin{jointype: joinType, from: from, condition: on}
	tq.joins = append(tq.joins, qj)
	return tq
}

// Variables implementation of SQuery for IQuery
func (tq *SQuery) Variables() []interface{} {
	vars := make([]interface{}, 0)
	var fromvars []interface{}
	fields := tq.fields
	for i := range fields {
		fromvars = fields[i].Variables()
		vars = append(vars, fromvars...)
	}
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
	/*if tq.having != nil {
		fromvars = tq.having.Variables()
		vars = append(vars, fromvars...)
	}*/
	return vars
}

// Distinct of SQuery indicates a distinct query results
func (tq *SQuery) Distinct() *SQuery {
	tq.distinct = true
	return tq
}

// SubQuery of SQuery generates a SSubQuery from a Query
func (tq *SQuery) SubQuery() *SSubQuery {
	sq := SSubQuery{query: tq, alias: getTableAliasName()}
	return &sq
}

func (tq *SQuery) database() *SDatabase {
	return tq.db
}

// Row of SQuery returns an instance of  sql.Row for native data fetching
func (tq *SQuery) Row() *sql.Row {
	sqlstr := tq.String()
	vars := tq.Variables()
	if DEBUG_SQLCHEMY {
		sqlDebug(sqlstr, vars)
	}
	if tq.db == nil {
		panic("tq.db")
	}
	if tq.db.db == nil {
		panic("tq.db.db")
	}
	return tq.db.db.QueryRow(sqlstr, vars...)
}

// Rows of SQuery returns an instance of sql.Rows for native data fetching
func (tq *SQuery) Rows() (*sql.Rows, error) {
	sqlstr := tq.String()
	vars := tq.Variables()
	if DEBUG_SQLCHEMY {
		sqlDebug(sqlstr, vars)
	}
	return tq.db.db.Query(sqlstr, vars...)
}

// Count of SQuery returns the count of a query
// use CountWithError instead
// deprecated
func (tq *SQuery) Count() int {
	cnt, _ := tq.CountWithError()
	return cnt
}

func (tq *SQuery) CountQuery() *SQuery {
	tq2 := *tq
	tq2.limit = 0
	tq2.offset = 0
	cq := &SQuery{
		fields: []IQueryField{
			COUNT("count"),
		},
		from: tq2.SubQuery(),
		db:   tq.database(),
	}
	return cq
}

// CountWithError of SQuery returns the row count of a query
func (tq *SQuery) CountWithError() (int, error) {
	cq := tq.CountQuery()
	count := 0
	err := cq.Row().Scan(&count)
	if err == nil {
		return count, nil
	}
	log.Errorf("SQuery count %s failed: %s", cq.String(), err)
	return -1, err
}

// Field implementation of SQuery for IQuery
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

// IRowScanner is an interface for sql data fetching
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
			results[f] = GetStringValue(value)
		}
	}
	return results, nil
}

func (tq *SQuery) rowScan2StringMap(row IRowScanner) (map[string]string, error) {
	queryFields := tq.QueryFields()
	fields := make([]string, len(queryFields))
	for i, f := range queryFields {
		fields[i] = f.Name()
	}
	return rowScan2StringMap(fields, row)
}

// FirstStringMap returns query result of the first row in a stringmap(map[string]string)
func (tq *SQuery) FirstStringMap() (map[string]string, error) {
	return tq.rowScan2StringMap(tq.Row())
}

// AllStringMap returns query result of all rows in an array of stringmap(map[string]string)
func (tq *SQuery) AllStringMap() ([]map[string]string, error) {
	rows, err := tq.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := make([]map[string]string, 0)
	for rows.Next() {
		result, err := tq.rowScan2StringMap(rows)
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

// First return query result of first row and store the result in a data struct
func (tq *SQuery) First(dest interface{}) error {
	mapResult, err := tq.FirstStringMap()
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

// All return query results of all rows and store the result in an array of data struct
func (tq *SQuery) All(dest interface{}) error {
	arrayType := reflect.TypeOf(dest).Elem()

	if arrayType.Kind() != reflect.Array && arrayType.Kind() != reflect.Slice {
		return errors.Wrap(ErrNeedsArray, "dest is not an array or slice")
	}
	elemType := arrayType.Elem()

	mapResults, err := tq.AllStringMap()
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

// Row2Map is a utility function that fetch stringmap(map[string]string) from a native sql.Row or sql.Rows
func (tq *SQuery) Row2Map(row IRowScanner) (map[string]string, error) {
	return tq.rowScan2StringMap(row)
}

// RowMap2Struct is a utility function that fetch struct from a native sql.Row or sql.Rows
func (tq *SQuery) RowMap2Struct(result map[string]string, dest interface{}) error {
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

// Row2Struct is a utility function that fill a struct with the value of a sql.Row or sql.Rows
func (tq *SQuery) Row2Struct(row IRowScanner, dest interface{}) error {
	result, err := tq.rowScan2StringMap(row)
	if err != nil {
		return err
	}
	return tq.RowMap2Struct(result, dest)
}

// Snapshot of SQuery take a snapshot of the query, so we can tell wether the query is modified later by comparing the SQL with snapshot
func (tq *SQuery) Snapshot() *SQuery {
	tq.snapshot = tq.String()
	return tq
}

// IsAltered of SQuery indicates whether a query was altered. By comparing with the saved query snapshot, we can tell whether a query is altered
func (tq *SQuery) IsAltered() bool {
	if len(tq.snapshot) == 0 {
		panic(fmt.Sprintf("Query %s has never been snapshot when IsAltered called", tq.String()))
	}
	return tq.String() != tq.snapshot
}
