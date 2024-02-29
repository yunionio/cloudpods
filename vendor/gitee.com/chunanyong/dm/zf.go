/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package dm

import (
	"context"
	"database/sql/driver"
	"io"
	"reflect"
	"time"

	"gitee.com/chunanyong/dm/util"
)

const SQL_GET_DSC_EP_SITE = "SELECT " +
	"dsc.ep_seqno, " +
	"(CASE mal.MAL_INST_HOST WHEN '' THEN mal.MAL_HOST ELSE mal.MAL_INST_HOST END) as ep_host, " +
	"dcr.EP_PORT, " +
	"dsc.EP_STATUS " +
	"FROM V$DSC_EP_INFO dsc " +
	"LEFT join V$DM_MAL_INI mal " +
	"on dsc.EP_NAME = mal.MAL_INST_NAME " +
	"LEFT join (SELECT grp.GROUP_TYPE GROUP_TYPE, ep.* FROM SYS.\"V$DCR_GROUP\" grp, SYS.\"V$DCR_EP\" ep where grp.GROUP_NAME = ep.GROUP_NAME) dcr " +
	"on dsc.EP_NAME = dcr.EP_NAME and GROUP_TYPE = 'DB' order by  dsc.ep_seqno asc;"

type reconnectFilter struct {
}

// 一定抛错
func (rf *reconnectFilter) autoReconnect(connection *DmConnection, err error) error {
	if dmErr, ok := err.(*DmError); ok {
		if dmErr.ErrCode == ECGO_COMMUNITION_ERROR.ErrCode || dmErr.ErrCode == ECGO_CONNECTION_CLOSED.ErrCode {

			if connection.dmConnector.driverReconnect {
				return rf.reconnect(connection, dmErr.getErrText())
			} else {
				connection.Access.Close()
				connection.closed.Set(true)
				return driver.ErrBadConn
			}
		}
	}
	return err
}

// 一定抛错
func (rf *reconnectFilter) reconnect(connection *DmConnection, reason string) error {
	// 读写分离，重连需要处理备机
	var err error
	if connection.dmConnector.rwSeparate {
		err = RWUtil.reconnect(connection)
	} else {
		err = connection.reconnect()
	}

	if err != nil {
		connection.closed.Set(true)
		return ECGO_CONNECTION_SWITCH_FAILED.addDetailln(reason).throw()
	}

	// 重连成功
	connection.closed.Set(false)
	return ECGO_CONNECTION_SWITCHED.addDetailln(reason).throw()
}

func (rf *reconnectFilter) loadDscEpSites(conn *DmConnection) []*ep {
	stmt, rs, err := conn.driverQuery(SQL_GET_DSC_EP_SITE)
	if err != nil {
		return nil
	}
	defer func() {
		rs.close()
		stmt.close()
	}()
	epList := make([]*ep, 0)
	dest := make([]driver.Value, 4)
	for err = rs.next(dest); err != io.EOF; err = rs.next(dest) {
		ep := newEP(dest[1].(string), dest[2].(int32))
		ep.epSeqno = dest[0].(int32)
		if util.StringUtil.EqualsIgnoreCase(dest[3].(string), "OK") {
			ep.epStatus = EP_STATUS_OK
		} else {
			ep.epStatus = EP_STATUS_ERROR
		}
		epList = append(epList, ep)
	}
	return epList
}

func (rf *reconnectFilter) checkAndRecover(conn *DmConnection) error {
	if conn.dmConnector.doSwitch != DO_SWITCH_WHEN_EP_RECOVER {
		return nil
	}
	// check trx finish
	if !conn.trxFinish {
		return nil
	}
	var curIndex = conn.getIndexOnEPGroup()
	if curIndex == 0 || (time.Now().UnixNano()/1000000-conn.recoverInfo.checkEpRecoverTs) < int64(conn.dmConnector.switchInterval) {
		return nil
	}
	// check db recover
	var dscEps []*ep
	if conn.dmConnector.cluster == CLUSTER_TYPE_DSC {
		dscEps = rf.loadDscEpSites(conn)
	}
	if dscEps == nil || len(dscEps) == 0 {
		return nil
	}
	var recover = false
	for _, okEp := range dscEps {
		if okEp.epStatus != EP_STATUS_OK {
			continue
		}
		for i := int32(0); i < curIndex; i++ {
			ep := conn.dmConnector.group.epList[i]
			if okEp.host == ep.host && okEp.port == ep.port {
				recover = true
				break
			}
		}
		if recover {
			break
		}
	}

	conn.recoverInfo.checkEpRecoverTs = time.Now().UnixNano() / 1000000
	if !recover {
		return nil
	}

	if conn.dmConnector.driverReconnect {
		return conn.reconnect()
	} else {
		conn.Access.Close()
		conn.closed.Set(false)
		return ECGO_CONNECTION_CLOSED.throw()
	}

	//return driver.ErrBadConn
	// do reconnect
	//return conn.reconnect()
}

// DmDriver
func (rf *reconnectFilter) DmDriverOpen(filterChain *filterChain, d *DmDriver, dsn string) (*DmConnection, error) {
	return filterChain.DmDriverOpen(d, dsn)
}

func (rf *reconnectFilter) DmDriverOpenConnector(filterChain *filterChain, d *DmDriver, dsn string) (*DmConnector, error) {
	return filterChain.DmDriverOpenConnector(d, dsn)
}

// DmConnector
func (rf *reconnectFilter) DmConnectorConnect(filterChain *filterChain, c *DmConnector, ctx context.Context) (*DmConnection, error) {
	return filterChain.DmConnectorConnect(c, ctx)
}

func (rf *reconnectFilter) DmConnectorDriver(filterChain *filterChain, c *DmConnector) *DmDriver {
	return filterChain.DmConnectorDriver(c)
}

// DmConnection
func (rf *reconnectFilter) DmConnectionBegin(filterChain *filterChain, c *DmConnection) (*DmConnection, error) {
	dc, err := filterChain.DmConnectionBegin(c)
	if err != nil {
		return nil, rf.autoReconnect(c, err)
	}
	return dc, err
}

func (rf *reconnectFilter) DmConnectionBeginTx(filterChain *filterChain, c *DmConnection, ctx context.Context, opts driver.TxOptions) (*DmConnection, error) {
	dc, err := filterChain.DmConnectionBeginTx(c, ctx, opts)
	if err != nil {
		return nil, rf.autoReconnect(c, err)
	}
	return dc, err
}

func (rf *reconnectFilter) DmConnectionCommit(filterChain *filterChain, c *DmConnection) error {
	if err := filterChain.DmConnectionCommit(c); err != nil {
		return rf.autoReconnect(c, err)
	}
	if err := rf.checkAndRecover(c); err != nil {
		return rf.autoReconnect(c, err)
	}
	return nil
}

func (rf *reconnectFilter) DmConnectionRollback(filterChain *filterChain, c *DmConnection) error {
	err := filterChain.DmConnectionRollback(c)
	if err != nil {
		err = rf.autoReconnect(c, err)
	}

	return err
}

func (rf *reconnectFilter) DmConnectionClose(filterChain *filterChain, c *DmConnection) error {
	err := filterChain.DmConnectionClose(c)
	if err != nil {
		err = rf.autoReconnect(c, err)
	}

	return err
}

func (rf *reconnectFilter) DmConnectionPing(filterChain *filterChain, c *DmConnection, ctx context.Context) error {
	err := filterChain.DmConnectionPing(c, ctx)
	if err != nil {
		err = rf.autoReconnect(c, err)
	}

	return err
}

func (rf *reconnectFilter) DmConnectionExec(filterChain *filterChain, c *DmConnection, query string, args []driver.Value) (*DmResult, error) {
	if err := rf.checkAndRecover(c); err != nil {
		return nil, rf.autoReconnect(c, err)
	}
	dr, err := filterChain.DmConnectionExec(c, query, args)
	if err != nil {
		return nil, rf.autoReconnect(c, err)
	}

	return dr, err
}

func (rf *reconnectFilter) DmConnectionExecContext(filterChain *filterChain, c *DmConnection, ctx context.Context, query string, args []driver.NamedValue) (*DmResult, error) {
	if err := rf.checkAndRecover(c); err != nil {
		return nil, rf.autoReconnect(c, err)
	}
	dr, err := filterChain.DmConnectionExecContext(c, ctx, query, args)
	if err != nil {
		return nil, rf.autoReconnect(c, err)
	}

	return dr, err
}

func (rf *reconnectFilter) DmConnectionQuery(filterChain *filterChain, c *DmConnection, query string, args []driver.Value) (*DmRows, error) {
	if err := rf.checkAndRecover(c); err != nil {
		return nil, rf.autoReconnect(c, err)
	}
	dr, err := filterChain.DmConnectionQuery(c, query, args)
	if err != nil {
		return nil, rf.autoReconnect(c, err)
	}

	return dr, err
}

func (rf *reconnectFilter) DmConnectionQueryContext(filterChain *filterChain, c *DmConnection, ctx context.Context, query string, args []driver.NamedValue) (*DmRows, error) {
	if err := rf.checkAndRecover(c); err != nil {
		return nil, rf.autoReconnect(c, err)
	}
	dr, err := filterChain.DmConnectionQueryContext(c, ctx, query, args)
	if err != nil {
		return nil, rf.autoReconnect(c, err)
	}

	return dr, err
}

func (rf *reconnectFilter) DmConnectionPrepare(filterChain *filterChain, c *DmConnection, query string) (*DmStatement, error) {
	ds, err := filterChain.DmConnectionPrepare(c, query)
	if err != nil {
		return nil, rf.autoReconnect(c, err)
	}

	return ds, err
}

func (rf *reconnectFilter) DmConnectionPrepareContext(filterChain *filterChain, c *DmConnection, ctx context.Context, query string) (*DmStatement, error) {
	ds, err := filterChain.DmConnectionPrepareContext(c, ctx, query)
	if err != nil {
		return nil, rf.autoReconnect(c, err)
	}

	return ds, err
}

func (rf *reconnectFilter) DmConnectionResetSession(filterChain *filterChain, c *DmConnection, ctx context.Context) error {
	err := filterChain.DmConnectionResetSession(c, ctx)
	if err != nil {
		err = rf.autoReconnect(c, err)
	}

	return err
}

func (rf *reconnectFilter) DmConnectionCheckNamedValue(filterChain *filterChain, c *DmConnection, nv *driver.NamedValue) error {
	err := filterChain.DmConnectionCheckNamedValue(c, nv)
	if err != nil {
		err = rf.autoReconnect(c, err)
	}

	return err
}

// DmStatement
func (rf *reconnectFilter) DmStatementClose(filterChain *filterChain, s *DmStatement) error {
	err := filterChain.DmStatementClose(s)
	if err != nil {
		err = rf.autoReconnect(s.dmConn, err)
	}

	return err
}

func (rf *reconnectFilter) DmStatementNumInput(filterChain *filterChain, s *DmStatement) int {
	var ret int
	defer func() {
		err := recover()
		if err != nil {
			rf.autoReconnect(s.dmConn, err.(error))
			ret = 0
		}
	}()
	ret = filterChain.DmStatementNumInput(s)
	return ret
}

func (rf *reconnectFilter) DmStatementExec(filterChain *filterChain, s *DmStatement, args []driver.Value) (*DmResult, error) {
	if err := rf.checkAndRecover(s.dmConn); err != nil {
		return nil, rf.autoReconnect(s.dmConn, err)
	}
	dr, err := filterChain.DmStatementExec(s, args)
	if err != nil {
		return nil, rf.autoReconnect(s.dmConn, err)
	}

	return dr, err
}

func (rf *reconnectFilter) DmStatementExecContext(filterChain *filterChain, s *DmStatement, ctx context.Context, args []driver.NamedValue) (*DmResult, error) {
	if err := rf.checkAndRecover(s.dmConn); err != nil {
		return nil, rf.autoReconnect(s.dmConn, err)
	}
	dr, err := filterChain.DmStatementExecContext(s, ctx, args)
	if err != nil {
		return nil, rf.autoReconnect(s.dmConn, err)
	}

	return dr, err
}

func (rf *reconnectFilter) DmStatementQuery(filterChain *filterChain, s *DmStatement, args []driver.Value) (*DmRows, error) {
	if err := rf.checkAndRecover(s.dmConn); err != nil {
		return nil, rf.autoReconnect(s.dmConn, err)
	}
	dr, err := filterChain.DmStatementQuery(s, args)
	if err != nil {
		return nil, rf.autoReconnect(s.dmConn, err)
	}

	return dr, err
}

func (rf *reconnectFilter) DmStatementQueryContext(filterChain *filterChain, s *DmStatement, ctx context.Context, args []driver.NamedValue) (*DmRows, error) {
	if err := rf.checkAndRecover(s.dmConn); err != nil {
		return nil, rf.autoReconnect(s.dmConn, err)
	}
	dr, err := filterChain.DmStatementQueryContext(s, ctx, args)
	if err != nil {
		return nil, rf.autoReconnect(s.dmConn, err)
	}

	return dr, err
}

func (rf *reconnectFilter) DmStatementCheckNamedValue(filterChain *filterChain, s *DmStatement, nv *driver.NamedValue) error {
	err := filterChain.DmStatementCheckNamedValue(s, nv)
	if err != nil {
		err = rf.autoReconnect(s.dmConn, err)
	}

	return err
}

// DmResult
func (rf *reconnectFilter) DmResultLastInsertId(filterChain *filterChain, r *DmResult) (int64, error) {
	i, err := filterChain.DmResultLastInsertId(r)
	if err != nil {
		err = rf.autoReconnect(r.dmStmt.dmConn, err)
		return 0, err
	}

	return i, err
}

func (rf *reconnectFilter) DmResultRowsAffected(filterChain *filterChain, r *DmResult) (int64, error) {
	i, err := filterChain.DmResultRowsAffected(r)
	if err != nil {
		err = rf.autoReconnect(r.dmStmt.dmConn, err)
		return 0, err
	}

	return i, err
}

// DmRows
func (rf *reconnectFilter) DmRowsColumns(filterChain *filterChain, r *DmRows) []string {
	var ret []string
	defer func() {
		err := recover()
		if err != nil {
			rf.autoReconnect(r.CurrentRows.dmStmt.dmConn, err.(error))
			ret = nil
		}
	}()
	ret = filterChain.DmRowsColumns(r)
	return ret
}

func (rf *reconnectFilter) DmRowsClose(filterChain *filterChain, r *DmRows) error {
	err := filterChain.DmRowsClose(r)
	if err != nil {
		err = rf.autoReconnect(r.CurrentRows.dmStmt.dmConn, err)
	}

	return err
}

func (rf *reconnectFilter) DmRowsNext(filterChain *filterChain, r *DmRows, dest []driver.Value) error {
	err := filterChain.DmRowsNext(r, dest)
	if err != nil {
		err = rf.autoReconnect(r.CurrentRows.dmStmt.dmConn, err)
	}

	return err
}

func (rf *reconnectFilter) DmRowsHasNextResultSet(filterChain *filterChain, r *DmRows) bool {
	var ret bool
	defer func() {
		err := recover()
		if err != nil {
			rf.autoReconnect(r.CurrentRows.dmStmt.dmConn, err.(error))
			ret = false
		}
	}()
	ret = filterChain.DmRowsHasNextResultSet(r)
	return ret
}

func (rf *reconnectFilter) DmRowsNextResultSet(filterChain *filterChain, r *DmRows) error {
	err := filterChain.DmRowsNextResultSet(r)
	if err != nil {
		err = rf.autoReconnect(r.CurrentRows.dmStmt.dmConn, err)
	}

	return err
}

func (rf *reconnectFilter) DmRowsColumnTypeScanType(filterChain *filterChain, r *DmRows, index int) reflect.Type {
	var ret reflect.Type
	defer func() {
		err := recover()
		if err != nil {
			rf.autoReconnect(r.CurrentRows.dmStmt.dmConn, err.(error))
			ret = scanTypeUnknown
		}
	}()
	ret = filterChain.DmRowsColumnTypeScanType(r, index)
	return ret
}

func (rf *reconnectFilter) DmRowsColumnTypeDatabaseTypeName(filterChain *filterChain, r *DmRows, index int) string {
	var ret string
	defer func() {
		err := recover()
		if err != nil {
			rf.autoReconnect(r.CurrentRows.dmStmt.dmConn, err.(error))
			ret = ""
		}
	}()
	ret = filterChain.DmRowsColumnTypeDatabaseTypeName(r, index)
	return ret
}

func (rf *reconnectFilter) DmRowsColumnTypeLength(filterChain *filterChain, r *DmRows, index int) (length int64, ok bool) {
	defer func() {
		err := recover()
		if err != nil {
			rf.autoReconnect(r.CurrentRows.dmStmt.dmConn, err.(error))
			length, ok = 0, false
		}
	}()
	return filterChain.DmRowsColumnTypeLength(r, index)
}

func (rf *reconnectFilter) DmRowsColumnTypeNullable(filterChain *filterChain, r *DmRows, index int) (nullable, ok bool) {
	defer func() {
		err := recover()
		if err != nil {
			rf.autoReconnect(r.CurrentRows.dmStmt.dmConn, err.(error))
			nullable, ok = false, false
		}
	}()
	return filterChain.DmRowsColumnTypeNullable(r, index)
}

func (rf *reconnectFilter) DmRowsColumnTypePrecisionScale(filterChain *filterChain, r *DmRows, index int) (precision, scale int64, ok bool) {
	defer func() {
		err := recover()
		if err != nil {
			rf.autoReconnect(r.CurrentRows.dmStmt.dmConn, err.(error))
			precision, scale, ok = 0, 0, false
		}
	}()
	return filterChain.DmRowsColumnTypePrecisionScale(r, index)
}
