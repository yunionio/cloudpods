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

type rwFilter struct {
}

//DmDriver
func (rwf *rwFilter) DmDriverOpen(filterChain *filterChain, d *DmDriver, dsn string) (*DmConnection, error) {
	return filterChain.DmDriverOpen(d, dsn)
}

func (rwf *rwFilter) DmDriverOpenConnector(filterChain *filterChain, d *DmDriver, dsn string) (*DmConnector, error) {
	return filterChain.DmDriverOpenConnector(d, dsn)
}

//DmConnector
func (rwf *rwFilter) DmConnectorConnect(filterChain *filterChain, c *DmConnector, ctx context.Context) (*DmConnection, error) {
	return RWUtil.connect(c, ctx)
}

func (rwf *rwFilter) DmConnectorDriver(filterChain *filterChain, c *DmConnector) *DmDriver {
	return filterChain.DmConnectorDriver(c)
}

//DmConnection
func (rwf *rwFilter) DmConnectionBegin(filterChain *filterChain, c *DmConnection) (*DmConnection, error) {
	if RWUtil.isStandbyAlive(c) {
		_, err := c.rwInfo.connStandby.begin()
		if err != nil {
			RWUtil.afterExceptionOnStandby(c, err)
		}
	}

	return filterChain.DmConnectionBegin(c)
}

func (rwf *rwFilter) DmConnectionBeginTx(filterChain *filterChain, c *DmConnection, ctx context.Context, opts driver.TxOptions) (*DmConnection, error) {
	if RWUtil.isStandbyAlive(c) {
		_, err := c.rwInfo.connStandby.beginTx(ctx, opts)
		if err != nil {
			RWUtil.afterExceptionOnStandby(c, err)
		}
	}

	return filterChain.DmConnectionBeginTx(c, ctx, opts)
}

func (rwf *rwFilter) DmConnectionCommit(filterChain *filterChain, c *DmConnection) error {
	if RWUtil.isStandbyAlive(c) {
		err := c.rwInfo.connStandby.commit()
		if err != nil {
			RWUtil.afterExceptionOnStandby(c, err)
		}
	}

	return filterChain.DmConnectionCommit(c)
}

func (rwf *rwFilter) DmConnectionRollback(filterChain *filterChain, c *DmConnection) error {
	if RWUtil.isStandbyAlive(c) {
		err := c.rwInfo.connStandby.rollback()
		if err != nil {
			RWUtil.afterExceptionOnStandby(c, err)
		}
	}

	return filterChain.DmConnectionRollback(c)
}

func (rwf *rwFilter) DmConnectionClose(filterChain *filterChain, c *DmConnection) error {
	if RWUtil.isStandbyAlive(c) {
		err := c.rwInfo.connStandby.close()
		if err != nil {
			RWUtil.afterExceptionOnStandby(c, err)
		}
	}

	return filterChain.DmConnectionClose(c)
}

func (rwf *rwFilter) DmConnectionPing(filterChain *filterChain, c *DmConnection, ctx context.Context) error {
	return filterChain.DmConnectionPing(c, ctx)
}

func (rwf *rwFilter) DmConnectionExec(filterChain *filterChain, c *DmConnection, query string, args []driver.Value) (*DmResult, error) {
	ret, err := RWUtil.executeByConn(c, query, func() (interface{}, error) {
		return c.rwInfo.connCurrent.exec(query, args)
	}, func(otherConn *DmConnection) (interface{}, error) {
		return otherConn.exec(query, args)
	})
	if err != nil {
		return nil, err
	}
	return ret.(*DmResult), nil
}

func (rwf *rwFilter) DmConnectionExecContext(filterChain *filterChain, c *DmConnection, ctx context.Context, query string, args []driver.NamedValue) (*DmResult, error) {
	ret, err := RWUtil.executeByConn(c, query, func() (interface{}, error) {
		return c.rwInfo.connCurrent.execContext(ctx, query, args)
	}, func(otherConn *DmConnection) (interface{}, error) {
		return otherConn.execContext(ctx, query, args)
	})
	if err != nil {
		return nil, err
	}
	return ret.(*DmResult), nil
}

func (rwf *rwFilter) DmConnectionQuery(filterChain *filterChain, c *DmConnection, query string, args []driver.Value) (*DmRows, error) {
	ret, err := RWUtil.executeByConn(c, query, func() (interface{}, error) {
		return c.rwInfo.connCurrent.query(query, args)
	}, func(otherConn *DmConnection) (interface{}, error) {
		return otherConn.query(query, args)
	})
	if err != nil {
		return nil, err
	}
	return ret.(*DmRows), nil
}

func (rwf *rwFilter) DmConnectionQueryContext(filterChain *filterChain, c *DmConnection, ctx context.Context, query string, args []driver.NamedValue) (*DmRows, error) {
	ret, err := RWUtil.executeByConn(c, query, func() (interface{}, error) {
		return c.rwInfo.connCurrent.queryContext(ctx, query, args)
	}, func(otherConn *DmConnection) (interface{}, error) {
		return otherConn.queryContext(ctx, query, args)
	})
	if err != nil {
		return nil, err
	}
	return ret.(*DmRows), nil
}

func (rwf *rwFilter) DmConnectionPrepare(filterChain *filterChain, c *DmConnection, query string) (*DmStatement, error) {
	stmt, err := c.prepare(query)
	if err != nil {
		return nil, err
	}
	stmt.rwInfo.stmtCurrent = stmt
	stmt.rwInfo.readOnly = RWUtil.checkReadonlyByStmt(stmt)
	if RWUtil.isCreateStandbyStmt(stmt) {
		stmt.rwInfo.stmtStandby, err = c.rwInfo.connStandby.prepare(query)
		if err == nil {
			stmt.rwInfo.stmtCurrent = stmt.rwInfo.stmtStandby
		} else {
			RWUtil.afterExceptionOnStandby(c, err)
		}
	}
	return stmt, nil
}

func (rwf *rwFilter) DmConnectionPrepareContext(filterChain *filterChain, c *DmConnection, ctx context.Context, query string) (*DmStatement, error) {
	stmt, err := c.prepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	stmt.rwInfo.stmtCurrent = stmt
	stmt.rwInfo.readOnly = RWUtil.checkReadonlyByStmt(stmt)
	if RWUtil.isCreateStandbyStmt(stmt) {
		stmt.rwInfo.stmtStandby, err = c.rwInfo.connStandby.prepareContext(ctx, query)
		if err == nil {
			stmt.rwInfo.stmtCurrent = stmt.rwInfo.stmtStandby
		} else {
			RWUtil.afterExceptionOnStandby(c, err)
		}
	}
	return stmt, nil
}

func (rwf *rwFilter) DmConnectionResetSession(filterChain *filterChain, c *DmConnection, ctx context.Context) error {
	if RWUtil.isStandbyAlive(c) {
		err := c.rwInfo.connStandby.resetSession(ctx)
		if err != nil {
			RWUtil.afterExceptionOnStandby(c, err)
		}
	}

	return filterChain.DmConnectionResetSession(c, ctx)
}

func (rwf *rwFilter) DmConnectionCheckNamedValue(filterChain *filterChain, c *DmConnection, nv *driver.NamedValue) error {
	return filterChain.DmConnectionCheckNamedValue(c, nv)
}

//DmStatement
func (rwf *rwFilter) DmStatementClose(filterChain *filterChain, s *DmStatement) error {
	if RWUtil.isStandbyStatementValid(s) {
		err := s.rwInfo.stmtStandby.close()
		if err != nil {
			RWUtil.afterExceptionOnStandby(s.dmConn, err)
		}
	}

	return filterChain.DmStatementClose(s)
}

func (rwf *rwFilter) DmStatementNumInput(filterChain *filterChain, s *DmStatement) int {
	return filterChain.DmStatementNumInput(s)
}

func (rwf *rwFilter) DmStatementExec(filterChain *filterChain, s *DmStatement, args []driver.Value) (*DmResult, error) {
	ret, err := RWUtil.executeByStmt(s, func() (interface{}, error) {
		return s.rwInfo.stmtCurrent.exec(args)
	}, func(otherStmt *DmStatement) (interface{}, error) {
		return otherStmt.exec(args)
	})
	if err != nil {
		return nil, err
	}
	return ret.(*DmResult), nil
}

func (rwf *rwFilter) DmStatementExecContext(filterChain *filterChain, s *DmStatement, ctx context.Context, args []driver.NamedValue) (*DmResult, error) {
	ret, err := RWUtil.executeByStmt(s, func() (interface{}, error) {
		return s.rwInfo.stmtCurrent.execContext(ctx, args)
	}, func(otherStmt *DmStatement) (interface{}, error) {
		return otherStmt.execContext(ctx, args)
	})
	if err != nil {
		return nil, err
	}
	return ret.(*DmResult), nil
}

func (rwf *rwFilter) DmStatementQuery(filterChain *filterChain, s *DmStatement, args []driver.Value) (*DmRows, error) {
	ret, err := RWUtil.executeByStmt(s, func() (interface{}, error) {
		return s.rwInfo.stmtCurrent.query(args)
	}, func(otherStmt *DmStatement) (interface{}, error) {
		return otherStmt.query(args)
	})
	if err != nil {
		return nil, err
	}
	return ret.(*DmRows), nil
}

func (rwf *rwFilter) DmStatementQueryContext(filterChain *filterChain, s *DmStatement, ctx context.Context, args []driver.NamedValue) (*DmRows, error) {
	ret, err := RWUtil.executeByStmt(s, func() (interface{}, error) {
		return s.rwInfo.stmtCurrent.queryContext(ctx, args)
	}, func(otherStmt *DmStatement) (interface{}, error) {
		return otherStmt.queryContext(ctx, args)
	})
	if err != nil {
		return nil, err
	}
	return ret.(*DmRows), nil
}

func (rwf *rwFilter) DmStatementCheckNamedValue(filterChain *filterChain, s *DmStatement, nv *driver.NamedValue) error {
	return filterChain.DmStatementCheckNamedValue(s, nv)
}

//DmResult
func (rwf *rwFilter) DmResultLastInsertId(filterChain *filterChain, r *DmResult) (int64, error) {
	return filterChain.DmResultLastInsertId(r)
}

func (rwf *rwFilter) DmResultRowsAffected(filterChain *filterChain, r *DmResult) (int64, error) {
	return filterChain.DmResultRowsAffected(r)
}

//DmRows
func (rwf *rwFilter) DmRowsColumns(filterChain *filterChain, r *DmRows) []string {
	return filterChain.DmRowsColumns(r)
}

func (rwf *rwFilter) DmRowsClose(filterChain *filterChain, r *DmRows) error {
	return filterChain.DmRowsClose(r)
}

func (rwf *rwFilter) DmRowsNext(filterChain *filterChain, r *DmRows, dest []driver.Value) error {
	return filterChain.DmRowsNext(r, dest)
}

func (rwf *rwFilter) DmRowsHasNextResultSet(filterChain *filterChain, r *DmRows) bool {
	return filterChain.DmRowsHasNextResultSet(r)
}

func (rwf *rwFilter) DmRowsNextResultSet(filterChain *filterChain, r *DmRows) error {
	return filterChain.DmRowsNextResultSet(r)
}

func (rwf *rwFilter) DmRowsColumnTypeScanType(filterChain *filterChain, r *DmRows, index int) reflect.Type {
	return filterChain.DmRowsColumnTypeScanType(r, index)
}

func (rwf *rwFilter) DmRowsColumnTypeDatabaseTypeName(filterChain *filterChain, r *DmRows, index int) string {
	return filterChain.DmRowsColumnTypeDatabaseTypeName(r, index)
}

func (rwf *rwFilter) DmRowsColumnTypeLength(filterChain *filterChain, r *DmRows, index int) (length int64, ok bool) {
	return filterChain.DmRowsColumnTypeLength(r, index)
}

func (rwf *rwFilter) DmRowsColumnTypeNullable(filterChain *filterChain, r *DmRows, index int) (nullable, ok bool) {
	return filterChain.DmRowsColumnTypeNullable(r, index)
}

func (rwf *rwFilter) DmRowsColumnTypePrecisionScale(filterChain *filterChain, r *DmRows, index int) (precision, scale int64, ok bool) {
	return filterChain.DmRowsColumnTypePrecisionScale(r, index)
}
