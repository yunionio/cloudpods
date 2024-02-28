/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package dm

import (
	"database/sql"
	"database/sql/driver"
)

var SQLName sqlName

type sqlName struct {
	m_name string // 描述对象自身名称,

	// 若为内置类型，则表示数据库端定义的名称，与dType相对应
	m_pkgName string // 所在包的名称，适用于包中类型的定义

	m_schName string // 描述对象所在模式名

	m_fulName string // 描述对象完全限定名， 记录用户发送的名称信息；

	// 以及接受服务器响应后，拼成的名称信息

	m_schId int // 保存模式id,模式名无法传出，利用模式id查找

	m_packId int // 保存包的id,包名无法传出，用于查找包名

	m_conn *DmConnection
}

func (SqlName *sqlName) init() {
	SqlName.m_name = ""
	SqlName.m_pkgName = ""
	SqlName.m_schName = ""
	SqlName.m_fulName = ""
	SqlName.m_schId = -1
	SqlName.m_packId = -1
	SqlName.m_conn = nil
}

func newSqlNameByFulName(fulName string) *sqlName {
	o := new(sqlName)
	o.init()
	o.m_fulName = fulName
	return o
}

func newSqlNameByConn(conn *DmConnection) *sqlName {
	o := new(sqlName)
	o.init()
	o.m_conn = conn
	return o
}

func (SqlName *sqlName) getFulName() (string, error) {
	// 说明非内嵌式数据类型名称描述信息传入或已经获取过描述信息
	if len(SqlName.m_fulName) > 0 {
		return SqlName.m_fulName, nil
	}

	// 内嵌式数据类型无名称描述信息返回，直接返回null
	if SqlName.m_name == "" {
		// DBError.throwUnsupportedSQLException();
		return "", nil
	}

	// 其他数据名描述信息
	if SqlName.m_packId != 0 || SqlName.m_schId != 0 {
		query := "SELECT NAME INTO ? FROM SYS.SYSOBJECTS WHERE ID=?"

		params := make([]driver.Value, 2)
		var v string
		params[0] = sql.Out{Dest: &v}
		if SqlName.m_packId != 0 {
			params[1] = SqlName.m_packId
		} else {
			params[1] = SqlName.m_schId
		}

		rs, err := SqlName.m_conn.query(query, params)
		if err != nil {
			return "", err
		}
		rs.close()

		// 说明是包中定义的对象
		if SqlName.m_packId != 0 {
			// pkg全名
			SqlName.m_pkgName = v
			SqlName.m_fulName = SqlName.m_pkgName + "." + SqlName.m_name
		} else {
			// 非包中定义的对象
			// schema 名称
			SqlName.m_schName = v
			SqlName.m_fulName = SqlName.m_schName + "." + SqlName.m_name
		}
	}
	// 将有效值返回
	if len(SqlName.m_fulName) > 0 {
		return SqlName.m_fulName, nil
	} else {
		return SqlName.m_name, nil
	}

}
