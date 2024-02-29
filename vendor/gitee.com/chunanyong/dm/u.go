/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package dm

import (
	"container/list"
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gitee.com/chunanyong/dm/util"
)

var rp = newRsPool()

type DmStatement struct {
	filterable

	dmConn *DmConnection
	rsMap  map[int16]*innerRows
	inUse  bool

	prepared  bool
	innerUsed bool

	innerExec bool

	id int32

	cursorName string

	readBaseColName bool

	execInfo *execRetInfo

	resultSetType int

	resultSetConcurrency int

	resultSetHoldability int

	nativeSql string

	maxFieldSize int

	maxRows int64

	escapeProcessing bool

	queryTimeout int32

	fetchDirection int

	fetchSize int

	cursorUpdateRow int64

	closeOnCompletion bool

	isBatch bool

	closed bool

	columns []column

	serverParams []parameter

	bindParams []parameter

	paramCount int32

	preExec bool
}

type stmtPoolInfo struct {
	id int32

	cursorName string

	readBaseColName bool
}

type rsPoolKey struct {
	dbGuid        string
	currentSchema string
	sql           string
	paramCount    int
}

func newRsPoolKey(stmt *DmStatement, sql string) rsPoolKey {
	rpk := new(rsPoolKey)
	rpk.dbGuid = stmt.dmConn.Guid
	rpk.currentSchema = stmt.dmConn.Schema
	rpk.paramCount = int(stmt.paramCount)

	rpk.sql = sql
	return *rpk
}

func (key rsPoolKey) equals(destKey rsPoolKey) bool {
	return key.dbGuid == destKey.dbGuid &&
		key.currentSchema == destKey.currentSchema &&
		key.sql == destKey.sql &&
		key.paramCount == destKey.paramCount

}

type rsPoolValue struct {
	m_lastChkTime int
	m_TbIds       []int32
	m_TbTss       []int64
	execInfo      *execRetInfo
}

func newRsPoolValue(execInfo *execRetInfo) rsPoolValue {
	rpv := new(rsPoolValue)
	rpv.execInfo = execInfo
	rpv.m_lastChkTime = time.Now().Nanosecond()
	copy(rpv.m_TbIds, execInfo.tbIds)
	copy(rpv.m_TbTss, execInfo.tbTss)
	return *rpv
}

func (rpv rsPoolValue) refreshed(conn *DmConnection) (bool, error) {

	if conn.dmConnector.rsRefreshFreq == 0 {
		return false, nil
	}

	if rpv.m_lastChkTime+conn.dmConnector.rsRefreshFreq*int(time.Second) > time.Now().Nanosecond() {
		return false, nil
	}

	tss, err := conn.Access.Dm_build_1499(interface{}(rpv.m_TbIds).([]uint32))
	if err != nil {
		return false, err
	}
	rpv.m_lastChkTime = time.Now().Nanosecond()

	var tbCount int
	if tss != nil {
		tbCount = len(tss)
	}

	if tbCount != len(rpv.m_TbTss) {
		return true, nil
	}

	for i := 0; i < tbCount; i++ {
		if rpv.m_TbTss[i] != tss[i] {
			return true, nil
		}

	}
	return false, nil
}

func (rpv rsPoolValue) getResultSet(stmt *DmStatement) *innerRows {
	destDatas := rpv.execInfo.rsDatas
	var totalRows int
	if rpv.execInfo.rsDatas != nil {
		totalRows = len(rpv.execInfo.rsDatas)
	}

	if stmt.maxRows > 0 && stmt.maxRows < int64(totalRows) {
		destDatas = make([][][]byte, stmt.maxRows)
		copy(destDatas[:len(destDatas)], rpv.execInfo.rsDatas[:len(destDatas)])
	}

	rs := newLocalInnerRows(stmt, stmt.columns, destDatas)
	rs.id = 1
	return rs
}

func (rpv rsPoolValue) getDataLen() int {
	return rpv.execInfo.rsSizeof
}

type rsPool struct {
	rsMap        map[rsPoolKey]rsPoolValue
	rsList       *list.List
	totalDataLen int
}

func newRsPool() *rsPool {
	rp := new(rsPool)
	rp.rsMap = make(map[rsPoolKey]rsPoolValue, 100)
	rp.rsList = list.New()
	return rp
}

func (rp *rsPool) removeInList(key rsPoolKey) {
	for e := rp.rsList.Front(); e != nil && e.Value.(rsPoolKey).equals(key); e = e.Next() {
		rp.rsList.Remove(e)
	}
}

func (rp *rsPool) put(stmt *DmStatement, sql string, execInfo *execRetInfo) {
	var dataLen int
	if execInfo != nil {
		dataLen = execInfo.rsSizeof
	}

	cacheSize := stmt.dmConn.dmConnector.rsCacheSize * 1024 * 1024

	for rp.totalDataLen+dataLen > cacheSize {
		if rp.totalDataLen == 0 {
			return
		}

		lk := rp.rsList.Back().Value.(rsPoolKey)
		rp.totalDataLen -= rp.rsMap[lk].getDataLen()
		rp.rsList.Remove(rp.rsList.Back())
		delete(rp.rsMap, rp.rsList.Back().Value.(rsPoolKey))
	}

	key := newRsPoolKey(stmt, sql)
	value := newRsPoolValue(execInfo)

	if _, ok := rp.rsMap[key]; !ok {
		rp.rsList.PushFront(key)
	} else {
		rp.removeInList(key)
		rp.rsList.PushFront(key)
	}

	rp.rsMap[key] = value
	rp.totalDataLen += dataLen
}

func (rp *rsPool) get(stmt *DmStatement, sql string) (*rsPoolValue, error) {
	key := newRsPoolKey(stmt, sql)

	v, ok := rp.rsMap[key]
	if ok {
		b, err := v.refreshed(stmt.dmConn)
		if err != nil {
			return nil, err
		}

		if b {
			rp.removeInList(key)
			delete(rp.rsMap, key)
			return nil, nil
		}

		rp.removeInList(key)
		rp.rsList.PushFront(key)
		return &v, nil
	} else {
		return nil, nil
	}
}

func (s *DmStatement) Close() error {
	if s.closed {
		return nil
	}
	if len(s.filterChain.filters) == 0 {
		return s.close()
	}
	return s.filterChain.reset().DmStatementClose(s)
}

func (s *DmStatement) NumInput() int {
	if err := s.checkClosed(); err != nil {
		return -1
	}
	if len(s.filterChain.filters) == 0 {
		return s.numInput()
	}
	return s.filterChain.reset().DmStatementNumInput(s)
}

func (s *DmStatement) Exec(args []driver.Value) (driver.Result, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}
	if len(s.filterChain.filters) == 0 {
		return s.exec(args)
	}
	return s.filterChain.reset().DmStatementExec(s, args)
}

func (s *DmStatement) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}
	if len(s.filterChain.filters) == 0 {
		return s.execContext(ctx, args)
	}
	return s.filterChain.reset().DmStatementExecContext(s, ctx, args)
}

func (s *DmStatement) Query(args []driver.Value) (driver.Rows, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}
	if len(s.filterChain.filters) == 0 {
		return s.query(args)
	}
	return s.filterChain.reset().DmStatementQuery(s, args)
}

func (s *DmStatement) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}
	if len(s.filterChain.filters) == 0 {
		return s.queryContext(ctx, args)
	}
	return s.filterChain.reset().DmStatementQueryContext(s, ctx, args)
}

func (s *DmStatement) CheckNamedValue(nv *driver.NamedValue) error {
	if len(s.filterChain.filters) == 0 {
		return s.checkNamedValue(nv)
	}
	return s.filterChain.reset().DmStatementCheckNamedValue(s, nv)
}

func (st *DmStatement) prepare() error {
	var err error
	if st.dmConn.dmConnector.escapeProcess {
		st.nativeSql, err = st.dmConn.escape(st.nativeSql, st.dmConn.dmConnector.keyWords)
		if err != nil {
			return err
		}
	}

	st.execInfo, err = st.dmConn.Access.Dm_build_1424(st, Dm_build_95)
	if err != nil {
		return err
	}
	st.prepared = true
	return nil
}

func (stmt *DmStatement) close() error {
	delete(stmt.dmConn.stmtMap, stmt.id)
	if stmt.closed {
		return nil
	}
	stmt.inUse = true

	return stmt.free()

}

func (stmt *DmStatement) numInput() int {
	return int(stmt.paramCount)
}

func (stmt *DmStatement) checkNamedValue(nv *driver.NamedValue) error {
	var err error
	var cvt = converter{stmt.dmConn, false}
	nv.Value, err = cvt.ConvertValue(nv.Value)
	stmt.isBatch = cvt.isBatch
	return err
}

func (stmt *DmStatement) exec(args []driver.Value) (*DmResult, error) {
	var err error

	stmt.inUse = true
	if stmt.isBatch && len(args) > 0 {
		var tmpArg []driver.Value
		var arg driver.Value
		for i := len(args) - 1; i >= 0; i-- {
			if args[i] != nil {
				arg = args[i]
				break
			}
		}
		for _, row := range arg.([][]interface{}) {
			tmpArg = append(tmpArg, row)
		}
		err = stmt.executeBatch(tmpArg)
	} else {
		err = stmt.executeInner(args, Dm_build_97)
	}
	if err != nil {
		return nil, err
	}
	return newDmResult(stmt, stmt.execInfo), nil
}

func (stmt *DmStatement) execContext(ctx context.Context, args []driver.NamedValue) (*DmResult, error) {
	stmt.inUse = true
	dargs, err := namedValueToValue(stmt, args)
	if err != nil {
		return nil, err
	}

	if err := stmt.dmConn.watchCancel(ctx); err != nil {
		return nil, err
	}
	defer stmt.dmConn.finish()

	return stmt.exec(dargs)
}

func (stmt *DmStatement) query(args []driver.Value) (*DmRows, error) {
	var err error
	stmt.inUse = true
	err = stmt.executeInner(args, Dm_build_96)
	if err != nil {
		return nil, err
	}

	if stmt.execInfo.hasResultSet {
		return newDmRows(newInnerRows(0, stmt, stmt.execInfo)), nil
	} else {
		return newDmRows(newLocalInnerRows(stmt, nil, nil)), nil
	}
}

func (stmt *DmStatement) queryContext(ctx context.Context, args []driver.NamedValue) (*DmRows, error) {
	stmt.inUse = true
	dargs, err := namedValueToValue(stmt, args)
	if err != nil {
		return nil, err
	}

	if err := stmt.dmConn.watchCancel(ctx); err != nil {
		return nil, err
	}
	defer stmt.dmConn.finish()

	rows, err := stmt.query(dargs)
	if err != nil {
		stmt.dmConn.finish()
		return nil, err
	}
	rows.finish = stmt.dmConn.finish
	return rows, err
}

func NewDmStmt(conn *DmConnection, sql string) (*DmStatement, error) {
	var s *DmStatement

	if s == nil {
		s = new(DmStatement)
		s.resetFilterable(&conn.filterable)
		s.objId = -1
		s.idGenerator = dmStmtIDGenerator
		s.dmConn = conn
		s.maxRows = int64(conn.dmConnector.maxRows)
		s.nativeSql = sql
		s.rsMap = make(map[int16]*innerRows)
		s.inUse = true
		s.isBatch = conn.isBatch

		err := conn.Access.Dm_build_1406(s)
		if err != nil {
			return nil, err
		}

		conn.stmtMap[s.id] = s
	}

	return s, nil

}

func (stmt *DmStatement) checkClosed() error {
	if stmt.dmConn.closed.IsSet() {
		return driver.ErrBadConn
	} else if stmt.closed {
		return ECGO_STATEMENT_HANDLE_CLOSED.throw()
	}

	return nil
}

func (stmt *DmStatement) free() error {
	delete(stmt.dmConn.stmtMap, stmt.id)
	for _, rs := range stmt.rsMap {
		rs.Close()
	}

	err := stmt.dmConn.Access.Dm_build_1411(int32(stmt.id))
	if err != nil {
		return err
	}
	stmt.inUse = false
	stmt.closed = true
	return nil
}

func bindInParam(stmt *DmStatement, i int, dtype int32, firstRow bool) {
	if !firstRow {
		return
	}
	isNil := dtype == NULL
	serverParam := &stmt.serverParams[i]
	bindParam := &stmt.bindParams[i]
	if serverParam == nil {
		bindParam.resetType(dtype)
	} else {
		bindParam.name = serverParam.name
		bindParam.typeDescriptor = serverParam.typeDescriptor
		bindParam.mask = serverParam.mask
		bindParam.typeFlag = serverParam.typeFlag

		if (serverParam.colType != UNKNOWN && (isNil || serverParam.typeFlag == TYPE_FLAG_EXACT)) || serverParam.mask != 0 {

			bindParam.colType = serverParam.colType
			bindParam.prec = serverParam.prec
			bindParam.scale = serverParam.scale
			bindParam.mask = serverParam.mask
		} else {

			bindParam.resetType(dtype)
		}
	}

	if bindParam.ioType == IO_TYPE_OUT || bindParam.ioType == IO_TYPE_INOUT {
		bindParam.ioType = IO_TYPE_INOUT
	} else {
		bindParam.ioType = IO_TYPE_IN
	}
}

func checkBindParameters(stmt *DmStatement, bytes []interface{}) error {

	for i := 0; int32(i) < stmt.paramCount; i++ {
		if stmt.bindParams[i].ioType == IO_TYPE_UNKNOWN {

			if stmt.serverParams[i].ioType == IO_TYPE_OUT {

				bytes[i] = nil
			} else {
				return ECGO_UNBINDED_PARAMETER.throw()
			}
		}

		if stmt.bindParams[i].colType == CURSOR {
			stmt.bindParams[i].ioType = IO_TYPE_INOUT
			continue
		}

		if stmt.serverParams[i].ioType != stmt.bindParams[i].ioType {

			stmt.bindParams[i].ioType = stmt.serverParams[i].ioType
		}
	}

	for i := 0; int32(i) < stmt.paramCount; i++ {
		if stmt.bindParams[i].ioType == IO_TYPE_INOUT || stmt.bindParams[i].ioType == IO_TYPE_OUT {
			continue
		}
		switch stmt.bindParams[i].colType {
		case CHAR, VARCHAR, VARCHAR2:
			length := -1
			if b, ok := bytes[i].([]byte); ok {
				length = len(b)
			}
			if length > VARCHAR_PREC {
				return ECGO_STRING_CUT.throw()
			}
			if length > int(stmt.bindParams[i].prec) {
				if length < VARCHAR_PREC/4 {
					stmt.bindParams[i].prec = VARCHAR_PREC / 4
				} else if length < VARCHAR_PREC/2 {
					stmt.bindParams[i].prec = VARCHAR_PREC / 2
				} else if length < VARCHAR_PREC*3/4 {
					stmt.bindParams[i].prec = VARCHAR_PREC * 3 / 4
				} else {
					stmt.bindParams[i].prec = VARCHAR_PREC
				}
			}
		}
	}
	return nil
}

func bindOutParam(stmt *DmStatement, i int, dtype int32) error {
	var err error
	serverParam := &stmt.serverParams[i]
	bindParam := &stmt.bindParams[i]

	if bindParam.ioType == IO_TYPE_OUT || bindParam.ioType == IO_TYPE_UNKNOWN {

		if serverParam == nil {

			bindParam.resetType(dtype)
		} else {

			bindParam.name = serverParam.name
			bindParam.typeDescriptor = serverParam.typeDescriptor
			bindParam.mask = serverParam.mask
			bindParam.typeFlag = serverParam.typeFlag

			if (serverParam.colType != UNKNOWN && serverParam.typeFlag == TYPE_FLAG_EXACT) || serverParam.mask != 0 {

				bindParam.colType = serverParam.colType
				bindParam.prec = serverParam.prec
				bindParam.scale = serverParam.scale
				bindParam.mask = serverParam.mask
			} else {

				bindParam.resetType(dtype)
			}
		}

		if bindParam.colType == CURSOR {
			bindParam.ioType = IO_TYPE_INOUT
			if bindParam.cursorStmt == nil {
				bindParam.cursorStmt = &DmStatement{dmConn: stmt.dmConn}
				bindParam.cursorStmt.resetFilterable(&stmt.dmConn.filterable)
				err = bindParam.cursorStmt.dmConn.Access.Dm_build_1406(bindParam.cursorStmt)
			}
		}
	}

	if bindParam.ioType == IO_TYPE_IN || bindParam.ioType == IO_TYPE_INOUT {
		bindParam.ioType = IO_TYPE_INOUT
	} else {
		bindParam.ioType = IO_TYPE_OUT
	}

	return err
}

func encodeArgs(stmt *DmStatement, args []driver.Value, firstRow bool) ([]interface{}, error) {
	bytes := make([]interface{}, len(args), len(args))

	var err error

	for i, arg := range args {
	nextSwitch:
		if stmt.serverParams[i].colType == CURSOR {
			bindInParam(stmt, i, CURSOR, firstRow)
			if stmt.bindParams[i].cursorStmt == nil {
				stmt.bindParams[i].cursorStmt = &DmStatement{dmConn: stmt.dmConn}
				stmt.bindParams[i].cursorStmt.resetFilterable(&stmt.dmConn.filterable)
				err = stmt.bindParams[i].cursorStmt.dmConn.Access.Dm_build_1406(stmt.bindParams[i].cursorStmt)
			}
			stmt.bindParams[i].ioType = IO_TYPE_INOUT
			continue
		}
		if arg == nil {
			bindInParam(stmt, i, NULL, firstRow)
			bytes[i] = nil

			continue
		}

		switch v := arg.(type) {
		case bool:
			bindInParam(stmt, i, TINYINT, firstRow)
			bytes[i], err = G2DB.fromBool(v, stmt.bindParams[i], stmt.dmConn)
		case int8:
			bindInParam(stmt, i, TINYINT, firstRow)
			bytes[i], err = G2DB.fromInt64(int64(v), stmt.bindParams[i], stmt.dmConn)
		case int16:
			bindInParam(stmt, i, SMALLINT, firstRow)
			bytes[i], err = G2DB.fromInt64(int64(v), stmt.bindParams[i], stmt.dmConn)
		case int32:
			bindInParam(stmt, i, INT, firstRow)
			bytes[i], err = G2DB.fromInt64(int64(v), stmt.bindParams[i], stmt.dmConn)
		case int64:
			bindInParam(stmt, i, BIGINT, firstRow)
			bytes[i], err = G2DB.fromInt64(int64(v), stmt.bindParams[i], stmt.dmConn)
		case int:
			bindInParam(stmt, i, BIGINT, firstRow)
			bytes[i], err = G2DB.fromInt64(int64(v), stmt.bindParams[i], stmt.dmConn)
		case uint8:
			bindInParam(stmt, i, SMALLINT, firstRow)
			bytes[i], err = G2DB.fromInt64(int64(v), stmt.bindParams[i], stmt.dmConn)
		case uint16:
			bindInParam(stmt, i, INT, firstRow)
			bytes[i], err = G2DB.fromInt64(int64(v), stmt.bindParams[i], stmt.dmConn)
		case uint32:
			bindInParam(stmt, i, BIGINT, firstRow)
			bytes[i], err = G2DB.fromInt64(int64(v), stmt.bindParams[i], stmt.dmConn)

		case float32:
			bindInParam(stmt, i, REAL, firstRow)
			bytes[i], err = G2DB.fromFloat32(v, stmt.bindParams[i], stmt.dmConn)
		case float64:
			bindInParam(stmt, i, DOUBLE, firstRow)
			bytes[i], err = G2DB.fromFloat64(float64(v), stmt.bindParams[i], stmt.dmConn)
		case []byte:
			if v == nil {
				bindInParam(stmt, i, NULL, firstRow)
				bytes[i] = nil

			} else {
				dtype := VARBINARY
				if len(v) >= VARBINARY_PREC {
					dtype = BLOB
				}
				bindInParam(stmt, i, int32(dtype), firstRow)
				bytes[i], err = G2DB.fromBytes(v, stmt.bindParams[i], stmt.dmConn)
			}
		case string:

			if v == "" && emptyStringToNil(stmt.serverParams[i].colType) {
				arg = nil
				goto nextSwitch
			}
			dtype := VARCHAR
			if len(v) >= VARCHAR_PREC {
				dtype = CLOB
			}
			bindInParam(stmt, i, int32(dtype), firstRow)
			bytes[i], err = G2DB.fromString(v, stmt.bindParams[i], stmt.dmConn)
		case time.Time:
			bindInParam(stmt, i, DATETIME, firstRow)
			bytes[i], err = G2DB.fromTime(v, stmt.bindParams[i], stmt.dmConn)
		case DmTimestamp:
			bindInParam(stmt, i, DATETIME, firstRow)
			bytes[i], err = G2DB.fromTime(v.ToTime(), stmt.bindParams[i], stmt.dmConn)
		case DmIntervalDT:
			bindInParam(stmt, i, INTERVAL_DT, firstRow)
			if stmt.bindParams[i].typeFlag != TYPE_FLAG_EXACT {
				stmt.bindParams[i].scale = int32(v.scaleForSvr)
			}
			bytes[i], err = G2DB.fromDmIntervalDT(v, stmt.bindParams[i], stmt.dmConn)
		case DmIntervalYM:
			bindInParam(stmt, i, INTERVAL_YM, firstRow)
			if stmt.bindParams[i].typeFlag != TYPE_FLAG_EXACT {
				stmt.bindParams[i].scale = int32(v.scaleForSvr)
			}
			bytes[i], err = G2DB.fromDmdbIntervalYM(v, stmt.bindParams[i], stmt.dmConn)
		case DmDecimal:
			bindInParam(stmt, i, DECIMAL, firstRow)
			bytes[i], err = G2DB.fromDecimal(v, stmt.bindParams[i], stmt.dmConn)

		case DmBlob:
			bindInParam(stmt, i, BLOB, firstRow)
			bytes[i], err = G2DB.fromBlob(DmBlob(v), stmt.bindParams[i], stmt.dmConn)
			if err != nil {
				return nil, err
			}
		case DmClob:
			bindInParam(stmt, i, CLOB, firstRow)
			bytes[i], err = G2DB.fromClob(DmClob(v), stmt.bindParams[i], stmt.dmConn)
			if err != nil {
				return nil, err
			}
		case DmArray:
			bindInParam(stmt, i, ARRAY, firstRow)
			da := &v
			da, err = da.create(stmt.dmConn)
			if err != nil {
				return nil, err
			}

			bytes[i], err = G2DB.fromArray(da, stmt.bindParams[i], stmt.dmConn)
		case DmStruct:
			bindInParam(stmt, i, CLASS, firstRow)
			ds := &v
			ds, err = ds.create(stmt.dmConn)
			if err != nil {
				return nil, err
			}

			bytes[i], err = G2DB.fromStruct(ds, stmt.bindParams[i], stmt.dmConn)
		case sql.Out:
			var cvt = converter{stmt.dmConn, false}
			if arg, err = cvt.ConvertValue(v.Dest); err != nil {
				return nil, err
			}
			goto nextSwitch

		case *DmTimestamp:
			bindInParam(stmt, i, DATETIME, firstRow)
			bytes[i], err = G2DB.fromTime(v.ToTime(), stmt.bindParams[i], stmt.dmConn)
		case *DmIntervalDT:
			bindInParam(stmt, i, INTERVAL_DT, firstRow)
			if stmt.bindParams[i].typeFlag != TYPE_FLAG_EXACT {
				stmt.bindParams[i].scale = int32(v.scaleForSvr)
			}
			bytes[i], err = G2DB.fromDmIntervalDT(*v, stmt.bindParams[i], stmt.dmConn)
		case *DmIntervalYM:
			bindInParam(stmt, i, INTERVAL_YM, firstRow)
			if stmt.bindParams[i].typeFlag != TYPE_FLAG_EXACT {
				stmt.bindParams[i].scale = int32(v.scaleForSvr)
			}
			bytes[i], err = G2DB.fromDmdbIntervalYM(*v, stmt.bindParams[i], stmt.dmConn)
		case *DmDecimal:
			bindInParam(stmt, i, DECIMAL, firstRow)
			bytes[i], err = G2DB.fromDecimal(*v, stmt.bindParams[i], stmt.dmConn)
		case *DmBlob:
			bindInParam(stmt, i, BLOB, firstRow)
			bytes[i], err = G2DB.fromBlob(DmBlob(*v), stmt.bindParams[i], stmt.dmConn)
		case *DmClob:
			bindInParam(stmt, i, CLOB, firstRow)
			bytes[i], err = G2DB.fromClob(DmClob(*v), stmt.bindParams[i], stmt.dmConn)
		case *DmArray:
			bindInParam(stmt, i, ARRAY, firstRow)
			v, err = v.create(stmt.dmConn)
			if err != nil {
				return nil, err
			}

			bytes[i], err = G2DB.fromArray(v, stmt.bindParams[i], stmt.dmConn)
		case *DmStruct:
			bindInParam(stmt, i, CLASS, firstRow)
			v, err = v.create(stmt.dmConn)
			if err != nil {
				return nil, err
			}

			bytes[i], err = G2DB.fromStruct(v, stmt.bindParams[i], stmt.dmConn)
		case *driver.Rows:
			if stmt.serverParams[i].colType == CURSOR {
				bindInParam(stmt, i, CURSOR, firstRow)
				if stmt.bindParams[i].cursorStmt == nil {
					stmt.bindParams[i].cursorStmt = &DmStatement{dmConn: stmt.dmConn}
					stmt.bindParams[i].cursorStmt.resetFilterable(&stmt.dmConn.filterable)
					err = stmt.bindParams[i].cursorStmt.dmConn.Access.Dm_build_1406(stmt.bindParams[i].cursorStmt)
				}
			}
		case io.Reader:
			bindInParam(stmt, i, stmt.serverParams[i].colType, firstRow)
			bytes[i], err = G2DB.fromReader(io.Reader(v), stmt.serverParams[i], stmt.dmConn)
			if err != nil {
				return nil, err
			}
		default:
			err = ECGO_UNSUPPORTED_INPARAM_TYPE.throw()
		}

		if err != nil {
			return nil, err
		}

	}
	checkBindParameters(stmt, bytes)

	return bytes, nil
}

type converter struct {
	conn    *DmConnection
	isBatch bool
}
type decimalDecompose interface {
	Decompose(buf []byte) (form byte, negative bool, coefficient []byte, exponent int32)
}

func (c *converter) ConvertValue(v interface{}) (driver.Value, error) {
	if driver.IsValue(v) {
		return v, nil
	}

	switch vr := v.(type) {
	case driver.Valuer:
		sv, err := callValuerValue(vr)
		if err != nil {
			return nil, err
		}

		return sv, nil

	case decimalDecompose, DmDecimal, *DmDecimal, DmTimestamp, *DmTimestamp, DmIntervalDT, *DmIntervalDT,
		DmIntervalYM, *DmIntervalYM, driver.Rows, *driver.Rows, DmArray, *DmArray, DmStruct, *DmStruct, sql.Out:
		return vr, nil
	case big.Int:
		return NewDecimalFromBigInt(&vr)
	case big.Float:
		return NewDecimalFromBigFloat(&vr)
	case DmClob:

		if vr.connection == nil {
			vr.connection = c.conn
		}
		return vr, nil
	case *DmClob:

		if vr.connection == nil {
			vr.connection = c.conn
		}
		return vr, nil
	case DmBlob:

		if vr.connection == nil {
			vr.connection = c.conn
		}
		return vr, nil
	case *DmBlob:

		if vr.connection == nil {
			vr.connection = c.conn
		}
		return vr, nil
	case io.Reader:
		return vr, nil
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr:
		if rv.IsNil() {
			return nil, nil
		} else {
			return c.ConvertValue(rv.Elem().Interface())
		}
	case reflect.Int:
		return rv.Int(), nil
	case reflect.Int8:
		return int8(rv.Int()), nil
	case reflect.Int16:
		return int16(rv.Int()), nil
	case reflect.Int32:
		return int32(rv.Int()), nil
	case reflect.Int64:
		return int64(rv.Int()), nil
	case reflect.Uint8:
		return uint8(rv.Uint()), nil
	case reflect.Uint16:
		return uint16(rv.Uint()), nil
	case reflect.Uint32:
		return uint32(rv.Uint()), nil
	case reflect.Uint64, reflect.Uint:
		u64 := rv.Uint()
		if u64 >= 1<<63 {
			bigInt := &big.Int{}
			bigInt.SetString(strconv.FormatUint(u64, 10), 10)
			return NewDecimalFromBigInt(bigInt)
		}
		return int64(u64), nil
	case reflect.Float32:
		return float32(rv.Float()), nil
	case reflect.Float64:
		return float64(rv.Float()), nil
	case reflect.Bool:
		return rv.Bool(), nil
	case reflect.Slice:
		ek := rv.Type().Elem().Kind()
		if ek == reflect.Uint8 {
			return rv.Bytes(), nil
		} else if ek == reflect.Slice {
			c.isBatch = true
			return v, nil
		}
		return nil, fmt.Errorf("unsupported type %T, a slice of %s", v, ek)
	case reflect.String:
		return rv.String(), nil
	}
	return nil, fmt.Errorf("unsupported type %T, a %s", v, rv.Kind())
}

var valuerReflectType = reflect.TypeOf((*driver.Valuer)(nil)).Elem()

func callValuerValue(vr driver.Valuer) (v driver.Value, err error) {
	if rv := reflect.ValueOf(vr); rv.Kind() == reflect.Ptr &&
		rv.IsNil() &&
		rv.Type().Elem().Implements(valuerReflectType) {
		return nil, nil
	}
	return vr.Value()
}

func namedValueToValue(stmt *DmStatement, named []driver.NamedValue) ([]driver.Value, error) {

	dargs := make([]driver.Value, stmt.paramCount)
	for i, _ := range dargs {
		found := false
		for _, nv := range named {
			if nv.Name != "" && strings.ToUpper(nv.Name) == strings.ToUpper(stmt.serverParams[i].name) {
				dargs[i] = nv.Value
				found = true
				break
			}
		}

		if !found && i < len(named) {
			dargs[i] = named[i].Value
		}

	}
	return dargs, nil
}

func (stmt *DmStatement) executeInner(args []driver.Value, executeType int16) (err error) {

	var bytes []interface{}

	if stmt.paramCount > 0 {
		bytes, err = encodeArgs(stmt, args, true)
		if err != nil {
			return err
		}
	}
	stmt.execInfo, err = stmt.dmConn.Access.Dm_build_1456(stmt, bytes, false)
	if err != nil {
		return err
	}
	if stmt.execInfo.outParamDatas != nil {
		for i, outParamData := range stmt.execInfo.outParamDatas {
			if stmt.bindParams[i].ioType == IO_TYPE_IN || stmt.bindParams[i].ioType == IO_TYPE_UNKNOWN {
				continue
			}

			var v sql.Out
			ok := true
			for ok {
				if v, ok = args[i].(sql.Out); ok {
					args[i] = v.Dest
				}
			}

			if sc, ok := args[i].(sql.Scanner); ok {
				var v interface{}
				if outParamData == nil && stmt.bindParams[i].colType != CURSOR {
					v = nil
					if err = sc.Scan(v); err != nil {
						return err
					}
					continue
				}

				switch stmt.bindParams[i].colType {
				case BOOLEAN:
					v, err = DB2G.toBool(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case BIT:
					if strings.ToLower(stmt.bindParams[i].typeName) == "boolean" {
						v, err = DB2G.toBool(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
					}

					v, err = DB2G.toInt8(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case TINYINT:
					v, err = DB2G.toInt8(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case SMALLINT:
					v, err = DB2G.toInt16(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case INT:
					v, err = DB2G.toInt32(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case BIGINT:
					v, err = DB2G.toInt64(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case REAL:
					v, err = DB2G.toFloat32(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case DOUBLE:
					v, err = DB2G.toFloat64(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case DATE, TIME, DATETIME, TIME_TZ, DATETIME_TZ, DATETIME2, DATETIME2_TZ:
					v, err = DB2G.toTime(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case INTERVAL_DT:
					v = newDmIntervalDTByBytes(outParamData)
				case INTERVAL_YM:
					v = newDmIntervalYMByBytes(outParamData)
				case DECIMAL:
					v, err = DB2G.toDmDecimal(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case BINARY, VARBINARY:
					v = util.StringUtil.BytesToHexString(outParamData, false)
				case BLOB:
					v = DB2G.toDmBlob(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case CHAR, VARCHAR2, VARCHAR:
					v = DB2G.toString(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case CLOB:
					v = DB2G.toDmClob(outParamData, stmt.dmConn, &stmt.bindParams[i].column)
				case ARRAY:
					v, err = TypeDataSV.bytesToArray(outParamData, nil, stmt.bindParams[i].typeDescriptor)
				case CLASS:
					v, err = TypeDataSV.bytesToObj(outParamData, nil, stmt.bindParams[i].typeDescriptor)
				case CURSOR:
					var tmpExecInfo *execRetInfo
					if tmpExecInfo, err = stmt.dmConn.Access.Dm_build_1466(stmt.bindParams[i].cursorStmt, 1); err != nil {
						return err
					}
					if tmpExecInfo.hasResultSet {
						v = newDmRows(newInnerRows(0, stmt.bindParams[i].cursorStmt, tmpExecInfo))
					}
				default:
					err = ECGO_UNSUPPORTED_OUTPARAM_TYPE.throw()
				}
				if err == nil {
					err = sc.Scan(v)
				}
			} else if args[i] == nil {
				if outParamData == nil && stmt.bindParams[i].colType != CURSOR {
					continue
				}

				switch stmt.bindParams[i].colType {
				case BOOLEAN:
					args[i], err = DB2G.toBool(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case BIT:
					if strings.ToLower(stmt.bindParams[i].typeName) == "boolean" {
						args[i], err = DB2G.toBool(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
					}

					args[i], err = DB2G.toInt8(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case TINYINT:
					args[i], err = DB2G.toInt8(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case SMALLINT:
					args[i], err = DB2G.toInt16(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case INT:
					args[i], err = DB2G.toInt32(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case BIGINT:
					args[i], err = DB2G.toInt64(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case REAL:
					args[i], err = DB2G.toFloat32(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case DOUBLE:
					args[i], err = DB2G.toFloat64(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case DATE, TIME, DATETIME, TIME_TZ, DATETIME_TZ, DATETIME2, DATETIME2_TZ:
					args[i], err = DB2G.toTime(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case INTERVAL_DT:
					args[i] = newDmIntervalDTByBytes(outParamData)
				case INTERVAL_YM:
					args[i] = newDmIntervalYMByBytes(outParamData)
				case DECIMAL:
					args[i], err = DB2G.toDmDecimal(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case BINARY, VARBINARY:
					args[i] = util.StringUtil.BytesToHexString(outParamData, false)
				case BLOB:
					args[i] = DB2G.toDmBlob(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case CHAR, VARCHAR2, VARCHAR:
					args[i] = DB2G.toString(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
				case CLOB:
					args[i] = DB2G.toDmClob(outParamData, stmt.dmConn, &stmt.bindParams[i].column)
				default:
					err = ECGO_UNSUPPORTED_OUTPARAM_TYPE.throw()
				}
			} else {
				switch v := args[i].(type) {
				case *string:
					if outParamData == nil {
						*v = ""
					} else {
						*v = DB2G.toString(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
					}
				case *sql.NullString:
					if outParamData == nil {
						v.String = ""
						v.Valid = false
					} else {
						v.String = DB2G.toString(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
						v.Valid = true
					}
				case *[]byte:
					if outParamData == nil {
						*v = nil
					} else {
						var val []byte
						if val, err = DB2G.toBytes(outParamData, &stmt.bindParams[i].column, stmt.dmConn); err != nil {
							return err
						}
						*v = val
					}
				case *bool:
					if outParamData == nil {
						*v = false
					} else {
						var val bool
						if val, err = DB2G.toBool(outParamData, &stmt.bindParams[i].column, stmt.dmConn); err != nil {
							return err
						}
						*v = val
					}
				case *sql.NullBool:
					if outParamData == nil {
						v.Bool = false
						v.Valid = false
					} else {
						var val bool
						if val, err = DB2G.toBool(outParamData, &stmt.bindParams[i].column, stmt.dmConn); err != nil {
							return err
						}
						v.Bool = val
						v.Valid = true
					}
				case *int8:
					if outParamData == nil {
						*v = 0
					} else {
						var val int8
						if val, err = DB2G.toInt8(outParamData, &stmt.bindParams[i].column, stmt.dmConn); err != nil {
							return err
						}
						*v = val
					}
				case *int16:
					if outParamData == nil {
						*v = 0
					} else {
						var val int16
						if val, err = DB2G.toInt16(outParamData, &stmt.bindParams[i].column, stmt.dmConn); err != nil {
							return err
						}
						*v = val
					}
				case *int32:
					if outParamData == nil {
						*v = 0
					} else {
						var val int32
						if val, err = DB2G.toInt32(outParamData, &stmt.bindParams[i].column, stmt.dmConn); err != nil {
							return err
						}
						*v = val
					}
				case *sql.NullInt32:
					if outParamData == nil {
						v.Int32 = 0
						v.Valid = false
					} else {
						var val int32
						if val, err = DB2G.toInt32(outParamData, &stmt.bindParams[i].column, stmt.dmConn); err != nil {
							return err
						}
						v.Int32 = val
						v.Valid = true
					}
				case *int64:
					if outParamData == nil {
						*v = 0
					} else {
						var val int64
						if val, err = DB2G.toInt64(outParamData, &stmt.bindParams[i].column, stmt.dmConn); err != nil {
							return err
						}
						*v = val
					}
				case *sql.NullInt64:
					if outParamData == nil {
						v.Int64 = 0
						v.Valid = false
					} else {
						var val int64
						if val, err = DB2G.toInt64(outParamData, &stmt.bindParams[i].column, stmt.dmConn); err != nil {
							return err
						}
						v.Int64 = val
						v.Valid = true
					}
				case *uint8:
					if outParamData == nil {
						*v = 0
					} else {
						var val uint8
						if val, err = DB2G.toByte(outParamData, &stmt.bindParams[i].column, stmt.dmConn); err != nil {
							return err
						}
						*v = val
					}
				case *uint16:
					if outParamData == nil {
						*v = 0
					} else {
						var val uint16
						if val, err = DB2G.toUInt16(outParamData, &stmt.bindParams[i].column, stmt.dmConn); err != nil {
							return err
						}
						*v = val
					}
				case *uint32:
					if outParamData == nil {
						*v = 0
					} else {
						var val uint32
						if val, err = DB2G.toUInt32(outParamData, &stmt.bindParams[i].column, stmt.dmConn); err != nil {
							return err
						}
						*v = val
					}
				case *uint64:
					if outParamData == nil {
						*v = 0
					} else {
						var val uint64
						if val, err = DB2G.toUInt64(outParamData, &stmt.bindParams[i].column, stmt.dmConn); err != nil {
							return err
						}
						*v = val
					}
				case *int:
					if outParamData == nil {
						*v = 0
					} else {
						var val int
						if val, err = DB2G.toInt(outParamData, &stmt.bindParams[i].column, stmt.dmConn); err != nil {
							return err
						}
						*v = val
					}
				case *uint:
					if outParamData == nil {
						*v = 0
					} else {
						var val uint
						if val, err = DB2G.toUInt(outParamData, &stmt.bindParams[i].column, stmt.dmConn); err != nil {
							return err
						}
						*v = val
					}
				case *float32:
					if outParamData == nil {
						*v = 0.0
					} else {
						var val float32
						if val, err = DB2G.toFloat32(outParamData, &stmt.bindParams[i].column, stmt.dmConn); err != nil {
							return err
						}
						*v = val
					}
				case *float64:
					if outParamData == nil {
						*v = 0.0
					} else {
						var val float64
						if val, err = DB2G.toFloat64(outParamData, &stmt.bindParams[i].column, stmt.dmConn); err != nil {
							return err
						}
						*v = val
					}
				case *sql.NullFloat64:
					if outParamData == nil {
						v.Float64 = 0.0
						v.Valid = false
					} else {
						var val float64
						if val, err = DB2G.toFloat64(outParamData, &stmt.bindParams[i].column, stmt.dmConn); err != nil {
							return err
						}
						v.Float64 = val
						v.Valid = true
					}
				case *time.Time:
					if outParamData == nil {
						*v = time.Time{}
					} else {
						var val time.Time
						if val, err = DB2G.toTime(outParamData, &stmt.bindParams[i].column, stmt.dmConn); err != nil {
							return err
						}
						*v = val
					}
				case *sql.NullTime:
					if outParamData == nil {
						v.Time = time.Time{}
						v.Valid = false
					} else {
						var val time.Time
						if val, err = DB2G.toTime(outParamData, &stmt.bindParams[i].column, stmt.dmConn); err != nil {
							return err
						}
						v.Time = val
						v.Valid = true
					}
				case *DmTimestamp:
					if outParamData == nil {
						*v = DmTimestamp{}
					} else {
						*v = *newDmTimestampFromBytes(outParamData, stmt.bindParams[i].column, stmt.dmConn)
					}
				case *DmIntervalDT:
					if outParamData == nil {
						*v = DmIntervalDT{}
					} else {
						*v = *newDmIntervalDTByBytes(outParamData)
					}
				case *DmIntervalYM:
					if outParamData == nil {
						*v = DmIntervalYM{}
					} else {
						*v = *newDmIntervalYMByBytes(outParamData)
					}
				case *DmDecimal:
					if outParamData == nil {
						*v = DmDecimal{}
					} else {
						var val *DmDecimal
						if val, err = DB2G.toDmDecimal(outParamData, &stmt.bindParams[i].column, stmt.dmConn); err != nil {
							return err
						}
						*v = *val
					}
				case *DmBlob:
					if outParamData == nil {
						*v = DmBlob{}
					} else {
						*v = *DB2G.toDmBlob(outParamData, &stmt.bindParams[i].column, stmt.dmConn)
					}
				case *DmClob:
					if outParamData == nil {
						*v = DmClob{}
					} else {
						*v = *DB2G.toDmClob(outParamData, stmt.dmConn, &stmt.bindParams[i].column)
					}
				case *driver.Rows:
					if stmt.bindParams[i].colType == CURSOR {
						var tmpExecInfo *execRetInfo
						tmpExecInfo, err = stmt.dmConn.Access.Dm_build_1466(stmt.bindParams[i].cursorStmt, 1)
						if err != nil {
							return err
						}

						if tmpExecInfo.hasResultSet {
							*v = newDmRows(newInnerRows(0, stmt.bindParams[i].cursorStmt, tmpExecInfo))
						} else {
							*v = nil
						}
					}
				case *DmArray:
					if outParamData == nil {
						*v = DmArray{}
					} else {
						var val *DmArray
						if val, err = TypeDataSV.bytesToArray(outParamData, nil, stmt.bindParams[i].typeDescriptor); err != nil {
							return err
						}
						*v = *val
					}
				case *DmStruct:
					if outParamData == nil {
						*v = DmStruct{}
					} else {
						var tmp interface{}
						if tmp, err = TypeDataSV.bytesToObj(outParamData, nil, stmt.bindParams[i].typeDescriptor); err != nil {
							return err
						}
						if val, ok := tmp.(*DmStruct); ok {
							*v = *val
						}
					}
				default:
					err = ECGO_UNSUPPORTED_OUTPARAM_TYPE.throw()
				}
			}
			if err != nil {
				return err
			}
		}

	}
	return err
}

func (stmt *DmStatement) executeBatch(args []driver.Value) (err error) {

	var bytes [][]interface{}

	if stmt.execInfo.retSqlType == Dm_build_110 || stmt.execInfo.retSqlType == Dm_build_115 {
		return ECGO_INVALID_SQL_TYPE.throw()
	}

	if stmt.paramCount > 0 && args != nil && len(args) > 0 {

		if len(args) == 1 || stmt.dmConn.dmConnector.batchType == 2 ||
			(stmt.dmConn.dmConnector.batchNotOnCall && stmt.execInfo.retSqlType == Dm_build_111) {
			return stmt.executeBatchByRow(args)
		} else {
			for i, arg := range args {
				var newArg []driver.Value
				for _, a := range arg.([]interface{}) {
					newArg = append(newArg, a)
				}
				tmpBytes, err := encodeArgs(stmt, newArg, i == 0)
				if err != nil {
					return err
				}
				bytes = append(bytes, tmpBytes)
			}
			stmt.execInfo, err = stmt.dmConn.Access.Dm_build_1445(stmt, bytes, stmt.preExec)
		}
	}
	return err
}

func (stmt *DmStatement) executeBatchByRow(args []driver.Value) (err error) {
	count := len(args)
	stmt.execInfo = NewExceInfo()
	stmt.execInfo.updateCounts = make([]int64, count)
	var sqlErrBuilder strings.Builder
	for i := 0; i < count; i++ {
		tmpExecInfo, err := stmt.dmConn.Access.Dm_build_1456(stmt, args[i].([]interface{}), stmt.preExec || i != 0)
		if err == nil {
			stmt.execInfo.union(tmpExecInfo, i, 1)
		} else {
			stmt.execInfo.updateCounts[i] = -1
			if stmt.dmConn.dmConnector.continueBatchOnError {
				sqlErrBuilder.WriteString("row[" + strconv.Itoa(i) + "]:" + err.Error() + util.LINE_SEPARATOR)
			} else {
				return ECGO_BATCH_ERROR.addDetailln(err.Error()).throw()
			}
		}
	}
	if sqlErrBuilder.Len() > 0 {
		return EC_BP_WITH_ERROR.addDetail(sqlErrBuilder.String()).throw()
	}
	return nil
}
