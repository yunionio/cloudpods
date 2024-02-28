/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package dm

const (
	PARAM_COUNT_LIMIT int32 = 65536

	IGNORE_TARGET_LENGTH int32 = -1

	IGNORE_TARGET_SCALE int32 = -1

	IGNORE_TARGET_TYPE = INT32_MIN

	TYPE_FLAG_UNKNOWN byte = 0 // 未知类型

	TYPE_FLAG_EXACT byte = 1 // 精确类型

	TYPE_FLAG_RECOMMEND byte = 2 // 推荐类型

	IO_TYPE_UNKNOWN int8 = -1

	IO_TYPE_IN int8 = 0

	IO_TYPE_OUT int8 = 1

	IO_TYPE_INOUT int8 = 2

	MASK_ORACLE_DATE int32 = 1

	MASK_ORACLE_FLOAT int32 = 2

	MASK_BFILE int32 = 3

	MASK_LOCAL_DATETIME int32 = 4
)

type execRetInfo struct {
	// param
	outParamDatas [][]byte

	// rs
	hasResultSet bool

	rsDatas [][][]byte

	rsSizeof int // 结果集数据占用多少空间，（消息中结果集起始位置到 rsCacheOffset
	// 的空间大小，这和实际的rsDatas占用空间大小有一定出入，这里粗略估算，用于结果集缓存时的空间管理）

	rsCacheOffset int32 // 缓存信息，在响应消息体中的偏移，0表示不存在，仅结果集缓存中可以用

	rsBdta bool

	rsUpdatable bool

	rsRowIds []int64

	// rs cache
	tbIds []int32

	tbTss []int64

	// print
	printLen int32

	printMsg string

	// explain
	explain string

	// 影响行数
	updateCount int64 // Insert/Update/Delet影响行数， select结果集的总行数

	updateCounts []int64 // 批量影响行数

	// 键
	rowid int64

	lastInsertId int64

	// other
	retSqlType int16 // 执行返回的语句类型

	execId int32
}

type column struct {
	typeName string

	colType int32

	prec int32

	scale int32

	name string

	tableName string

	schemaName string

	nullable bool

	identity bool

	readonly bool // 是否只读

	baseName string

	// lob info
	lob bool

	lobTabId int32

	lobColId int16

	// 用于描述ARRAY、STRUCT类型的特有描述信息
	typeDescriptor *TypeDescriptor

	isBdta bool

	mask int32
}

type parameter struct {
	column

	typeFlag byte

	ioType int8

	outJType int32

	outScale int32

	outObjectName string

	cursorStmt *DmStatement

	hasDefault bool
}

func (column *column) InitColumn() *column {
	column.typeName = ""

	column.colType = 0

	column.prec = 0

	column.scale = 0

	column.name = ""

	column.tableName = ""

	column.schemaName = ""

	column.nullable = false

	column.identity = false

	column.readonly = false

	column.baseName = ""

	// lob info
	column.lob = false

	column.lobTabId = 0

	column.lobColId = 0

	// 用于描述ARRAY、STRUCT类型的特有描述信息
	column.typeDescriptor = nil

	column.isBdta = false

	return column
}

func (parameter *parameter) InitParameter() *parameter {
	parameter.InitColumn()

	parameter.typeFlag = TYPE_FLAG_UNKNOWN

	parameter.ioType = IO_TYPE_UNKNOWN

	parameter.outJType = IGNORE_TARGET_TYPE

	parameter.outScale = IGNORE_TARGET_SCALE

	parameter.outObjectName = ""

	parameter.cursorStmt = nil

	return parameter
}

func (parameter *parameter) resetType(colType int32) {
	parameter.colType = colType
	parameter.scale = 0
	switch colType {
	case BIT, BOOLEAN:
		parameter.prec = BIT_PREC
	case TINYINT:
		parameter.prec = TINYINT_PREC
	case SMALLINT:
		parameter.prec = SMALLINT_PREC
	case INT:
		parameter.prec = INT_PREC
	case BIGINT:
		parameter.prec = BIGINT_PREC
	case CHAR, VARCHAR, VARCHAR2:
		parameter.prec = VARCHAR_PREC
	case CLOB:
		parameter.prec = CLOB_PREC
	case BINARY, VARBINARY:
		parameter.prec = VARBINARY_PREC
	case BLOB:
		parameter.prec = BLOB_PREC
	case DATE:
		parameter.prec = DATE_PREC
	case TIME:
		parameter.prec = TIME_PREC
		parameter.scale = 6
	case TIME_TZ:
		parameter.prec = TIME_TZ_PREC
		parameter.scale = 6
	case DATETIME:
		parameter.prec = DATETIME_PREC
		parameter.scale = 6
	case DATETIME_TZ:
		parameter.prec = DATETIME_TZ_PREC
		parameter.scale = 6
	case DATETIME2:
		parameter.prec = DATETIME2_PREC
		parameter.scale = 9
	case DATETIME2_TZ:
		parameter.prec = DATETIME2_TZ_PREC
		parameter.scale = 9
	case REAL,DOUBLE,DECIMAL,INTERVAL_YM,INTERVAL_DT,ARRAY,CLASS,PLTYPE_RECORD,SARRAY:
		parameter.prec = 0
	case UNKNOWN, NULL:
		// UNKNOWN 导致服务器断言 // setNull导致服务器报错“字符转换失败”
		parameter.colType = VARCHAR
		parameter.prec = VARCHAR_PREC
	default:
	}
}

func (execInfo *execRetInfo) union(other *execRetInfo, startRow int, count int) {
	if count == 1 {
		execInfo.updateCounts[startRow] = other.updateCount
	} else if execInfo.updateCounts != nil {
		copy(execInfo.updateCounts[startRow:startRow+count], other.updateCounts[0:count])
	}
	if execInfo.outParamDatas != nil {
		execInfo.outParamDatas = append(execInfo.outParamDatas, other.outParamDatas...)
	}
}

func NewExceInfo() *execRetInfo {

	execInfo := execRetInfo{}

	execInfo.outParamDatas = nil

	execInfo.hasResultSet = false

	execInfo.rsDatas = nil

	execInfo.rsSizeof = 0

	execInfo.rsCacheOffset = 0

	execInfo.rsBdta = false

	execInfo.rsUpdatable = false

	execInfo.rsRowIds = nil

	execInfo.tbIds = nil

	execInfo.tbTss = nil

	execInfo.printLen = 0

	execInfo.printMsg = ""

	execInfo.explain = ""

	execInfo.updateCount = 0

	execInfo.updateCounts = nil

	execInfo.rowid = -1

	execInfo.lastInsertId = 0
	// other
	execInfo.retSqlType = -1 // 执行返回的语句类型

	execInfo.execId = 0

	return &execInfo
}
