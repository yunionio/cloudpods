/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package dm

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	STATUS_VALID_TIME = 20 * time.Second // ms

	// sort 值
	SORT_SERVER_MODE_INVALID = -1 // 不允许连接的模式

	SORT_SERVER_NOT_ALIVE = -2 // 站点无法连接

	SORT_UNKNOWN = INT32_MAX // 站点还未连接过，模式未知

	SORT_NORMAL = 30

	SORT_PRIMARY = 20

	SORT_STANDBY = 10

	// OPEN>MOUNT>SUSPEND
	SORT_OPEN = 3

	SORT_MOUNT = 2

	SORT_SUSPEND = 1
)

type ep struct {
	host            string
	port            int32
	alive           bool
	statusRefreshTs int64 // 状态更新的时间点
	serverMode      int32
	serverStatus    int32
	dscControl      bool
	sort            int32
	epSeqno         int32
	epStatus        int32
	lock            sync.Mutex
}

func newEP(host string, port int32) *ep {
	ep := new(ep)
	ep.host = host
	ep.port = port
	ep.serverMode = -1
	ep.serverStatus = -1
	ep.sort = SORT_UNKNOWN
	return ep
}

func (ep *ep) getSort(checkTime bool) int32 {
	if checkTime {
		if time.Now().UnixNano()-ep.statusRefreshTs < int64(STATUS_VALID_TIME) {
			return ep.sort
		} else {
			return SORT_UNKNOWN
		}
	}
	return ep.sort
}

func (ep *ep) calcSort(loginMode int32) int32 {
	var sort int32 = 0
	switch loginMode {
	case LOGIN_MODE_PRIMARY_FIRST:
		{
			// 主机优先：PRIMARY>NORMAL>STANDBY
			switch ep.serverMode {
			case SERVER_MODE_NORMAL:
				sort += SORT_NORMAL * 10
			case SERVER_MODE_PRIMARY:
				sort += SORT_PRIMARY * 100
			case SERVER_MODE_STANDBY:
				sort += SORT_STANDBY
			}
		}
	case LOGIN_MODE_STANDBY_FIRST:
		{
			// STANDBY优先: STANDBY>PRIMARY>NORMAL
			switch ep.serverMode {
			case SERVER_MODE_NORMAL:
				sort += SORT_NORMAL
			case SERVER_MODE_PRIMARY:
				sort += SORT_PRIMARY * 10
			case SERVER_MODE_STANDBY:
				sort += SORT_STANDBY * 100
			}
		}
	case LOGIN_MODE_NORMAL_FIRST:
		{
			// NORMAL优先: NORMAL>PRIMARY>STANDBY
			switch ep.serverMode {
			case SERVER_MODE_STANDBY:
				sort += SORT_STANDBY
			case SERVER_MODE_PRIMARY:
				sort += SORT_PRIMARY * 10
			case SERVER_MODE_NORMAL:
				sort += SORT_NORMAL * 100
			}
		}
	case LOGIN_MODE_PRIMARY_ONLY:
		if ep.serverMode != SERVER_MODE_PRIMARY {
			return SORT_SERVER_MODE_INVALID
		}
		sort += SORT_PRIMARY
	case LOGIN_MODE_STANDBY_ONLY:
		if ep.serverMode != SERVER_MODE_STANDBY {
			return SORT_SERVER_MODE_INVALID
		}
		sort += SORT_STANDBY
	}

	switch ep.serverStatus {
	case SERVER_STATUS_MOUNT:
		sort += SORT_MOUNT
	case SERVER_STATUS_OPEN:
		sort += SORT_OPEN
	case SERVER_STATUS_SUSPEND:
		sort += SORT_SUSPEND
	}
	return sort
}

func (ep *ep) refreshStatus(alive bool, conn *DmConnection) {
	ep.lock.Lock()
	defer ep.lock.Unlock()
	ep.alive = alive
	ep.statusRefreshTs = time.Now().UnixNano()
	if alive {
		ep.serverMode = conn.SvrMode
		ep.serverStatus = conn.SvrStat
		ep.dscControl = conn.dscControl
		ep.sort = ep.calcSort(int32(conn.dmConnector.loginMode))
	} else {
		ep.serverMode = -1
		ep.serverStatus = -1
		ep.dscControl = false
		ep.sort = SORT_SERVER_NOT_ALIVE
	}
}

func (ep *ep) connect(connector *DmConnector) (*DmConnection, error) {
	connector.host = ep.host
	connector.port = ep.port
	conn, err := connector.connectSingle(context.Background())
	if err != nil {
		ep.refreshStatus(false, conn)
		return nil, err
	}
	ep.refreshStatus(true, conn)
	return conn, nil
}

func (ep *ep) getServerStatusDesc(serverStatus int32) string {
	ret := ""
	switch ep.serverStatus {
	case SERVER_STATUS_OPEN:
		ret = "OPEN"
	case SERVER_STATUS_MOUNT:
		ret = "MOUNT"
	case SERVER_STATUS_SUSPEND:
		ret = "SUSPEND"
	default:
		ret = "UNKNOWN"
	}
	return ret
}

func (ep *ep) getServerModeDesc(serverMode int32) string {
	ret := ""
	switch ep.serverMode {
	case SERVER_MODE_NORMAL:
		ret = "NORMAL"
	case SERVER_MODE_PRIMARY:
		ret = "PRIMARY"
	case SERVER_MODE_STANDBY:
		ret = "STANDBY"
	default:
		ret = "UNKNOWN"
	}
	return ret
}

func (ep *ep) String() string {
	dscControl := ")"
	if ep.dscControl {
		dscControl = ", DSC CONTROL)"
	}
	return strings.TrimSpace(ep.host) + ":" + strconv.Itoa(int(ep.port)) +
		" (" + ep.getServerModeDesc(ep.serverMode) + ", " + ep.getServerStatusDesc(ep.serverStatus) + dscControl
}
