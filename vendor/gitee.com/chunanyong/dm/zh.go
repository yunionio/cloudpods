/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package dm

import (
	"bytes"
	"context"
	"database/sql/driver"
	"fmt"
	"reflect"
	"time"
)

type statFilter struct {
}

//DmDriver
func (sf *statFilter) DmDriverOpen(filterChain *filterChain, d *DmDriver, dsn string) (*DmConnection, error) {
	conn, err := filterChain.DmDriverOpen(d, dsn)
	if err != nil {
		return nil, err
	}
	conn.statInfo.init(conn)
	conn.statInfo.setConstructNano()
	conn.statInfo.getConnStat().incrementConn()
	return conn, nil
}

func (sf *statFilter) DmDriverOpenConnector(filterChain *filterChain, d *DmDriver, dsn string) (*DmConnector, error) {
	return filterChain.DmDriverOpenConnector(d, dsn)
}

//DmConnector
func (sf *statFilter) DmConnectorConnect(filterChain *filterChain, c *DmConnector, ctx context.Context) (*DmConnection, error) {
	conn, err := filterChain.DmConnectorConnect(c, ctx)
	if err != nil {
		return nil, err
	}
	conn.statInfo.init(conn)
	conn.statInfo.setConstructNano()
	conn.statInfo.getConnStat().incrementConn()
	return conn, nil
}

func (sf *statFilter) DmConnectorDriver(filterChain *filterChain, c *DmConnector) *DmDriver {
	return filterChain.DmConnectorDriver(c)
}

//DmConnection
func (sf *statFilter) DmConnectionBegin(filterChain *filterChain, c *DmConnection) (*DmConnection, error) {
	return filterChain.DmConnectionBegin(c)
}

func (sf *statFilter) DmConnectionBeginTx(filterChain *filterChain, c *DmConnection, ctx context.Context, opts driver.TxOptions) (*DmConnection, error) {
	return filterChain.DmConnectionBeginTx(c, ctx, opts)
}

func (sf *statFilter) DmConnectionCommit(filterChain *filterChain, c *DmConnection) error {
	err := filterChain.DmConnectionCommit(c)
	if err != nil {
		return err
	}
	c.statInfo.getConnStat().incrementCommitCount()
	return nil
}

func (sf *statFilter) DmConnectionRollback(filterChain *filterChain, c *DmConnection) error {
	err := filterChain.DmConnectionRollback(c)
	if err != nil {
		return err
	}
	c.statInfo.getConnStat().incrementRollbackCount()
	return nil
}

func (sf *statFilter) DmConnectionClose(filterChain *filterChain, c *DmConnection) error {
	if !c.closed.IsSet() {
		c.statInfo.getConnStat().decrementStmtByActiveStmtCount(int64(getActiveStmtCount(c)))
		c.statInfo.getConnStat().decrementConn()
	}

	return filterChain.DmConnectionClose(c)
}

func (sf *statFilter) DmConnectionPing(filterChain *filterChain, c *DmConnection, ctx context.Context) error {
	return c.ping(ctx)
}

func (sf *statFilter) DmConnectionExec(filterChain *filterChain, c *DmConnection, query string, args []driver.Value) (*DmResult, error) {
	connExecBefore(c, query)
	dr, err := filterChain.DmConnectionExec(c, query, args)
	if err != nil {
		connExecuteErrorAfter(c, args, err)
		return nil, err
	}
	connExecAfter(c, query, args, int(dr.affectedRows))
	return dr, nil
}

func (sf *statFilter) DmConnectionExecContext(filterChain *filterChain, c *DmConnection, ctx context.Context, query string, args []driver.NamedValue) (*DmResult, error) {
	connExecBefore(c, query)
	dr, err := filterChain.DmConnectionExecContext(c, ctx, query, args)
	if err != nil {
		connExecuteErrorAfter(c, args, err)
		return nil, err
	}
	connExecAfter(c, query, args, int(dr.affectedRows))
	return dr, nil
}

func (sf *statFilter) DmConnectionQuery(filterChain *filterChain, c *DmConnection, query string, args []driver.Value) (*DmRows, error) {
	connQueryBefore(c, query)
	dr, err := filterChain.DmConnectionQuery(c, query, args)
	if err != nil {
		connExecuteErrorAfter(c, args, err)
		return nil, err
	}
	connQueryAfter(c, query, args, dr)
	return dr, nil
}

func (sf *statFilter) DmConnectionQueryContext(filterChain *filterChain, c *DmConnection, ctx context.Context, query string, args []driver.NamedValue) (*DmRows, error) {
	connQueryBefore(c, query)
	dr, err := filterChain.DmConnectionQueryContext(c, ctx, query, args)
	if err != nil {
		connExecuteErrorAfter(c, args, err)
		return nil, err
	}
	connQueryAfter(c, query, args, dr)
	return dr, nil
}

func (sf *statFilter) DmConnectionPrepare(filterChain *filterChain, c *DmConnection, query string) (*DmStatement, error) {
	stmt, err := filterChain.DmConnectionPrepare(c, query)
	if err != nil {
		return nil, err
	}
	statementCreateAfter(c, stmt)
	return stmt, nil
}

func (sf *statFilter) DmConnectionPrepareContext(filterChain *filterChain, c *DmConnection, ctx context.Context, query string) (*DmStatement, error) {
	stmt, err := filterChain.DmConnectionPrepareContext(c, ctx, query)
	if err != nil {
		return nil, err
	}
	statementCreateAfter(c, stmt)
	return stmt, nil
}

func (sf *statFilter) DmConnectionResetSession(filterChain *filterChain, c *DmConnection, ctx context.Context) error {
	return filterChain.DmConnectionResetSession(c, ctx)
}

func (sf *statFilter) DmConnectionCheckNamedValue(filterChain *filterChain, c *DmConnection, nv *driver.NamedValue) error {
	return filterChain.DmConnectionCheckNamedValue(c, nv)
}

//DmStatement
func (sf *statFilter) DmStatementClose(filterChain *filterChain, s *DmStatement) error {
	if !s.closed {
		statementCloseBefore(s)
	}
	return filterChain.DmStatementClose(s)
}

func (sf *statFilter) DmStatementNumInput(filterChain *filterChain, s *DmStatement) int {
	return filterChain.DmStatementNumInput(s)
}

func (sf *statFilter) DmStatementExec(filterChain *filterChain, s *DmStatement, args []driver.Value) (*DmResult, error) {
	stmtExecBefore(s)
	dr, err := filterChain.DmStatementExec(s, args)
	if err != nil {
		statementExecuteErrorAfter(s, args, err)
		return nil, err
	}
	stmtExecAfter(s, args, int(dr.affectedRows))
	return dr, nil
}

func (sf *statFilter) DmStatementExecContext(filterChain *filterChain, s *DmStatement, ctx context.Context, args []driver.NamedValue) (*DmResult, error) {
	stmtExecBefore(s)
	dr, err := filterChain.DmStatementExecContext(s, ctx, args)
	if err != nil {
		statementExecuteErrorAfter(s, args, err)
		return nil, err
	}
	stmtExecAfter(s, args, int(dr.affectedRows))
	return dr, nil
}

func (sf *statFilter) DmStatementQuery(filterChain *filterChain, s *DmStatement, args []driver.Value) (*DmRows, error) {
	stmtQueryBefore(s)
	dr, err := filterChain.DmStatementQuery(s, args)
	if err != nil {
		statementExecuteErrorAfter(s, args, err)
		return nil, err
	}
	stmtQueryAfter(s, args, dr)
	return dr, nil
}

func (sf *statFilter) DmStatementQueryContext(filterChain *filterChain, s *DmStatement, ctx context.Context, args []driver.NamedValue) (*DmRows, error) {
	stmtQueryBefore(s)
	dr, err := filterChain.DmStatementQueryContext(s, ctx, args)
	if err != nil {
		statementExecuteErrorAfter(s, args, err)
		return nil, err
	}
	stmtQueryAfter(s, args, dr)
	return dr, nil
}

func (sf *statFilter) DmStatementCheckNamedValue(filterChain *filterChain, s *DmStatement, nv *driver.NamedValue) error {
	return filterChain.DmStatementCheckNamedValue(s, nv)
}

//DmResult
func (sf *statFilter) DmResultLastInsertId(filterChain *filterChain, r *DmResult) (int64, error) {
	return filterChain.DmResultLastInsertId(r)
}

func (sf *statFilter) DmResultRowsAffected(filterChain *filterChain, r *DmResult) (int64, error) {
	return filterChain.DmResultRowsAffected(r)
}

//DmRows
func (sf *statFilter) DmRowsColumns(filterChain *filterChain, r *DmRows) []string {
	return filterChain.DmRowsColumns(r)
}

func (sf *statFilter) DmRowsClose(filterChain *filterChain, r *DmRows) error {
	if !r.CurrentRows.closed {
		resultSetCloseBefore(r)
	}
	return filterChain.DmRowsClose(r)
}

func (sf *statFilter) DmRowsNext(filterChain *filterChain, r *DmRows, dest []driver.Value) error {
	return filterChain.DmRowsNext(r, dest)
}

func (sf *statFilter) DmRowsHasNextResultSet(filterChain *filterChain, r *DmRows) bool {
	return filterChain.DmRowsHasNextResultSet(r)
}

func (sf *statFilter) DmRowsNextResultSet(filterChain *filterChain, r *DmRows) error {
	return filterChain.DmRowsNextResultSet(r)
}

func (sf *statFilter) DmRowsColumnTypeScanType(filterChain *filterChain, r *DmRows, index int) reflect.Type {
	return filterChain.DmRowsColumnTypeScanType(r, index)
}

func (sf *statFilter) DmRowsColumnTypeDatabaseTypeName(filterChain *filterChain, r *DmRows, index int) string {
	return filterChain.DmRowsColumnTypeDatabaseTypeName(r, index)
}

func (sf *statFilter) DmRowsColumnTypeLength(filterChain *filterChain, r *DmRows, index int) (length int64, ok bool) {
	return filterChain.DmRowsColumnTypeLength(r, index)
}

func (sf *statFilter) DmRowsColumnTypeNullable(filterChain *filterChain, r *DmRows, index int) (nullable, ok bool) {
	return filterChain.DmRowsColumnTypeNullable(r, index)
}

func (sf *statFilter) DmRowsColumnTypePrecisionScale(filterChain *filterChain, r *DmRows, index int) (precision, scale int64, ok bool) {
	return filterChain.DmRowsColumnTypePrecisionScale(r, index)
}

func getActiveStmtCount(conn *DmConnection) int {
	if conn.stmtMap == nil {
		return 0
	} else {
		return len(conn.stmtMap)
	}
}

func statementCreateAfter(conn *DmConnection, stmt *DmStatement) {
	stmt.statInfo.setConstructNano()
	conn.statInfo.getConnStat().incrementStmt()
}

func connExecBefore(conn *DmConnection, sql string) {
	conn.statInfo.setLastExecuteSql(sql)
	conn.statInfo.setFirstResultSet(false)
	conn.statInfo.setLastExecuteType(ExecuteUpdate)
	internalBeforeConnExecute(conn, sql)
}

func connExecAfter(conn *DmConnection, sql string, args interface{}, updateCount int) {
	internalAfterConnExecute(conn, args, updateCount)
}

func connQueryBefore(conn *DmConnection, sql string) {
	conn.statInfo.setLastExecuteSql(sql)
	conn.statInfo.setFirstResultSet(true)
	conn.statInfo.setLastExecuteType(ExecuteQuery)
	internalBeforeConnExecute(conn, sql)
}

func connQueryAfter(conn *DmConnection, sql string, args interface{}, resultSet *DmRows) {
	if resultSet != nil {
		connResultSetCreateAfter(resultSet, conn)
	}
	internalAfterConnExecute(conn, args, 0)
}

func stmtExecBefore(stmt *DmStatement) {
	stmt.statInfo.setLastExecuteSql(stmt.nativeSql)
	stmt.statInfo.setFirstResultSet(false)
	stmt.statInfo.setLastExecuteType(ExecuteUpdate)
	internalBeforeStatementExecute(stmt, stmt.nativeSql)
}

func stmtExecAfter(stmt *DmStatement, args interface{}, updateCount int) {
	internalAfterStatementExecute(stmt, args, updateCount)
}

func stmtQueryBefore(stmt *DmStatement) {
	stmt.statInfo.setLastExecuteSql(stmt.nativeSql)
	stmt.statInfo.setFirstResultSet(true)
	stmt.statInfo.setLastExecuteType(ExecuteQuery)
	internalBeforeStatementExecute(stmt, stmt.nativeSql)
}

func stmtQueryAfter(stmt *DmStatement, args interface{}, resultSet *DmRows) {
	if resultSet != nil {
		stmtResultSetCreateAfter(resultSet, stmt)
	}
	internalAfterStatementExecute(stmt, args, 0)
}

func internalBeforeConnExecute(conn *DmConnection, sql string) {
	connStat := conn.statInfo.getConnStat()
	connStat.incrementExecuteCount()
	conn.statInfo.beforeExecute()

	sqlStat := conn.statInfo.getSqlStat()
	if sqlStat == nil || sqlStat.Removed == 1 || !(sqlStat.Sql == sql) {
		sqlStat = connStat.createSqlStat(sql)
		conn.statInfo.setSqlStat(sqlStat)
	}

	inTransaction := false
	inTransaction = !conn.autoCommit

	if sqlStat != nil {
		sqlStat.ExecuteLastStartTime = time.Now().UnixNano()
		sqlStat.incrementRunningCount()

		if inTransaction {
			sqlStat.incrementInTransactionCount()
		}
	}
}

func internalAfterConnExecute(conn *DmConnection, args interface{}, updateCount int) {
	nowNano := time.Now().UnixNano()
	nanos := nowNano - conn.statInfo.getLastExecuteStartNano()

	conn.statInfo.afterExecute(nanos)

	sqlStat := conn.statInfo.getSqlStat()

	if sqlStat != nil {
		sqlStat.incrementExecuteSuccessCount()

		sqlStat.decrementRunningCount()

		parameters := buildSlowParameters(args)

		sqlStat.addExecuteTimeAndResultHoldTimeHistogramRecord(conn.statInfo.getLastExecuteType(), conn.statInfo.isFirstResultSet(),
			nanos, parameters)

		if !conn.statInfo.isFirstResultSet() &&
			conn.statInfo.getLastExecuteType() == ExecuteUpdate {
			if updateCount < 0 {
				updateCount = 0
			}
			sqlStat.addUpdateCount(int64(updateCount))
		}
	}

}

func internalBeforeStatementExecute(stmt *DmStatement, sql string) {
	connStat := stmt.dmConn.statInfo.getConnStat()
	connStat.incrementExecuteCount()
	stmt.statInfo.beforeExecute()

	sqlStat := stmt.statInfo.getSqlStat()
	if sqlStat == nil || sqlStat.Removed == 1 || !(sqlStat.Sql == sql) {
		sqlStat = connStat.createSqlStat(sql)
		stmt.statInfo.setSqlStat(sqlStat)
	}

	inTransaction := false
	inTransaction = !stmt.dmConn.autoCommit

	if sqlStat != nil {
		sqlStat.ExecuteLastStartTime = time.Now().UnixNano()
		sqlStat.incrementRunningCount()

		if inTransaction {
			sqlStat.incrementInTransactionCount()
		}
	}
}

func internalAfterStatementExecute(stmt *DmStatement, args interface{}, updateCount int) {
	nowNano := time.Now().UnixNano()
	nanos := nowNano - stmt.statInfo.getLastExecuteStartNano()

	stmt.statInfo.afterExecute(nanos)

	sqlStat := stmt.statInfo.getSqlStat()

	if sqlStat != nil {
		sqlStat.incrementExecuteSuccessCount()

		sqlStat.decrementRunningCount()

		parameters := ""
		if stmt.paramCount > 0 {
			parameters = buildStmtSlowParameters(stmt, args)
		}
		sqlStat.addExecuteTimeAndResultHoldTimeHistogramRecord(stmt.statInfo.getLastExecuteType(), stmt.statInfo.isFirstResultSet(),
			nanos, parameters)

		if (!stmt.statInfo.isFirstResultSet()) &&
			stmt.statInfo.getLastExecuteType() == ExecuteUpdate {
			updateCount := stmt.execInfo.updateCount
			if updateCount < 0 {
				updateCount = 0
			}
			sqlStat.addUpdateCount(updateCount)
		}

	}

}

func buildSlowParameters(args interface{}) string {
	switch v := args.(type) {
	case []driver.Value:
		sb := bytes.NewBufferString("")
		for i := 0; i < len(v); i++ {
			if i != 0 {
				sb.WriteString(",")
			} else {
				sb.WriteString("[")
			}

			sb.WriteString(fmt.Sprint(v[i]))
		}

		if len(v) > 0 {
			sb.WriteString("]")
		}
		return sb.String()
	case []driver.NamedValue:
		sb := bytes.NewBufferString("")
		for i := 0; i < len(v); i++ {
			if i != 0 {
				sb.WriteString(",")
			} else {
				sb.WriteString("[")
			}

			sb.WriteString(fmt.Sprint(v[i]))
		}
		if len(v) > 0 {
			sb.WriteString("]")
		}
		return sb.String()
	default:
		return ""
	}
}

func buildStmtSlowParameters(stmt *DmStatement, args interface{}) string {
	switch v := args.(type) {
	case []driver.Value:
		sb := bytes.NewBufferString("")
		for i := 0; i < int(stmt.paramCount); i++ {
			if i != 0 {
				sb.WriteString(",")
			} else {
				sb.WriteString("[")
			}

			sb.WriteString(fmt.Sprint(v[i]))
		}
		if len(v) > 0 {
			sb.WriteString("]")
		}
		return sb.String()
	case []driver.NamedValue:
		sb := bytes.NewBufferString("")
		for i := 0; i < int(stmt.paramCount); i++ {
			if i != 0 {
				sb.WriteString(",")
			} else {
				sb.WriteString("[")
			}

			sb.WriteString(fmt.Sprint(v[i]))
		}
		if len(v) > 0 {
			sb.WriteString("]")
		}
		return sb.String()
	default:
		return ""
	}
}

func connExecuteErrorAfter(conn *DmConnection, args interface{}, err error) {
	nanos := time.Now().UnixNano() - conn.statInfo.getLastExecuteStartNano()
	conn.statInfo.getConnStat().incrementErrorCount()
	conn.statInfo.afterExecute(nanos)

	// SQL
	sqlStat := conn.statInfo.getSqlStat()
	if sqlStat != nil {
		sqlStat.decrementRunningCount()
		sqlStat.error(err)
		parameters := buildSlowParameters(args)
		sqlStat.addExecuteTimeAndResultHoldTimeHistogramRecord(conn.statInfo.getLastExecuteType(), conn.statInfo.isFirstResultSet(),
			nanos, parameters)
	}

}

func statementExecuteErrorAfter(stmt *DmStatement, args interface{}, err error) {
	nanos := time.Now().UnixNano() - stmt.statInfo.getLastExecuteStartNano()
	stmt.dmConn.statInfo.getConnStat().incrementErrorCount()
	stmt.statInfo.afterExecute(nanos)

	// SQL
	sqlStat := stmt.statInfo.getSqlStat()
	if sqlStat != nil {
		sqlStat.decrementRunningCount()
		sqlStat.error(err)
		parameters := ""
		if stmt.paramCount > 0 {
			parameters = buildStmtSlowParameters(stmt, args)
		}
		sqlStat.addExecuteTimeAndResultHoldTimeHistogramRecord(stmt.statInfo.getLastExecuteType(), stmt.statInfo.isFirstResultSet(),
			nanos, parameters)
	}

}

func statementCloseBefore(stmt *DmStatement) {
	stmt.dmConn.statInfo.getConnStat().decrementStmt()
}

func connResultSetCreateAfter(dmdbResultSet *DmRows, conn *DmConnection) {
	dmdbResultSet.statInfo.setSql(conn.statInfo.getLastExecuteSql())
	dmdbResultSet.statInfo.setSqlStat(conn.statInfo.getSqlStat())
	dmdbResultSet.statInfo.setConstructNano()
}

func stmtResultSetCreateAfter(dmdbResultSet *DmRows, stmt *DmStatement) {
	dmdbResultSet.statInfo.setSql(stmt.statInfo.getLastExecuteSql())
	dmdbResultSet.statInfo.setSqlStat(stmt.statInfo.getSqlStat())
	dmdbResultSet.statInfo.setConstructNano()
}

func resultSetCloseBefore(resultSet *DmRows) {
	nanos := time.Now().UnixNano() - resultSet.statInfo.getConstructNano()
	fetchRowCount := getFetchedRows(resultSet)
	sqlStat := resultSet.statInfo.getSqlStat()
	if sqlStat != nil && resultSet.statInfo.getCloseCount() == 0 {
		sqlStat.addFetchRowCount(fetchRowCount)
		stmtExecuteNano := resultSet.statInfo.getLastExecuteTimeNano()
		sqlStat.addResultSetHoldTimeNano2(stmtExecuteNano, nanos)
		if resultSet.statInfo.getReadStringLength() > 0 {
			sqlStat.addStringReadLength(resultSet.statInfo.getReadStringLength())
		}
		if resultSet.statInfo.getReadBytesLength() > 0 {
			sqlStat.addReadBytesLength(resultSet.statInfo.getReadBytesLength())
		}
		if resultSet.statInfo.getOpenInputStreamCount() > 0 {
			sqlStat.addInputStreamOpenCount(int64(resultSet.statInfo.getOpenInputStreamCount()))
		}
		if resultSet.statInfo.getOpenReaderCount() > 0 {
			sqlStat.addReaderOpenCount(int64(resultSet.statInfo.getOpenReaderCount()))
		}
	}

	resultSet.statInfo.incrementCloseCount()
}

func getFetchedRows(rs *DmRows) int64 {
	if rs.CurrentRows.currentPos >= rs.CurrentRows.totalRowCount {
		return rs.CurrentRows.totalRowCount
	} else {
		return rs.CurrentRows.currentPos + 1
	}
}
