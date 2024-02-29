/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package i18n

const Messages_zh_CN = `{
  "language": "zh-Hans",
  "messages": [
    {
      "id": "error.dsn.invalidSchema",
      "translation": "DSN串必须以dm://开头"
    },
    {
      "id": "error.unsupported.scan",
      "translation": "Scan类型转换出错"
    },
    {
      "id": "error.invalidParameterNumber",
      "translation": "参数个数不匹配"
    },
    {
      "id": "error.initThirdPartCipherFailed",
      "translation": "第三方加密初始化失败"
    },
    {
      "id": "error.connectionSwitchFailed",
      "translation": "连接重置失败"
    },
    {
      "id": "error.connectionSwitched",
      "translation": "连接已重置"
    },
    {
      "id": "error.invalidServerMode",
      "translation": "服务器模式不匹配"
    },
    {
      "id": "error.osauthError",
      "translation": "同时使用了指定用户登录和OS认证登录, 请确定一种方式."
    },
    {
      "id": "error.notQuerySQL",
      "translation": "非查询SQL语句"
    },
    {
      "id": "error.notExecSQL",
      "translation": "非执行SQL语句"
    },
    {
      "id": "error.invalidTranIsolation",
      "translation": "非法的事务隔离级"
    },
    {
      "id": "errorCommitInAutoCommitMode",
      "translation": "自动提交模式下不能手动提交"
    },
    {
      "id": "errorRollbackInAutoCommitMode",
      "translation": "自动提交模式下不能手动回滚"
    },
    {
      "id": "errorStatementHandleClosed",
      "translation": "语句已经关闭"
    },
    {
      "id": "errorResultSetColsed",
      "translation": "结果集已经关闭"
    },
    {
      "id": "error.communicationError",
      "translation": "网络通信异常"
    },
    {
      "id": "error.msgCheckError",
      "translation": "消息校验异常"
    },
    {
      "id": "error.unkownNetWork",
      "translation": "未知的网络"
    },
    {
      "id": "error.serverVersion",
      "translation": "服务器版本太低"
    },
    {
      "id": "error.usernameTooLong",
      "translation": "用户名超长"
    },
    {
      "id": "error.passwordTooLong",
      "translation": "密码超长"
    },
    {
      "id": "error.dataTooLong",
      "translation": "数据大小已超过可支持范围"
    },
    {
      "id": "error.invalidColumnType",
      "translation": "无效的列类型"
    },
    {
      "id": "error.dataConvertionError",
      "translation": "类型转换异常"
    },
    {
      "id": "error.invalidConn",
      "translation": "连接失效"
    },
    {
      "id": "error.invalidHex",
      "translation": "无效的十六进制数字"
    },
	{
      "id": "error.invalidBFile",
      "translation": "无效的BFile格式串"
    },
    {
      "id": "error.dataOverflow",
      "translation": "数字溢出"
    },
    {
      "id": "error.invalidDateTimeFormat",
      "translation": "错误的日期时间类型格式"
    },
    {
      "id": "error.datetimeOverflow",
      "translation": "数字溢出"
    },
    {
      "id": "error.invalidTimeInterval",
      "translation": "错误的时间间隔类型数据"
    },
    {
      "id": "error.unsupportedInparamType",
      "translation": "输入参数类型不支持"
    },
    {
      "id": "error.unsupportedOutparamType",
      "translation": "输出参数类型不支持"
    },
    {
      "id": "error.unsupportedType",
      "translation": "不支持该数据类型"
    },
    {
      "id": "error.invalidObjBlob",
      "translation": "无效的对象BLOB数据"
    },
    {
      "id": "error.structMemNotMatch",
      "translation": "记录或类数据成员不匹配"
    },
    {
      "id": "error.invalidComplexTypeName",
      "translation": "无效的类型描述名称"
    },
    {
      "id": "error.invalidParamterValue",
      "translation": "无效的参数值"
    },
    {
      "id": "error.invalidArrayLen",
      "translation": "静态数组长度大于定义时长度"
    },
    {
      "id": "error.invalidSequenceNumber",
      "translation": "无效的列序号"
    },
    {
      "id": "error.resultsetInReadOnlyStatus",
      "translation": "结果集处于只读状态"
    },
    {
      "id": "error.SSLInitFailed",
      "translation": "初始化SSL环境失败"
    },
    {
      "id": "error.LobDataHasFreed",
      "translation": "LOB数据已经被释放"
    },
    {
      "id": "error.fatalError",
      "translation": "致命错误"
    },
    {
      "id": "error.invalidLenOrOffset",
      "translation": "长度或偏移错误"
    },
    {
      "id": "error.intervalValueOverflow",
      "translation": "时间间隔类型数据溢出"
    },
    {
      "id": "error.invalidCipher",
      "translation": "不支持的加密类型"
    },
    {
      "id": "error.storeInNilPointer",
      "translation": "无法将数据存入空指针"
    },
	{
      "id": "error.batchError",
	  "translation": "批量执行出错"
	},
	{
      "id": "warning.bpWithErr",
	  "translation": "警告:批量执行部分行产生错误"
	},
	{
      "id": "error.invalidSqlType",
	  "translation": "非法的SQL语句类型"
	},
	{
      "id": "error.invalidDateTimeValue",
	  "translation": "无效的日期时间类型值"
	},
	{
      "id": "error.msgTooLong",
	  "translation": "消息长度超出限制512M"
	},
	{
      "id": "error.isNull",
	  "translation": "数据为NULL"
	},
	{
      "id": "error.ParamCountLimit",
	  "translation": "参数个数超过最大值65536."
	},
	{
      "id": "error.unbindedParameter",
	  "translation": "有参数未绑定"
	},
	{
      "id": "error.stringCut",
	  "translation": "字符串截断"
	},
	{
      "id": "error.connectionClosedOrNotBuild",
	  "translation": "连接尚未建立或已经关闭"
	}
  ]
}`
