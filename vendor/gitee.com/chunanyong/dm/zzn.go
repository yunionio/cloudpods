/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package dm

import (
	"database/sql"
	"database/sql/driver"
	"math"
	"reflect"
	"strings"
	"time"
)

const (
	INT8_MAX int8 = math.MaxInt8

	INT8_MIN int8 = math.MinInt8

	BYTE_MAX byte = math.MaxUint8

	BYTE_MIN byte = 0

	INT16_MAX int16 = math.MaxInt16

	INT16_MIN int16 = math.MinInt16

	UINT16_MAX uint16 = math.MaxUint16

	UINT16_MIN uint16 = 0

	INT32_MAX int32 = math.MaxInt32

	INT32_MIN int32 = math.MinInt32

	UINT32_MAX uint32 = math.MaxUint32

	UINT32_MIN uint32 = 0

	INT64_MAX int64 = math.MaxInt64

	INT64_MIN int64 = math.MinInt64

	UINT64_MAX uint64 = math.MaxUint64

	UINT64_MIN uint64 = 0

	FLOAT32_MAX float32 = 3.4e+38

	FLOAT32_MIN float32 = -3.4e+38

	BYTE_SIZE = 1

	USINT_SIZE = 2

	ULINT_SIZE = 4

	DDWORD_SIZE = 8

	LINT64_SIZE = 8

	CHAR = 0

	VARCHAR2 = 1

	VARCHAR = 2

	BIT = 3

	TINYINT = 5

	SMALLINT = 6

	INT = 7

	BIGINT = 8

	DECIMAL = 9

	REAL = 10

	DOUBLE = 11

	BLOB = 12

	BOOLEAN = 13

	DATE = 14

	TIME = 15

	DATETIME = 16

	BINARY = 17

	VARBINARY = 18

	CLOB = 19

	INTERVAL_YM = 20

	INTERVAL_DT = 21

	TIME_TZ = 22

	DATETIME_TZ = 23

	XDEC_INT32 = 24

	XDEC_INT64 = 25

	DATETIME2 = 26

	DATETIME2_TZ = 27

	NULL = 28

	ANY = 31

	STAR_ALL = 32

	STAR = 33

	RECORD = 40

	TYPE = 41

	TYPE_REF = 42

	UNKNOWN = 54

	ARRAY = 117

	CLASS = 119

	CURSOR = 120

	PLTYPE_RECORD = 121

	SARRAY = 122

	CURSOR_ORACLE = -10

	BIT_PREC = BYTE_SIZE

	TINYINT_PREC = BYTE_SIZE

	SMALLINT_PREC = USINT_SIZE

	INT_PREC = ULINT_SIZE

	BIGINT_PREC = LINT64_SIZE

	REAL_PREC = 4

	DOUBLE_PREC = 8

	DATE_PREC = 3

	TIME_PREC = 5

	DATETIME_PREC = 8

	DATETIME2_PREC = 9

	TIME_TZ_PREC = TIME_PREC + 2

	DATETIME_TZ_PREC = DATETIME_PREC + 2

	DATETIME2_TZ_PREC = DATETIME2_PREC + 2

	INTERVAL_YM_PREC = 3 * ULINT_SIZE

	INTERVAL_DT_PREC = 6 * ULINT_SIZE

	VARCHAR_PREC = 8188

	VARBINARY_PREC = 8188

	BLOB_PREC int32 = INT32_MAX

	CLOB_PREC int32 = INT32_MAX

	NULL_PREC = 0

	LOCAL_TIME_ZONE_SCALE_MASK = 0x00001000

	BFILE_PREC = 512

	BFILE_SCALE = 6

	COMPLEX_SCALE = 5

	CURRENCY_PREC = 19

	CURRENCY_SCALE = 4

	LOCAL_DATETIME_SCALE_MASK int32 = 0x00001000

	ORACLE_FLOAT_SCALE_MASK int32 = 0x81

	ORACLE_DATE_SCALE_MASK int32 = 0x00002000
)

func isComplexType(colType int, scale int) bool {
	return (colType == BLOB && scale == COMPLEX_SCALE) || colType == ARRAY || colType == SARRAY || colType == CLASS || colType == PLTYPE_RECORD
}

func isLocalTimeZone(colType int, scale int) bool {
	return (colType == DATETIME || colType == DATETIME2) && (scale&LOCAL_TIME_ZONE_SCALE_MASK) != 0
}

func getLocalTimeZoneScale(colType int, scale int) int {
	return scale & (^LOCAL_TIME_ZONE_SCALE_MASK)
}

func isFloat(colType int, scale int) bool {
	return colType == DECIMAL && scale == int(ORACLE_FLOAT_SCALE_MASK)
}

func getFloatPrec(prec int) int {
	return int(math.Round(float64(prec)*0.30103)) + 1
}

func getFloatScale(scale int) int {
	return scale & (^int(ORACLE_FLOAT_SCALE_MASK))
}

var (
	scanTypeFloat32    = reflect.TypeOf(float32(0))
	scanTypeFloat64    = reflect.TypeOf(float64(0))
	scanTypeBool       = reflect.TypeOf(false)
	scanTypeInt8       = reflect.TypeOf(int8(0))
	scanTypeInt16      = reflect.TypeOf(int16(0))
	scanTypeInt32      = reflect.TypeOf(int32(0))
	scanTypeInt64      = reflect.TypeOf(int64(0))
	scanTypeNullBool   = reflect.TypeOf(sql.NullBool{})
	scanTypeNullFloat  = reflect.TypeOf(sql.NullFloat64{})
	scanTypeNullInt    = reflect.TypeOf(sql.NullInt64{})
	scanTypeNullString = reflect.TypeOf(sql.NullString{})
	scanTypeNullTime   = reflect.TypeOf(sql.NullTime{})
	scanTypeRawBytes   = reflect.TypeOf(sql.RawBytes{})
	scanTypeString     = reflect.TypeOf("")
	scanTypeTime       = reflect.TypeOf(time.Now())
	scanTypeUnknown    = reflect.TypeOf(new(interface{}))
)

func (column *column) ScanType() reflect.Type {

	switch column.colType {
	case BOOLEAN:
		if column.nullable {
			return scanTypeNullBool
		}

		return scanTypeBool

	case BIT:
		if strings.ToLower(column.typeName) == "boolean" {

			if column.nullable {
				return scanTypeNullBool
			}

			return scanTypeBool
		} else {

			if column.nullable {
				return scanTypeNullInt
			}
			return scanTypeInt8
		}

	case TINYINT:
		if column.nullable {
			return scanTypeNullInt
		}
		return scanTypeInt8

	case SMALLINT:
		if column.nullable {
			return scanTypeNullInt
		}
		return scanTypeInt16

	case INT:
		if column.nullable {
			return scanTypeNullInt
		}

		return scanTypeInt32

	case BIGINT:
		if column.nullable {
			return scanTypeNullInt
		}
		return scanTypeInt64

	case REAL:
		if column.nullable {
			return scanTypeNullFloat
		}

		return scanTypeFloat32

	case DOUBLE:

		if strings.ToLower(column.typeName) == "float" {
			if column.nullable {
				return scanTypeNullFloat
			}

			return scanTypeFloat32
		}

		if column.nullable {
			return scanTypeNullFloat
		}

		return scanTypeFloat64
	case DATE, TIME, TIME_TZ, DATETIME, DATETIME_TZ, DATETIME2, DATETIME2_TZ:
		if column.nullable {
			return scanTypeNullTime
		}

		return scanTypeTime

	case DECIMAL, BINARY, VARBINARY, BLOB:
		return scanTypeRawBytes

	case CHAR, VARCHAR2, VARCHAR, CLOB:
		if column.nullable {
			return scanTypeNullString
		}
		return scanTypeString
	}

	return scanTypeUnknown
}

func (column *column) Length() (length int64, ok bool) {
	switch column.colType {
	case BINARY, VARBINARY, BLOB, CHAR, VARCHAR2, VARCHAR, CLOB:
		return int64(column.prec), true
	}

	return int64(0), false
}

func (column *column) PrecisionScale() (precision, scale int64, ok bool) {
	switch column.colType {
	case DECIMAL:
		if column.prec == 0 {
			return 38, int64(column.scale), true
		} else {
			return int64(column.prec), int64(column.scale), true
		}
	}

	return int64(0), int64(0), false
}

func (column *column) getColumnData(bytes []byte, conn *DmConnection) (driver.Value, error) {
	if bytes == nil {
		return nil, nil
	}

	switch column.colType {
	case BOOLEAN:
		return bytes[0] != 0, nil
	case BIT:
		if strings.ToLower(column.typeName) == "boolean" {
			return bytes[0] != 0, nil
		}

		return int8(bytes[0]), nil
	case TINYINT:
		return int8(bytes[0]), nil
	case SMALLINT:
		return Dm_build_650.Dm_build_747(bytes, 0), nil
	case INT:
		return Dm_build_650.Dm_build_752(bytes, 0), nil
	case BIGINT:
		return Dm_build_650.Dm_build_757(bytes, 0), nil
	case REAL:
		return Dm_build_650.Dm_build_762(bytes, 0), nil
	case DOUBLE:

		return Dm_build_650.Dm_build_766(bytes, 0), nil
	case DATE, TIME, DATETIME, TIME_TZ, DATETIME_TZ, DATETIME2, DATETIME2_TZ:
		return DB2G.toTime(bytes, column, conn)
	case INTERVAL_DT:
		return newDmIntervalDTByBytes(bytes).String(), nil
	case INTERVAL_YM:
		return newDmIntervalYMByBytes(bytes).String(), nil
	case DECIMAL:
		tmp, err := DB2G.toDmDecimal(bytes, column, conn)
		if err != nil {
			return nil, err
		}
		return tmp.String(), nil

	case BINARY, VARBINARY:
		return bytes, nil
	case BLOB:
		if isComplexType(int(column.colType), int(column.scale)) {
			return DB2G.toComplexType(bytes, column, conn)
		}
		blob := DB2G.toDmBlob(bytes, column, conn)

		l, err := blob.GetLength()
		if err != nil {
			return nil, err
		}
		return blob.getBytes(1, int32(l))

	case CHAR, VARCHAR2, VARCHAR:
		return Dm_build_650.Dm_build_807(bytes, 0, len(bytes), conn.getServerEncoding(), conn), nil
	case CLOB:
		clob := DB2G.toDmClob(bytes, conn, column)

		l, err := clob.GetLength()
		if err != nil {
			return nil, err
		}
		return clob.getSubString(1, int32(l))

	}

	return string(bytes), nil
}

func emptyStringToNil(t int32) bool {
	switch t {
	case BOOLEAN, BIT, TINYINT, SMALLINT, INT, BIGINT, REAL, DOUBLE, DECIMAL, DATE, TIME,
		DATETIME, INTERVAL_DT, INTERVAL_YM, TIME_TZ, DATETIME_TZ, DATETIME2, DATETIME2_TZ:
		return true
	default:
		return false
	}
}
