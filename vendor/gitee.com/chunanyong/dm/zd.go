/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package dm

import (
	"context"
	"database/sql/driver"
	"reflect"
)

type filterChain struct {
	filters []filter
	fpos    int
}

func newFilterChain(filters []filter) *filterChain {
	fc := new(filterChain)
	fc.filters = filters
	fc.fpos = 0
	return fc
}

func (filterChain *filterChain) reset() *filterChain {
	filterChain.fpos = 0
	return filterChain
}

func (filterChain *filterChain) DmDriverOpen(d *DmDriver, dsn string) (*DmConnection, error) {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmDriverOpen(filterChain, d, dsn)
	}

	return d.open(dsn)
}

func (filterChain *filterChain) DmDriverOpenConnector(d *DmDriver, dsn string) (*DmConnector, error) {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmDriverOpenConnector(filterChain, d, dsn)
	}

	return d.openConnector(dsn)
}

//DmConnector
func (filterChain *filterChain) DmConnectorConnect(c *DmConnector, ctx context.Context) (*DmConnection, error) {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmConnectorConnect(filterChain, c, ctx)
	}

	return c.connect(ctx)
}

func (filterChain *filterChain) DmConnectorDriver(c *DmConnector) *DmDriver {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmConnectorDriver(filterChain, c)
	}

	return c.driver()
}

//DmConnection
func (filterChain *filterChain) DmConnectionBegin(c *DmConnection) (*DmConnection, error) {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmConnectionBegin(filterChain, c)
	}

	return c.begin()
}
func (filterChain *filterChain) DmConnectionBeginTx(c *DmConnection, ctx context.Context, opts driver.TxOptions) (*DmConnection, error) {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmConnectionBeginTx(filterChain, c, ctx, opts)
	}

	return c.beginTx(ctx, opts)
}

func (filterChain *filterChain) DmConnectionCommit(c *DmConnection) error {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmConnectionCommit(filterChain, c)
	}

	return c.commit()
}

func (filterChain *filterChain) DmConnectionRollback(c *DmConnection) error {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmConnectionRollback(filterChain, c)
	}

	return c.rollback()
}

func (filterChain *filterChain) DmConnectionClose(c *DmConnection) error {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmConnectionClose(filterChain, c)
	}

	return c.close()
}

func (filterChain *filterChain) DmConnectionPing(c *DmConnection, ctx context.Context) error {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmConnectionPing(filterChain, c, ctx)
	}

	return c.ping(ctx)
}

func (filterChain *filterChain) DmConnectionExec(c *DmConnection, query string, args []driver.Value) (*DmResult, error) {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmConnectionExec(filterChain, c, query, args)
	}

	return c.exec(query, args)
}

func (filterChain *filterChain) DmConnectionExecContext(c *DmConnection, ctx context.Context, query string, args []driver.NamedValue) (*DmResult, error) {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmConnectionExecContext(filterChain, c, ctx, query, args)
	}

	return c.execContext(ctx, query, args)
}

func (filterChain *filterChain) DmConnectionQuery(c *DmConnection, query string, args []driver.Value) (*DmRows, error) {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmConnectionQuery(filterChain, c, query, args)
	}

	return c.query(query, args)
}

func (filterChain *filterChain) DmConnectionQueryContext(c *DmConnection, ctx context.Context, query string, args []driver.NamedValue) (*DmRows, error) {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmConnectionQueryContext(filterChain, c, ctx, query, args)
	}

	return c.queryContext(ctx, query, args)
}

func (filterChain *filterChain) DmConnectionPrepare(c *DmConnection, query string) (*DmStatement, error) {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmConnectionPrepare(filterChain, c, query)
	}

	return c.prepare(query)
}

func (filterChain *filterChain) DmConnectionPrepareContext(c *DmConnection, ctx context.Context, query string) (*DmStatement, error) {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmConnectionPrepareContext(filterChain, c, ctx, query)
	}

	return c.prepareContext(ctx, query)
}

func (filterChain *filterChain) DmConnectionResetSession(c *DmConnection, ctx context.Context) error {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmConnectionResetSession(filterChain, c, ctx)
	}

	return c.resetSession(ctx)
}

func (filterChain *filterChain) DmConnectionCheckNamedValue(c *DmConnection, nv *driver.NamedValue) error {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmConnectionCheckNamedValue(filterChain, c, nv)
	}

	return c.checkNamedValue(nv)
}

//DmStatement
func (filterChain *filterChain) DmStatementClose(s *DmStatement) error {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmStatementClose(filterChain, s)
	}

	return s.close()
}

func (filterChain *filterChain) DmStatementNumInput(s *DmStatement) int {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmStatementNumInput(filterChain, s)
	}

	return s.numInput()
}

func (filterChain *filterChain) DmStatementExec(s *DmStatement, args []driver.Value) (*DmResult, error) {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmStatementExec(filterChain, s, args)
	}

	return s.exec(args)
}

func (filterChain *filterChain) DmStatementExecContext(s *DmStatement, ctx context.Context, args []driver.NamedValue) (*DmResult, error) {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmStatementExecContext(filterChain, s, ctx, args)
	}

	return s.execContext(ctx, args)
}

func (filterChain *filterChain) DmStatementQuery(s *DmStatement, args []driver.Value) (*DmRows, error) {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmStatementQuery(filterChain, s, args)
	}

	return s.query(args)
}

func (filterChain *filterChain) DmStatementQueryContext(s *DmStatement, ctx context.Context, args []driver.NamedValue) (*DmRows, error) {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmStatementQueryContext(filterChain, s, ctx, args)
	}

	return s.queryContext(ctx, args)
}

func (filterChain *filterChain) DmStatementCheckNamedValue(s *DmStatement, nv *driver.NamedValue) error {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmStatementCheckNamedValue(filterChain, s, nv)
	}

	return s.checkNamedValue(nv)
}

//DmResult
func (filterChain *filterChain) DmResultLastInsertId(r *DmResult) (int64, error) {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmResultLastInsertId(filterChain, r)
	}

	return r.lastInsertId()
}

func (filterChain *filterChain) DmResultRowsAffected(r *DmResult) (int64, error) {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmResultRowsAffected(filterChain, r)
	}

	return r.rowsAffected()
}

//DmRows
func (filterChain *filterChain) DmRowsColumns(r *DmRows) []string {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmRowsColumns(filterChain, r)
	}

	return r.columns()
}

func (filterChain *filterChain) DmRowsClose(r *DmRows) error {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmRowsClose(filterChain, r)
	}

	return r.close()
}

func (filterChain *filterChain) DmRowsNext(r *DmRows, dest []driver.Value) error {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmRowsNext(filterChain, r, dest)
	}

	return r.next(dest)
}

func (filterChain *filterChain) DmRowsHasNextResultSet(r *DmRows) bool {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmRowsHasNextResultSet(filterChain, r)
	}

	return r.hasNextResultSet()
}

func (filterChain *filterChain) DmRowsNextResultSet(r *DmRows) error {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmRowsNextResultSet(filterChain, r)
	}

	return r.nextResultSet()
}

func (filterChain *filterChain) DmRowsColumnTypeScanType(r *DmRows, index int) reflect.Type {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmRowsColumnTypeScanType(filterChain, r, index)
	}

	return r.columnTypeScanType(index)
}

func (filterChain *filterChain) DmRowsColumnTypeDatabaseTypeName(r *DmRows, index int) string {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmRowsColumnTypeDatabaseTypeName(filterChain, r, index)
	}

	return r.columnTypeDatabaseTypeName(index)
}

func (filterChain *filterChain) DmRowsColumnTypeLength(r *DmRows, index int) (length int64, ok bool) {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmRowsColumnTypeLength(filterChain, r, index)
	}

	return r.columnTypeLength(index)
}

func (filterChain *filterChain) DmRowsColumnTypeNullable(r *DmRows, index int) (nullable, ok bool) {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmRowsColumnTypeNullable(filterChain, r, index)
	}

	return r.columnTypeNullable(index)
}

func (filterChain *filterChain) DmRowsColumnTypePrecisionScale(r *DmRows, index int) (precision, scale int64, ok bool) {
	if filterChain.fpos < len(filterChain.filters) {
		f := filterChain.filters[filterChain.fpos]
		filterChain.fpos++
		return f.DmRowsColumnTypePrecisionScale(filterChain, r, index)
	}

	return r.columnTypePrecisionScale(index)
}
