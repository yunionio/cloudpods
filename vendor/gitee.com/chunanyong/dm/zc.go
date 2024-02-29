/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package dm

import (
	"context"
	"database/sql/driver"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

type filter interface {
	DmDriverOpen(filterChain *filterChain, d *DmDriver, dsn string) (*DmConnection, error)
	DmDriverOpenConnector(filterChain *filterChain, d *DmDriver, dsn string) (*DmConnector, error)

	DmConnectorConnect(filterChain *filterChain, c *DmConnector, ctx context.Context) (*DmConnection, error)
	DmConnectorDriver(filterChain *filterChain, c *DmConnector) *DmDriver

	DmConnectionBegin(filterChain *filterChain, c *DmConnection) (*DmConnection, error)
	DmConnectionBeginTx(filterChain *filterChain, c *DmConnection, ctx context.Context, opts driver.TxOptions) (*DmConnection, error)
	DmConnectionCommit(filterChain *filterChain, c *DmConnection) error
	DmConnectionRollback(filterChain *filterChain, c *DmConnection) error
	DmConnectionClose(filterChain *filterChain, c *DmConnection) error
	DmConnectionPing(filterChain *filterChain, c *DmConnection, ctx context.Context) error
	DmConnectionExec(filterChain *filterChain, c *DmConnection, query string, args []driver.Value) (*DmResult, error)
	DmConnectionExecContext(filterChain *filterChain, c *DmConnection, ctx context.Context, query string, args []driver.NamedValue) (*DmResult, error)
	DmConnectionQuery(filterChain *filterChain, c *DmConnection, query string, args []driver.Value) (*DmRows, error)
	DmConnectionQueryContext(filterChain *filterChain, c *DmConnection, ctx context.Context, query string, args []driver.NamedValue) (*DmRows, error)
	DmConnectionPrepare(filterChain *filterChain, c *DmConnection, query string) (*DmStatement, error)
	DmConnectionPrepareContext(filterChain *filterChain, c *DmConnection, ctx context.Context, query string) (*DmStatement, error)
	DmConnectionResetSession(filterChain *filterChain, c *DmConnection, ctx context.Context) error
	DmConnectionCheckNamedValue(filterChain *filterChain, c *DmConnection, nv *driver.NamedValue) error

	DmStatementClose(filterChain *filterChain, s *DmStatement) error
	DmStatementNumInput(filterChain *filterChain, s *DmStatement) int
	DmStatementExec(filterChain *filterChain, s *DmStatement, args []driver.Value) (*DmResult, error)
	DmStatementExecContext(filterChain *filterChain, s *DmStatement, ctx context.Context, args []driver.NamedValue) (*DmResult, error)
	DmStatementQuery(filterChain *filterChain, s *DmStatement, args []driver.Value) (*DmRows, error)
	DmStatementQueryContext(filterChain *filterChain, s *DmStatement, ctx context.Context, args []driver.NamedValue) (*DmRows, error)
	DmStatementCheckNamedValue(filterChain *filterChain, s *DmStatement, nv *driver.NamedValue) error

	DmResultLastInsertId(filterChain *filterChain, r *DmResult) (int64, error)
	DmResultRowsAffected(filterChain *filterChain, r *DmResult) (int64, error)

	DmRowsColumns(filterChain *filterChain, r *DmRows) []string
	DmRowsClose(filterChain *filterChain, r *DmRows) error
	DmRowsNext(filterChain *filterChain, r *DmRows, dest []driver.Value) error
	DmRowsHasNextResultSet(filterChain *filterChain, r *DmRows) bool
	DmRowsNextResultSet(filterChain *filterChain, r *DmRows) error
	DmRowsColumnTypeScanType(filterChain *filterChain, r *DmRows, index int) reflect.Type
	DmRowsColumnTypeDatabaseTypeName(filterChain *filterChain, r *DmRows, index int) string
	DmRowsColumnTypeLength(filterChain *filterChain, r *DmRows, index int) (length int64, ok bool)
	DmRowsColumnTypeNullable(filterChain *filterChain, r *DmRows, index int) (nullable, ok bool)
	DmRowsColumnTypePrecisionScale(filterChain *filterChain, r *DmRows, index int) (precision, scale int64, ok bool)
}

type IDGenerator int64

var dmDriverIDGenerator = new(IDGenerator)
var dmConntorIDGenerator = new(IDGenerator)
var dmConnIDGenerator = new(IDGenerator)
var dmStmtIDGenerator = new(IDGenerator)
var dmResultIDGenerator = new(IDGenerator)
var dmRowsIDGenerator = new(IDGenerator)

func (g *IDGenerator) incrementAndGet() int64 {
	return atomic.AddInt64((*int64)(g), 1)
}

type RWSiteEnum int

const (
	PRIMARY RWSiteEnum = iota
	STANDBY
	ANYSITE
)

var (
	goMapMu sync.RWMutex
	goMap   = make(map[string]goRun, 2)
)

type filterable struct {
	filterChain *filterChain
	rwInfo      *rwInfo
	logInfo     *logInfo
	recoverInfo *recoverInfo
	statInfo    *statInfo
	objId       int64
	idGenerator *IDGenerator
}

func runLog() {
	goMapMu.Lock()
	_, ok := goMap["log"]
	if !ok {
		goMap["log"] = &logWriter{
			flushQueue: make(chan []byte, LogFlushQueueSize),
			date:       time.Now().Format("2006-01-02"),
			logFile:    nil,
			flushFreq:  LogFlushFreq,
			filePath:   LogDir,
			filePrefix: "dm_go",
			buffer:     Dm_build_935(),
		}
		go goMap["log"].doRun()
	}
	goMapMu.Unlock()
}

func runStat() {
	goMapMu.Lock()
	_, ok := goMap["stat"]
	if !ok {
		goMap["stat"] = newStatFlusher()
		go goMap["stat"].doRun()
	}
	goMapMu.Unlock()
}

func (f *filterable) createFilterChain(bc *DmConnector, props *Properties) {
	var filters = make([]filter, 0, 5)

	if bc != nil {
		if LogLevel != LOG_OFF {
			filters = append(filters, &logFilter{})
			f.logInfo = &logInfo{logRecord: new(LogRecord)}
			runLog()
		}

		if StatEnable {
			filters = append(filters, &statFilter{})
			f.statInfo = newStatInfo()
			goStatMu.Lock()
			if goStat == nil {
				goStat = newGoStat(1000)
			}
			goStatMu.Unlock()
			runStat()
		}

		if bc.doSwitch != DO_SWITCH_OFF {
			filters = append(filters, &reconnectFilter{})
			f.recoverInfo = newRecoverInfo()
		}

		if bc.rwSeparate {
			filters = append(filters, &rwFilter{})
			f.rwInfo = newRwInfo()
		}
	} else if props != nil {
		if ParseLogLevel(props) != LOG_OFF {
			filters = append(filters, &logFilter{})
			f.logInfo = &logInfo{logRecord: new(LogRecord)}
			runLog()
		}

		if props.GetBool("statEnable", StatEnable) {
			filters = append(filters, &statFilter{})
			f.statInfo = newStatInfo()
			goStatMu.Lock()
			if goStat == nil {
				goStat = newGoStat(1000)
			}
			goStatMu.Unlock()
			runStat()
		}

		if props.GetInt(DoSwitchKey, int(DO_SWITCH_OFF), 0, 2) != int(DO_SWITCH_OFF) {
			filters = append(filters, &reconnectFilter{})
			f.recoverInfo = newRecoverInfo()
		}

		if props.GetBool("rwSeparate", false) {
			filters = append(filters, &rwFilter{})
			f.rwInfo = newRwInfo()
		}
	}

	f.filterChain = newFilterChain(filters)
}

func (f *filterable) resetFilterable(src *filterable) {
	f.filterChain = src.filterChain
	f.logInfo = src.logInfo
	f.rwInfo = src.rwInfo
	f.statInfo = src.statInfo
}

func (f *filterable) getID() int64 {
	if f.objId < 0 {
		f.objId = f.idGenerator.incrementAndGet()
	}
	return f.objId
}

type logInfo struct {
	logRecord            *LogRecord
	lastExecuteStartNano time.Time
}

type rwInfo struct {
	distribute RWSiteEnum

	rwCounter *rwCounter

	connStandby *DmConnection

	connCurrent *DmConnection

	tryRecoverTs int64

	stmtStandby *DmStatement

	stmtCurrent *DmStatement

	readOnly bool
}

func newRwInfo() *rwInfo {
	rwInfo := new(rwInfo)
	rwInfo.distribute = PRIMARY
	rwInfo.readOnly = true
	return rwInfo
}

func (rwi *rwInfo) cleanup() {
	rwi.distribute = PRIMARY
	rwi.rwCounter = nil
	rwi.connStandby = nil
	rwi.connCurrent = nil
	rwi.stmtStandby = nil
	rwi.stmtCurrent = nil
}

func (rwi *rwInfo) toPrimary() RWSiteEnum {
	if rwi.distribute != PRIMARY {

		rwi.rwCounter.countPrimary()
	}
	rwi.distribute = PRIMARY
	return rwi.distribute
}

func (rwi *rwInfo) toAny() RWSiteEnum {

	rwi.distribute = rwi.rwCounter.count(ANYSITE, rwi.connStandby)
	return rwi.distribute
}

type recoverInfo struct {
	checkEpRecoverTs int64
}

func newRecoverInfo() *recoverInfo {
	recoverInfo := new(recoverInfo)
	recoverInfo.checkEpRecoverTs = 0
	return recoverInfo
}

type statInfo struct {
	constructNano int64

	connStat *connectionStat

	lastExecuteStartNano int64

	lastExecuteTimeNano int64

	lastExecuteType ExecuteTypeEnum

	firstResultSet bool

	lastExecuteSql string

	sqlStat *sqlStat

	sql string

	cursorIndex int

	closeCount int

	readStringLength int64

	readBytesLength int64

	openInputStreamCount int

	openReaderCount int
}

var (
	goStatMu sync.RWMutex
	goStat   *GoStat
)

func newStatInfo() *statInfo {
	si := new(statInfo)
	return si
}
func (si *statInfo) init(conn *DmConnection) {
	si.connStat = goStat.createConnStat(conn)
}

func (si *statInfo) setConstructNano() {
	si.constructNano = time.Now().UnixNano()
}

func (si *statInfo) getConstructNano() int64 {
	return si.constructNano
}

func (si *statInfo) getConnStat() *connectionStat {
	return si.connStat
}

func (si *statInfo) getLastExecuteStartNano() int64 {
	return si.lastExecuteStartNano
}

func (si *statInfo) setLastExecuteStartNano(lastExecuteStartNano int64) {
	si.lastExecuteStartNano = lastExecuteStartNano
}

func (si *statInfo) getLastExecuteTimeNano() int64 {
	return si.lastExecuteTimeNano
}

func (si *statInfo) setLastExecuteTimeNano(lastExecuteTimeNano int64) {
	si.lastExecuteTimeNano = lastExecuteTimeNano
}

func (si *statInfo) getLastExecuteType() ExecuteTypeEnum {
	return si.lastExecuteType
}

func (si *statInfo) setLastExecuteType(lastExecuteType ExecuteTypeEnum) {
	si.lastExecuteType = lastExecuteType
}

func (si *statInfo) isFirstResultSet() bool {
	return si.firstResultSet
}

func (si *statInfo) setFirstResultSet(firstResultSet bool) {
	si.firstResultSet = firstResultSet
}

func (si *statInfo) getLastExecuteSql() string {
	return si.lastExecuteSql
}

func (si *statInfo) setLastExecuteSql(lastExecuteSql string) {
	si.lastExecuteSql = lastExecuteSql
}

func (si *statInfo) getSqlStat() *sqlStat {
	return si.sqlStat
}

func (si *statInfo) setSqlStat(sqlStat *sqlStat) {
	si.sqlStat = sqlStat
}

func (si *statInfo) setConnStat(connStat *connectionStat) {
	si.connStat = connStat
}

func (si *statInfo) setConstructNanoWithConstructNano(constructNano int64) {
	si.constructNano = constructNano
}

func (si *statInfo) afterExecute(nanoSpan int64) {
	si.lastExecuteTimeNano = nanoSpan
}

func (si *statInfo) beforeExecute() {
	si.lastExecuteStartNano = time.Now().UnixNano()
}

func (si *statInfo) getSql() string {
	return si.sql
}

func (si *statInfo) setSql(sql string) {
	si.sql = sql
}

func (si *statInfo) getCursorIndex() int {
	return si.cursorIndex
}

func (si *statInfo) setCursorIndex(cursorIndex int) {
	si.cursorIndex = cursorIndex
}

func (si *statInfo) getCloseCount() int {
	return si.closeCount
}

func (si *statInfo) setCloseCount(closeCount int) {
	si.closeCount = closeCount
}

func (si *statInfo) getReadStringLength() int64 {
	return si.readStringLength
}

func (si *statInfo) setReadStringLength(readStringLength int64) {
	si.readStringLength = readStringLength
}

func (si *statInfo) getReadBytesLength() int64 {
	return si.readBytesLength
}

func (si *statInfo) setReadBytesLength(readBytesLength int64) {
	si.readBytesLength = readBytesLength
}

func (si *statInfo) getOpenInputStreamCount() int {
	return si.openInputStreamCount
}

func (si *statInfo) setOpenInputStreamCount(openInputStreamCount int) {
	si.openInputStreamCount = openInputStreamCount
}

func (si *statInfo) getOpenReaderCount() int {
	return si.openReaderCount
}

func (si *statInfo) setOpenReaderCount(openReaderCount int) {
	si.openReaderCount = openReaderCount
}

func (si *statInfo) incrementCloseCount() {
	si.closeCount++
}
