/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package dm

import "sort"

const (
	TYPE_WELL_DISTRIBUTE = 0
	TYPE_HEAD_FIRST      = 1
)

type epSelector struct {
	dbs []*ep
}

func newEPSelector(dbs []*ep) *epSelector {
	return &epSelector{dbs}
}

func (s *epSelector) sortDBList(first bool) []*ep {
	if !first {
		// 按sort从大到小排序，相同sort值顺序不变
		sort.Slice(s.dbs, func(i, j int) bool {
			return s.dbs[i].getSort(first) > s.dbs[j].getSort(first)
		})
	}
	return s.dbs
}

func (s *epSelector) checkServerMode(conn *DmConnection, last bool) (bool, error) {
	// 只连dsc control节点
	if conn.dmConnector.loginDscCtrl && !conn.dscControl {
		conn.close()
		return false, ECGO_INVALID_SERVER_MODE.throw()
	}
	// 模式不匹配, 这里使用的是连接之前的sort，连接之后server的状态可能发生改变sort也可能改变
	if conn.dmConnector.loginStatus > 0 && int(conn.SvrStat) != conn.dmConnector.loginStatus {
		conn.close()
		return false, ECGO_INVALID_SERVER_MODE.throw()
	}
	if last {
		switch conn.dmConnector.loginMode {
		case LOGIN_MODE_PRIMARY_ONLY:
			return conn.SvrMode == SERVER_MODE_PRIMARY, nil
		case LOGIN_MODE_STANDBY_ONLY:
			return conn.SvrMode == SERVER_MODE_STANDBY, nil
		default:
			return true, nil
		}
	}
	switch conn.dmConnector.loginMode {
	case LOGIN_MODE_NORMAL_FIRST:
		return conn.SvrMode == SERVER_MODE_NORMAL, nil
	case LOGIN_MODE_PRIMARY_FIRST, LOGIN_MODE_PRIMARY_ONLY:
		return conn.SvrMode == SERVER_MODE_PRIMARY, nil
	case LOGIN_MODE_STANDBY_FIRST, LOGIN_MODE_STANDBY_ONLY:
		return conn.SvrMode == SERVER_MODE_STANDBY, nil
	default:
		break
	}
	return false, nil
}
