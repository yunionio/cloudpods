/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package dm

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"sync"
	"sync/atomic"

	"gitee.com/chunanyong/dm/parser"

	"gitee.com/chunanyong/dm/util"
	"golang.org/x/text/encoding"
)

type DmConnection struct {
	filterable
	mu sync.Mutex

	dmConnector *DmConnector
	Access      *dm_build_1345
	stmtMap     map[int32]*DmStatement

	lastExecInfo       *execRetInfo
	lexer              *parser.Lexer
	encode             encoding.Encoding
	encodeBuffer       *bytes.Buffer
	transformReaderDst []byte
	transformReaderSrc []byte

	serverEncoding     string
	GlobalServerSeries int
	ServerVersion      string
	Malini2            bool
	Execute2           bool
	LobEmptyCompOrcl   bool
	IsoLevel           int32
	ReadOnly           bool
	NewLobFlag         bool
	sslEncrypt         int
	MaxRowSize         int32
	DDLAutoCommit      bool
	BackslashEscape    bool
	SvrStat            int32
	SvrMode            int32
	ConstParaOpt       bool
	DbTimezone         int16
	LifeTimeRemainder  int16
	InstanceName       string
	Schema             string
	LastLoginIP        string
	LastLoginTime      string
	FailedAttempts     int32
	LoginWarningID     int32
	GraceTimeRemainder int32
	Guid               string
	DbName             string
	StandbyHost        string
	StandbyPort        int32
	StandbyCount       int32
	SessionID          int64
	OracleDateLanguage byte
	FormatDate         string
	FormatTimestamp    string
	FormatTimestampTZ  string
	FormatTime         string
	FormatTimeTZ       string
	Local              bool
	MsgVersion         int32
	TrxStatus          int32
	dscControl         bool
	trxFinish          bool
	autoCommit         bool
	isBatch            bool

	watching bool
	watcher  chan<- context.Context
	closech  chan struct{}
	finished chan<- struct{}
	canceled atomicError
	closed   atomicBool
}

func (conn *DmConnection) setTrxFinish(status int32) {
	switch status & Dm_build_132 {
	case Dm_build_129, Dm_build_130, Dm_build_131:
		conn.trxFinish = true
	default:
		conn.trxFinish = false
	}
}

func (dmConn *DmConnection) init() {

	dmConn.stmtMap = make(map[int32]*DmStatement)
	dmConn.DbTimezone = 0
	dmConn.GlobalServerSeries = 0
	dmConn.MaxRowSize = 0
	dmConn.LobEmptyCompOrcl = false
	dmConn.ReadOnly = false
	dmConn.DDLAutoCommit = false
	dmConn.ConstParaOpt = false
	dmConn.IsoLevel = -1
	dmConn.Malini2 = true
	dmConn.NewLobFlag = true
	dmConn.Execute2 = true
	dmConn.serverEncoding = ENCODING_GB18030
	dmConn.TrxStatus = Dm_build_80
	dmConn.setTrxFinish(dmConn.TrxStatus)
	dmConn.OracleDateLanguage = byte(Locale)
	dmConn.lastExecInfo = NewExceInfo()
	dmConn.MsgVersion = Dm_build_13

	dmConn.idGenerator = dmConnIDGenerator
}

func (dmConn *DmConnection) reset() {
	dmConn.DbTimezone = 0
	dmConn.GlobalServerSeries = 0
	dmConn.MaxRowSize = 0
	dmConn.LobEmptyCompOrcl = false
	dmConn.ReadOnly = false
	dmConn.DDLAutoCommit = false
	dmConn.ConstParaOpt = false
	dmConn.IsoLevel = -1
	dmConn.Malini2 = true
	dmConn.NewLobFlag = true
	dmConn.Execute2 = true
	dmConn.serverEncoding = ENCODING_GB18030
	dmConn.TrxStatus = Dm_build_80
	dmConn.setTrxFinish(dmConn.TrxStatus)
}

func (dc *DmConnection) checkClosed() error {
	if dc.closed.IsSet() {
		return driver.ErrBadConn
	}

	return nil
}

func (dc *DmConnection) executeInner(query string, execType int16) (interface{}, error) {

	stmt, err := NewDmStmt(dc, query)

	if err != nil {
		return nil, err
	}

	if execType == Dm_build_97 {
		defer stmt.close()
	}

	stmt.innerUsed = true
	if stmt.dmConn.dmConnector.escapeProcess {
		stmt.nativeSql, err = stmt.dmConn.escape(stmt.nativeSql, stmt.dmConn.dmConnector.keyWords)
		if err != nil {
			stmt.close()
			return nil, err
		}
	}

	var optParamList []OptParameter

	if stmt.dmConn.ConstParaOpt {
		optParamList = make([]OptParameter, 0)
		stmt.nativeSql, optParamList, err = stmt.dmConn.execOpt(stmt.nativeSql, optParamList, stmt.dmConn.getServerEncoding())
		if err != nil {
			stmt.close()
			optParamList = nil
		}
	}

	if execType == Dm_build_96 && dc.dmConnector.enRsCache {
		rpv, err := rp.get(stmt, query)
		if err != nil {
			return nil, err
		}

		if rpv != nil {
			stmt.execInfo = rpv.execInfo
			dc.lastExecInfo = rpv.execInfo
			return newDmRows(rpv.getResultSet(stmt)), nil
		}
	}

	var info *execRetInfo

	if optParamList != nil && len(optParamList) > 0 {
		info, err = dc.Access.Dm_build_1428(stmt, optParamList)
		if err != nil {
			stmt.nativeSql = query
			info, err = dc.Access.Dm_build_1434(stmt, execType)
		}
	} else {
		info, err = dc.Access.Dm_build_1434(stmt, execType)
	}

	if err != nil {
		stmt.close()
		return nil, err
	}
	dc.lastExecInfo = info

	if execType == Dm_build_96 && info.hasResultSet {
		return newDmRows(newInnerRows(0, stmt, info)), nil
	} else {
		return newDmResult(stmt, info), nil
	}
}

func g2dbIsoLevel(isoLevel int32) int32 {
	switch isoLevel {
	case 1:
		return Dm_build_84
	case 2:
		return Dm_build_85
	case 4:
		return Dm_build_86
	case 6:
		return Dm_build_87
	default:
		return -1
	}
}

func (dc *DmConnection) Begin() (driver.Tx, error) {
	if len(dc.filterChain.filters) == 0 {
		return dc.begin()
	} else {
		return dc.filterChain.reset().DmConnectionBegin(dc)
	}
}

func (dc *DmConnection) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if len(dc.filterChain.filters) == 0 {
		return dc.beginTx(ctx, opts)
	}
	return dc.filterChain.reset().DmConnectionBeginTx(dc, ctx, opts)
}

func (dc *DmConnection) Commit() error {
	if len(dc.filterChain.filters) == 0 {
		return dc.commit()
	} else {
		return dc.filterChain.reset().DmConnectionCommit(dc)
	}
}

func (dc *DmConnection) Rollback() error {
	if len(dc.filterChain.filters) == 0 {
		return dc.rollback()
	} else {
		return dc.filterChain.reset().DmConnectionRollback(dc)
	}
}

func (dc *DmConnection) Close() error {
	if len(dc.filterChain.filters) == 0 {
		return dc.close()
	} else {
		return dc.filterChain.reset().DmConnectionClose(dc)
	}
}

func (dc *DmConnection) Ping(ctx context.Context) error {
	if len(dc.filterChain.filters) == 0 {
		return dc.ping(ctx)
	} else {
		return dc.filterChain.reset().DmConnectionPing(dc, ctx)
	}
}

func (dc *DmConnection) Exec(query string, args []driver.Value) (driver.Result, error) {
	if len(dc.filterChain.filters) == 0 {
		return dc.exec(query, args)
	}
	return dc.filterChain.reset().DmConnectionExec(dc, query, args)
}

func (dc *DmConnection) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if len(dc.filterChain.filters) == 0 {
		return dc.execContext(ctx, query, args)
	}
	return dc.filterChain.reset().DmConnectionExecContext(dc, ctx, query, args)
}

func (dc *DmConnection) Query(query string, args []driver.Value) (driver.Rows, error) {
	if len(dc.filterChain.filters) == 0 {
		return dc.query(query, args)
	}
	return dc.filterChain.reset().DmConnectionQuery(dc, query, args)
}

func (dc *DmConnection) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if len(dc.filterChain.filters) == 0 {
		return dc.queryContext(ctx, query, args)
	}
	return dc.filterChain.reset().DmConnectionQueryContext(dc, ctx, query, args)
}

func (dc *DmConnection) Prepare(query string) (driver.Stmt, error) {
	if len(dc.filterChain.filters) == 0 {
		return dc.prepare(query)
	}
	return dc.filterChain.reset().DmConnectionPrepare(dc, query)
}

func (dc *DmConnection) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if len(dc.filterChain.filters) == 0 {
		return dc.prepareContext(ctx, query)
	}
	return dc.filterChain.reset().DmConnectionPrepareContext(dc, ctx, query)
}

func (dc *DmConnection) ResetSession(ctx context.Context) error {
	if len(dc.filterChain.filters) == 0 {
		return dc.resetSession(ctx)
	}
	if err := dc.filterChain.reset().DmConnectionResetSession(dc, ctx); err != nil {
		return driver.ErrBadConn
	} else {
		return nil
	}
}

func (dc *DmConnection) CheckNamedValue(nv *driver.NamedValue) error {
	if len(dc.filterChain.filters) == 0 {
		return dc.checkNamedValue(nv)
	}
	return dc.filterChain.reset().DmConnectionCheckNamedValue(dc, nv)
}

func (dc *DmConnection) begin() (*DmConnection, error) {
	return dc.beginTx(context.Background(), driver.TxOptions{driver.IsolationLevel(sql.LevelDefault), false})
}

func (dc *DmConnection) beginTx(ctx context.Context, opts driver.TxOptions) (*DmConnection, error) {
	if err := dc.watchCancel(ctx); err != nil {
		return nil, err
	}
	defer dc.finish()

	err := dc.checkClosed()
	if err != nil {
		return nil, err
	}

	dc.autoCommit = false

	if dc.ReadOnly != opts.ReadOnly {
		dc.ReadOnly = opts.ReadOnly
		var readonly = 0
		if opts.ReadOnly {
			readonly = 1
		}
		dc.exec(fmt.Sprintf("SP_SET_SESSION_READONLY(%d)", readonly), nil)
	}

	if dc.IsoLevel != int32(opts.Isolation) {
		switch sql.IsolationLevel(opts.Isolation) {
		case sql.LevelDefault:
			dc.IsoLevel = int32(sql.LevelReadCommitted)
		case sql.LevelReadUncommitted, sql.LevelReadCommitted, sql.LevelSerializable:
			dc.IsoLevel = int32(opts.Isolation)
		case sql.LevelRepeatableRead:
			if dc.CompatibleMysql() {
				dc.IsoLevel = int32(sql.LevelReadCommitted)
			} else {
				return nil, ECGO_INVALID_TRAN_ISOLATION.throw()
			}
		default:
			return nil, ECGO_INVALID_TRAN_ISOLATION.throw()
		}

		err = dc.Access.Dm_build_1488(dc)
		if err != nil {
			return nil, err
		}
	}

	return dc, nil
}

func (dc *DmConnection) commit() error {
	err := dc.checkClosed()
	if err != nil {
		return err
	}

	defer func() {
		dc.autoCommit = dc.dmConnector.autoCommit
		if dc.ReadOnly {
			dc.exec("SP_SET_SESSION_READONLY(0)", nil)
		}
	}()

	if !dc.autoCommit {
		err = dc.Access.Commit()
		if err != nil {
			return err
		}
		dc.trxFinish = true
		return nil
	} else if !dc.dmConnector.alwayseAllowCommit {
		return ECGO_COMMIT_IN_AUTOCOMMIT_MODE.throw()
	}

	return nil
}

func (dc *DmConnection) rollback() error {
	err := dc.checkClosed()
	if err != nil {
		return err
	}

	defer func() {
		dc.autoCommit = dc.dmConnector.autoCommit
		if dc.ReadOnly {
			dc.exec("SP_SET_SESSION_READONLY(0)", nil)
		}
	}()

	if !dc.autoCommit {
		err = dc.Access.Rollback()
		if err != nil {
			return err
		}
		dc.trxFinish = true
		return nil
	} else if !dc.dmConnector.alwayseAllowCommit {
		return ECGO_ROLLBACK_IN_AUTOCOMMIT_MODE.throw()
	}

	return nil
}

func (dc *DmConnection) reconnect() error {
	err := dc.Access.Close()
	if err != nil {
		return err
	}

	for _, stmt := range dc.stmtMap {

		for id, rs := range stmt.rsMap {
			rs.Close()
			delete(stmt.rsMap, id)
		}
	}

	var newConn *DmConnection
	if dc.dmConnector.group != nil {
		if newConn, err = dc.dmConnector.group.connect(dc.dmConnector); err != nil {
			return err
		}
	} else {
		newConn, err = dc.dmConnector.connect(context.Background())
	}

	oldMap := dc.stmtMap
	newConn.mu = dc.mu
	newConn.filterable = dc.filterable
	*dc = *newConn

	for _, stmt := range oldMap {
		if stmt.closed {
			continue
		}
		err = dc.Access.Dm_build_1406(stmt)
		if err != nil {
			stmt.free()
			continue
		}

		if stmt.prepared || stmt.paramCount > 0 {
			if err = stmt.prepare(); err != nil {
				continue
			}
		}

		dc.stmtMap[stmt.id] = stmt
	}

	return nil
}

func (dc *DmConnection) cleanup() {
	dc.close()
}

func (dc *DmConnection) close() error {
	if !dc.closed.TrySet(true) {
		return nil
	}

	util.AbsorbPanic(func() {
		close(dc.closech)
	})
	if dc.Access == nil {
		return nil
	}

	dc.rollback()

	for _, stmt := range dc.stmtMap {
		stmt.free()
	}

	dc.Access.Close()

	return nil
}

func (dc *DmConnection) ping(ctx context.Context) error {
	if err := dc.watchCancel(ctx); err != nil {
		return err
	}
	defer dc.finish()

	rows, err := dc.query("select 1", nil)
	if err != nil {
		return err
	}
	return rows.close()
}

func (dc *DmConnection) exec(query string, args []driver.Value) (*DmResult, error) {
	err := dc.checkClosed()
	if err != nil {
		return nil, err
	}

	if args != nil && len(args) > 0 {
		stmt, err := dc.prepare(query)
		if err != nil {
			return nil, err
		}
		defer stmt.close()
		dc.lastExecInfo = stmt.execInfo

		return stmt.exec(args)
	} else {
		r1, err := dc.executeInner(query, Dm_build_97)
		if err != nil {
			return nil, err
		}

		if r2, ok := r1.(*DmResult); ok {
			return r2, nil
		} else {
			return nil, ECGO_NOT_EXEC_SQL.throw()
		}
	}
}

func (dc *DmConnection) execContext(ctx context.Context, query string, args []driver.NamedValue) (*DmResult, error) {
	if err := dc.watchCancel(ctx); err != nil {
		return nil, err
	}
	defer dc.finish()

	err := dc.checkClosed()
	if err != nil {
		return nil, err
	}

	if args != nil && len(args) > 0 {
		stmt, err := dc.prepare(query)
		if err != nil {
			return nil, err
		}
		defer stmt.close()
		dc.lastExecInfo = stmt.execInfo
		dargs, err := namedValueToValue(stmt, args)
		if err != nil {
			return nil, err
		}
		return stmt.exec(dargs)
	} else {
		r1, err := dc.executeInner(query, Dm_build_97)
		if err != nil {
			return nil, err
		}

		if r2, ok := r1.(*DmResult); ok {
			return r2, nil
		} else {
			return nil, ECGO_NOT_EXEC_SQL.throw()
		}
	}
}

func (dc *DmConnection) query(query string, args []driver.Value) (*DmRows, error) {

	err := dc.checkClosed()
	if err != nil {
		return nil, err
	}

	if args != nil && len(args) > 0 {
		stmt, err := dc.prepare(query)
		if err != nil {
			return nil, err
		}
		dc.lastExecInfo = stmt.execInfo

		stmt.innerUsed = true
		return stmt.query(args)

	} else {
		r1, err := dc.executeInner(query, Dm_build_96)
		if err != nil {
			return nil, err
		}

		if r2, ok := r1.(*DmRows); ok {
			return r2, nil
		} else {
			return nil, ECGO_NOT_QUERY_SQL.throw()
		}
	}
}

func (dc *DmConnection) queryContext(ctx context.Context, query string, args []driver.NamedValue) (*DmRows, error) {
	if err := dc.watchCancel(ctx); err != nil {
		return nil, err
	}
	defer dc.finish()

	err := dc.checkClosed()
	if err != nil {
		return nil, err
	}

	if args != nil && len(args) > 0 {
		stmt, err := dc.prepare(query)
		if err != nil {
			return nil, err
		}
		dc.lastExecInfo = stmt.execInfo

		stmt.innerUsed = true
		dargs, err := namedValueToValue(stmt, args)
		if err != nil {
			return nil, err
		}
		return stmt.query(dargs)

	} else {
		r1, err := dc.executeInner(query, Dm_build_96)
		if err != nil {
			return nil, err
		}

		if r2, ok := r1.(*DmRows); ok {
			return r2, nil
		} else {
			return nil, ECGO_NOT_QUERY_SQL.throw()
		}
	}

}

func (dc *DmConnection) prepare(query string) (stmt *DmStatement, err error) {
	if err = dc.checkClosed(); err != nil {
		return
	}
	if stmt, err = NewDmStmt(dc, query); err != nil {
		return
	}
	if err = stmt.prepare(); err != nil {
		stmt.close()
		stmt = nil
		return
	}
	return
}

func (dc *DmConnection) prepareContext(ctx context.Context, query string) (*DmStatement, error) {
	if err := dc.watchCancel(ctx); err != nil {
		return nil, err
	}
	defer dc.finish()

	return dc.prepare(query)
}

func (dc *DmConnection) resetSession(ctx context.Context) error {
	if err := dc.watchCancel(ctx); err != nil {
		return err
	}
	defer dc.finish()

	err := dc.checkClosed()
	if err != nil {
		return err
	}

	return nil
}

func (dc *DmConnection) checkNamedValue(nv *driver.NamedValue) error {
	var err error
	var cvt = converter{dc, false}
	nv.Value, err = cvt.ConvertValue(nv.Value)
	dc.isBatch = cvt.isBatch
	return err
}

func (dc *DmConnection) driverQuery(query string) (*DmStatement, *DmRows, error) {
	stmt, err := NewDmStmt(dc, query)
	if err != nil {
		return nil, nil, err
	}
	stmt.innerUsed = true
	stmt.innerExec = true
	info, err := dc.Access.Dm_build_1434(stmt, Dm_build_96)
	if err != nil {
		return nil, nil, err
	}
	dc.lastExecInfo = info
	stmt.innerExec = false
	return stmt, newDmRows(newInnerRows(0, stmt, info)), nil
}

func (dc *DmConnection) getIndexOnEPGroup() int32 {
	if dc.dmConnector.group == nil || dc.dmConnector.group.epList == nil {
		return -1
	}
	for i := 0; i < len(dc.dmConnector.group.epList); i++ {
		ep := dc.dmConnector.group.epList[i]
		if dc.dmConnector.host == ep.host && dc.dmConnector.port == ep.port {
			return int32(i)
		}
	}
	return -1
}

func (dc *DmConnection) getServerEncoding() string {
	if dc.dmConnector.charCode != "" {
		return dc.dmConnector.charCode
	}
	return dc.serverEncoding
}

func (dc *DmConnection) lobFetchAll() bool {
	return dc.dmConnector.lobMode == 2
}

func (conn *DmConnection) CompatibleOracle() bool {
	return conn.dmConnector.compatibleMode == COMPATIBLE_MODE_ORACLE
}

func (conn *DmConnection) CompatibleMysql() bool {
	return conn.dmConnector.compatibleMode == COMPATIBLE_MODE_MYSQL
}

func (conn *DmConnection) cancel(err error) {
	conn.canceled.Set(err)
	conn.close()

}

func (conn *DmConnection) finish() {
	if !conn.watching || conn.finished == nil {
		return
	}
	select {
	case conn.finished <- struct{}{}:
		conn.watching = false
	case <-conn.closech:
	}
}

func (conn *DmConnection) startWatcher() {
	watcher := make(chan context.Context, 1)
	conn.watcher = watcher
	finished := make(chan struct{})
	conn.finished = finished
	go func() {
		for {
			var ctx context.Context
			select {
			case ctx = <-watcher:
			case <-conn.closech:
				return
			}

			select {
			case <-ctx.Done():
				conn.cancel(ctx.Err())
			case <-finished:
			case <-conn.closech:
				return
			}
		}
	}()
}

func (conn *DmConnection) watchCancel(ctx context.Context) error {
	if conn.watching {

		conn.cleanup()
		return nil
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if ctx.Done() == nil {
		return nil
	}

	if conn.watcher == nil {
		return nil
	}

	conn.watching = true
	conn.watcher <- ctx
	return nil
}

type noCopy struct{}

func (*noCopy) Lock() {}

type atomicBool struct {
	_noCopy noCopy
	value   uint32
}

func (ab *atomicBool) IsSet() bool {
	return atomic.LoadUint32(&ab.value) > 0
}

func (ab *atomicBool) Set(value bool) {
	if value {
		atomic.StoreUint32(&ab.value, 1)
	} else {
		atomic.StoreUint32(&ab.value, 0)
	}
}

func (ab *atomicBool) TrySet(value bool) bool {
	if value {
		return atomic.SwapUint32(&ab.value, 1) == 0
	}
	return atomic.SwapUint32(&ab.value, 0) > 0
}

type atomicError struct {
	_noCopy noCopy
	value   atomic.Value
}

func (ae *atomicError) Set(value error) {
	ae.value.Store(value)
}

func (ae *atomicError) Value() error {
	if v := ae.value.Load(); v != nil {

		return v.(error)
	}
	return nil
}
