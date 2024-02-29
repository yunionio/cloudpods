/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package i18n

const Messages_zh_HK = `{
  "language": "zh-Hant",
  "messages": [
    {
      "id": "error.dsn.invalidSchema",
      "translation": "DSN串必須以dm://開頭"
    },
    {
      "id": "error.unsupported.scan",
      "translation": "Scan類型轉換出錯"
    },
    {
      "id": "error.invalidParameterNumber",
      "translation": "參數個數不匹配"
    },
    {
      "id": "error.initThirdPartCipherFailed",
      "translation": "第三方加密初始化失敗"
    },
    {
      "id": "error.connectionSwitchFailed",
      "translation": "連接重置失敗"
    },
    {
      "id": "error.connectionSwitched",
      "translation": "連接已重置"
    },
    {
      "id": "error.invalidServerMode",
      "translation": "服務器模式不匹配"
    },
    {
      "id": "error.osauthError",
      "translation": "同時使用了指定用戶登錄和OS認證登錄, 請確定一種方式."
    },
    {
      "id": "error.notQuerySQL",
      "translation": "非查詢SQL語句"
    },
    {
      "id": "error.notExecSQL",
      "translation": "非執行SQL語句"
    },
    {
      "id": "error.invalidTranIsolation",
      "translation": "非法的事務隔離級"
    },
    {
      "id": "errorCommitInAutoCommitMode",
      "translation": "自動提交模式下不能手動提交"
    },
    {
      "id": "errorRollbackInAutoCommitMode",
      "translation": "自動提交模式下不能手動回滾"
    },
    {
      "id": "errorStatementHandleClosed",
      "translation": "語句已經關閉"
    },
    {
      "id": "errorResultSetColsed",
      "translation": "結果集已經關閉"
    },
    {
      "id": "error.communicationError",
      "translation": "網絡通信異常"
    },
    {
      "id": "error.msgCheckError",
      "translation": "消息校驗異常"
    },
    {
      "id": "error.unkownNetWork",
      "translation": "未知的網絡"
    },
    {
      "id": "error.serverVersion",
      "translation": "服務器版本太低"
    },
    {
      "id": "error.usernameTooLong",
      "translation": "用戶名超長"
    },
    {
      "id": "error.passwordTooLong",
      "translation": "密碼超長"
    },
    {
      "id": "error.dataTooLong",
      "translation": "數據大小已超過可支持範圍"
    },
    {
      "id": "error.invalidColumnType",
      "translation": "無效的列類型"
    },
    {
      "id": "error.dataConvertionError",
      "translation": "類型轉換異常"
    },
    {
      "id": "error.invalidConn",
      "translation": "連接失效"
    },
    {
      "id": "error.invalidHex",
      "translation": "無效的十六進制数字"
    },
	{
      "id": "error.invalidBFile",
      "translation": "無效的BFile格式串"
    },
    {
      "id": "error.dataOverflow",
      "translation": "数字溢出"
    },
    {
      "id": "error.invalidDateTimeFormat",
      "translation": "錯誤的日期時間類型格式"
    },
    {
      "id": "error.datetimeOverflow",
      "translation": "数字溢出"
    },
    {
      "id": "error.invalidTimeInterval",
      "translation": "錯誤的時間間隔類型數據"
    },
    {
      "id": "error.unsupportedInparamType",
      "translation": "輸入參數類型不支持"
    },
    {
      "id": "error.unsupportedOutparamType",
      "translation": "輸出參數類型不支持"
    },
    {
      "id": "error.unsupportedType",
      "translation": "不支持該數據類型"
    },
    {
      "id": "error.invalidObjBlob",
      "translation": "無效的對象BLOB數據"
    },
    {
      "id": "error.structMemNotMatch",
      "translation": "記錄或類數據成員不匹配"
    },
    {
      "id": "error.invalidComplexTypeName",
      "translation": "無效的類型描述名稱"
    },
    {
      "id": "error.invalidParamterValue",
      "translation": "無效的參數值"
    },
    {
      "id": "error.invalidArrayLen",
      "translation": "靜態數組長度大於定義時長度"
    },
    {
      "id": "error.invalidSequenceNumber",
      "translation": "無效的列序號"
    },
    {
      "id": "error.resultsetInReadOnlyStatus",
      "translation": "結果集處於只讀狀態"
    },
    {
      "id": "error.SSLInitFailed",
      "translation": "初始化SSL環境失敗"
    },
    {
      "id": "error.LobDataHasFreed",
      "translation": "LOB數據已經被釋放"
    },
    {
      "id": "error.fatalError",
      "translation": "致命錯誤"
    },
    {
      "id": "error.invalidLenOrOffset",
      "translation": "長度或偏移錯誤"
    },
    {
      "id": "error.intervalValueOverflow",
      "translation": "時間間隔類型數據溢出"
    },
    {
      "id": "error.invalidCipher",
      "translation": "不支持的加密類型"
    },
    {
      "id": "error.storeInNilPointer",
      "translation": "無法將數據存入空指針"
    },
	{
      "id": "error.batchError",
	  "translation": "批量執行出錯"
	},
	{
      "id": "warning.bpWithErr",
	  "translation": "警告:批量執行部分行產生錯誤"
	},
	{
      "id": "error.invalidSqlType",
	  "translation": "非法的SQL語句類型"
	},
	{
      "id": "error.invalidDateTimeValue",
	  "translation": "無效的日期時間類型值"
	},
	{
      "id": "error.msgTooLong",
	  "translation": "消息長度超出限制512M"
	},
	{
      "id": "error.isNull",
	  "translation": "數據為NULL"
	},
	{
      "id": "error.ParamCountLimit",
	  "translation": "參數個數超過最大值65536."
	},
	{
      "id": "error.unbindedParameter",
	  "translation": "有參數未綁定"
	},
	{
      "id": "error.stringCut",
	  "translation": "字符串截斷"
	},
	{
      "id": "error.connectionClosedOrNotBuild",
	  "translation": "連接尚未建立或已經關閉"
	}
  ]
}`