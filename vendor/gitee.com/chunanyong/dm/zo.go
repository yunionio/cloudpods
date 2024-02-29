/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package dm

const (
	ParamDataEnum_Null = 0
	/**
	 * 只有大字段才有行内数据、行外数据的概念
	 */
	ParamDataEnum_OFF_ROW = 1
)

// JDBC中的Data
type lobCtl struct {
	value []byte
}

// lob数据返回信息，自bug610335后，服务器不光返回字节数组，还返回字符数
type lobRetInfo struct {
	charLen int64  // 字符长度
	data    []byte //lob数据
}
