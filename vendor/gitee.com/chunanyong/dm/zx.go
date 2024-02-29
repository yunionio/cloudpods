/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package dm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"regexp"
	"strings"
	"time"

	"gitee.com/chunanyong/dm/util"
)

const (
	SQL_SELECT_STANDBY = "select distinct mailIni.inst_name, mailIni.INST_IP, mailIni.INST_PORT, archIni.arch_status " +
		"from  v$arch_status archIni " +
		"left join (select * from V$DM_MAL_INI) mailIni on archIni.arch_dest = mailIni.inst_name " +
		"left join V$MAL_LINK_STATUS on CTL_LINK_STATUS  = 'CONNECTED' AND DATA_LINK_STATUS = 'CONNECTED' " +
		"where archIni.arch_type in ('TIMELY', 'REALTIME') AND  archIni.arch_status = 'VALID'"

	SQL_SELECT_STANDBY2 = "select distinct " +
		"mailIni.mal_inst_name, mailIni.mal_INST_HOST, mailIni.mal_INST_PORT, archIni.arch_status " +
		"from v$arch_status archIni " + "left join (select * from V$DM_MAL_INI) mailIni " +
		"on archIni.arch_dest = mailIni.mal_inst_name " + "left join V$MAL_LINK_STATUS " +
		"on CTL_LINK_STATUS  = 'CONNECTED' AND DATA_LINK_STATUS = 'CONNECTED' " +
		"where archIni.arch_type in ('TIMELY', 'REALTIME') AND  archIni.arch_status = 'VALID'"
)

type rwUtil struct {
}

var RWUtil = rwUtil{}

func (RWUtil rwUtil) connect(c *DmConnector, ctx context.Context) (*DmConnection, error) {
	c.loginMode = LOGIN_MODE_PRIMARY_ONLY
	connection, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}

	connection.rwInfo.rwCounter = getRwCounterInstance(connection, connection.StandbyCount)
	err = RWUtil.connectStandby(connection)

	return connection, err
}

func (RWUtil rwUtil) reconnect(connection *DmConnection) error {
	if connection.rwInfo == nil {
		return nil
	}

	RWUtil.removeStandby(connection)

	err := connection.reconnect()
	if err != nil {
		return err
	}
	connection.rwInfo.cleanup()
	connection.rwInfo.rwCounter = getRwCounterInstance(connection, connection.StandbyCount)

	err = RWUtil.connectStandby(connection)

	return err
}

func (RWUtil rwUtil) recoverStandby(connection *DmConnection) error {
	if connection.closed.IsSet() || RWUtil.isStandbyAlive(connection) {
		return nil
	}

	ts := time.Now().UnixNano() / 1000000

	freq := int64(connection.dmConnector.rwStandbyRecoverTime)
	if freq <= 0 || ts-connection.rwInfo.tryRecoverTs < freq {
		return nil
	}

	err := RWUtil.connectStandby(connection)
	connection.rwInfo.tryRecoverTs = ts

	return err
}

func (RWUtil rwUtil) connectStandby(connection *DmConnection) error {
	var err error
	db, err := RWUtil.chooseValidStandby(connection)
	if err != nil {
		return err
	}
	if db == nil {
		return nil
	}

	standbyConnectorValue := *connection.dmConnector
	standbyConnector := &standbyConnectorValue
	standbyConnector.host = db.host
	standbyConnector.port = db.port
	standbyConnector.rwStandby = true
	standbyConnector.group = nil
	standbyConnector.loginMode = LOGIN_MODE_STANDBY_ONLY
	standbyConnector.switchTimes = 0
	connection.rwInfo.connStandby, err = standbyConnector.connectSingle(context.Background())
	if err != nil {
		return err
	}

	if connection.rwInfo.connStandby.SvrMode != SERVER_MODE_STANDBY || connection.rwInfo.connStandby.SvrStat != SERVER_STATUS_OPEN {
		RWUtil.removeStandby(connection)
	}
	return nil
}

func (RWUtil rwUtil) chooseValidStandby(connection *DmConnection) (*ep, error) {
	stmt, rs, err := connection.driverQuery(SQL_SELECT_STANDBY2)
	if err != nil {
		stmt, rs, err = connection.driverQuery(SQL_SELECT_STANDBY)
	}
	defer func() {
		if rs != nil {
			rs.close()
		}
		if stmt != nil {
			stmt.close()
		}
	}()
	if err == nil {
		count := int32(rs.CurrentRows.getRowCount())
		if count > 0 {
			connection.rwInfo.rwCounter = getRwCounterInstance(connection, count)
			i := int32(0)
			rowIndex := connection.rwInfo.rwCounter.random(count)
			dest := make([]driver.Value, 3)
			for err := rs.next(dest); err != io.EOF; err = rs.next(dest) {
				if i == rowIndex {
					ep := newEP(dest[1].(string), dest[2].(int32))
					return ep, nil
				}
				i++
			}
		}
	}
	if err != nil {
		return nil, errors.New("choose valid standby error!" + err.Error())
	}
	return nil, nil
}

func (RWUtil rwUtil) afterExceptionOnStandby(connection *DmConnection, e error) {
	if e.(*DmError).ErrCode == ECGO_COMMUNITION_ERROR.ErrCode {
		RWUtil.removeStandby(connection)
	}
}

func (RWUtil rwUtil) removeStandby(connection *DmConnection) {
	if connection.rwInfo.connStandby != nil {
		connection.rwInfo.connStandby.close()
		connection.rwInfo.connStandby = nil
	}
}

func (RWUtil rwUtil) isCreateStandbyStmt(stmt *DmStatement) bool {
	return stmt != nil && stmt.rwInfo.readOnly && RWUtil.isStandbyAlive(stmt.dmConn)
}

func (RWUtil rwUtil) executeByConn(conn *DmConnection, query string, execute1 func() (interface{}, error), execute2 func(otherConn *DmConnection) (interface{}, error)) (interface{}, error) {

	if err := RWUtil.recoverStandby(conn); err != nil {
		return nil, err
	}
	RWUtil.distributeSqlByConn(conn, query)

	turnToPrimary := false

	ret, err := execute1()
	if err != nil {
		if conn.rwInfo.connCurrent == conn.rwInfo.connStandby {

			RWUtil.afterExceptionOnStandby(conn, err)
			turnToPrimary = true
		} else {

			return nil, err
		}
	}

	curConn := conn.rwInfo.connCurrent
	var otherConn *DmConnection
	if curConn != conn {
		otherConn = conn
	} else {
		otherConn = conn.rwInfo.connStandby
	}

	switch curConn.lastExecInfo.retSqlType {
	case Dm_build_101, Dm_build_102, Dm_build_106, Dm_build_113, Dm_build_112, Dm_build_104:
		{

			if otherConn != nil {
				execute2(otherConn)
			}
		}
	case Dm_build_111:
		{

			sqlhead := regexp.MustCompile("[ (]").Split(strings.TrimSpace(query), 2)[0]
			if util.StringUtil.EqualsIgnoreCase(sqlhead, "SP_SET_PARA_VALUE") || util.StringUtil.EqualsIgnoreCase(sqlhead, "SP_SET_SESSION_READONLY") {
				if otherConn != nil {
					execute2(otherConn)
				}
			}
		}
	case Dm_build_110:
		{

			if conn.dmConnector.rwHA && curConn == conn.rwInfo.connStandby &&
				(curConn.lastExecInfo.rsDatas == nil || len(curConn.lastExecInfo.rsDatas) == 0) {
				turnToPrimary = true
			}
		}
	}

	if turnToPrimary {
		conn.rwInfo.toPrimary()
		conn.rwInfo.connCurrent = conn

		return execute2(conn)
	}
	return ret, nil
}

func (RWUtil rwUtil) executeByStmt(stmt *DmStatement, execute1 func() (interface{}, error), execute2 func(otherStmt *DmStatement) (interface{}, error)) (interface{}, error) {
	orgStmt := stmt.rwInfo.stmtCurrent
	query := stmt.nativeSql

	if err := RWUtil.recoverStandby(stmt.dmConn); err != nil {
		return nil, err
	}
	RWUtil.distributeSqlByStmt(stmt)
	if orgStmt != stmt.rwInfo.stmtCurrent {
		RWUtil.copyStatement(orgStmt, stmt.rwInfo.stmtCurrent)
		stmt.rwInfo.stmtCurrent.nativeSql = orgStmt.nativeSql
	}

	turnToPrimary := false

	ret, err := execute1()
	if err != nil {

		if stmt.rwInfo.stmtCurrent == stmt.rwInfo.stmtStandby {
			RWUtil.afterExceptionOnStandby(stmt.dmConn, err)
			turnToPrimary = true
		} else {
			return nil, err
		}
	}

	curStmt := stmt.rwInfo.stmtCurrent
	var otherStmt *DmStatement
	if curStmt != stmt {
		otherStmt = stmt
	} else {
		otherStmt = stmt.rwInfo.stmtStandby
	}

	switch curStmt.execInfo.retSqlType {
	case Dm_build_101, Dm_build_102, Dm_build_106, Dm_build_113, Dm_build_112, Dm_build_104:
		{

			if otherStmt != nil {
				RWUtil.copyStatement(curStmt, otherStmt)
				execute2(otherStmt)
			}
		}
	case Dm_build_111:
		{

			var tmpsql string
			if query != "" {
				tmpsql = strings.TrimSpace(query)
			} else if stmt.nativeSql != "" {
				tmpsql = strings.TrimSpace(stmt.nativeSql)
			} else {
				tmpsql = ""
			}
			sqlhead := regexp.MustCompile("[ (]").Split(tmpsql, 2)[0]
			if util.StringUtil.EqualsIgnoreCase(sqlhead, "SP_SET_PARA_VALUE") || util.StringUtil.EqualsIgnoreCase(sqlhead, "SP_SET_SESSION_READONLY") {
				if otherStmt != nil {
					RWUtil.copyStatement(curStmt, otherStmt)
					execute2(otherStmt)
				}
			}
		}
	case Dm_build_110:
		{

			if stmt.dmConn.dmConnector.rwHA && curStmt == stmt.rwInfo.stmtStandby &&
				(curStmt.execInfo.rsDatas == nil || len(curStmt.execInfo.rsDatas) == 0) {
				turnToPrimary = true
			}
		}
	}

	if turnToPrimary {
		stmt.dmConn.rwInfo.toPrimary()
		stmt.rwInfo.stmtCurrent = stmt

		RWUtil.copyStatement(stmt.rwInfo.stmtStandby, stmt)

		return execute2(stmt)
	}
	return ret, nil
}

func (RWUtil rwUtil) checkReadonlyByConn(conn *DmConnection, sql string) bool {
	readonly := true

	if sql != "" && !conn.dmConnector.rwIgnoreSql {
		tmpsql := strings.TrimSpace(sql)
		sqlhead := strings.SplitN(tmpsql, " ", 2)[0]
		if util.StringUtil.EqualsIgnoreCase(sqlhead, "INSERT") ||
			util.StringUtil.EqualsIgnoreCase(sqlhead, "UPDATE") ||
			util.StringUtil.EqualsIgnoreCase(sqlhead, "DELETE") ||
			util.StringUtil.EqualsIgnoreCase(sqlhead, "CREATE") ||
			util.StringUtil.EqualsIgnoreCase(sqlhead, "TRUNCATE") ||
			util.StringUtil.EqualsIgnoreCase(sqlhead, "DROP") ||
			util.StringUtil.EqualsIgnoreCase(sqlhead, "ALTER") {
			readonly = false
		} else {
			readonly = true
		}
	}
	return readonly
}

func (RWUtil rwUtil) checkReadonlyByStmt(stmt *DmStatement) bool {
	return RWUtil.checkReadonlyByConn(stmt.dmConn, stmt.nativeSql)
}

func (RWUtil rwUtil) distributeSqlByConn(conn *DmConnection, query string) RWSiteEnum {
	var dest RWSiteEnum
	if !RWUtil.isStandbyAlive(conn) {

		dest = conn.rwInfo.toPrimary()
	} else if !RWUtil.checkReadonlyByConn(conn, query) {

		dest = conn.rwInfo.toPrimary()
	} else if (conn.rwInfo.distribute == PRIMARY && !conn.trxFinish) ||
		(conn.rwInfo.distribute == STANDBY && !conn.rwInfo.connStandby.trxFinish) {

		dest = conn.rwInfo.distribute
	} else if conn.IsoLevel != int32(sql.LevelSerializable) {

		dest = conn.rwInfo.toAny()
	} else {
		dest = conn.rwInfo.toPrimary()
	}

	if dest == PRIMARY {
		conn.rwInfo.connCurrent = conn
	} else {
		conn.rwInfo.connCurrent = conn.rwInfo.connStandby
	}
	return dest
}

func (RWUtil rwUtil) distributeSqlByStmt(stmt *DmStatement) RWSiteEnum {
	var dest RWSiteEnum
	if !RWUtil.isStandbyAlive(stmt.dmConn) {

		dest = stmt.dmConn.rwInfo.toPrimary()
	} else if !RWUtil.checkReadonlyByStmt(stmt) {

		dest = stmt.dmConn.rwInfo.toPrimary()
	} else if (stmt.dmConn.rwInfo.distribute == PRIMARY && !stmt.dmConn.trxFinish) ||
		(stmt.dmConn.rwInfo.distribute == STANDBY && !stmt.dmConn.rwInfo.connStandby.trxFinish) {

		dest = stmt.dmConn.rwInfo.distribute
	} else if stmt.dmConn.IsoLevel != int32(sql.LevelSerializable) {

		dest = stmt.dmConn.rwInfo.toAny()
	} else {
		dest = stmt.dmConn.rwInfo.toPrimary()
	}

	if dest == STANDBY && !RWUtil.isStandbyStatementValid(stmt) {

		var err error
		stmt.rwInfo.stmtStandby, err = stmt.dmConn.rwInfo.connStandby.prepare(stmt.nativeSql)
		if err != nil {
			dest = stmt.dmConn.rwInfo.toPrimary()
		}
	}

	if dest == PRIMARY {
		stmt.rwInfo.stmtCurrent = stmt
	} else {
		stmt.rwInfo.stmtCurrent = stmt.rwInfo.stmtStandby
	}
	return dest
}

func (RWUtil rwUtil) isStandbyAlive(connection *DmConnection) bool {
	return connection.rwInfo.connStandby != nil && !connection.rwInfo.connStandby.closed.IsSet()
}

func (RWUtil rwUtil) isStandbyStatementValid(statement *DmStatement) bool {
	return statement.rwInfo.stmtStandby != nil && !statement.rwInfo.stmtStandby.closed
}

func (RWUtil rwUtil) copyStatement(srcStmt *DmStatement, destStmt *DmStatement) {
	destStmt.nativeSql = srcStmt.nativeSql
	destStmt.serverParams = srcStmt.serverParams
	destStmt.bindParams = srcStmt.bindParams
	destStmt.paramCount = srcStmt.paramCount
}
