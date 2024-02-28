/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package dm

import (
	"strconv"
	"time"

	"gitee.com/chunanyong/dm/util"
)

var DB2G db2g

type db2g struct {
}

func (DB2G db2g) processVarchar2(bytes []byte, prec int) []byte {
	rbytes := make([]byte, prec)
	copy(rbytes[:len(bytes)], bytes[:])
	for i := len(bytes); i < len(rbytes); i++ {
		rbytes[i] = ' '
	}
	return rbytes
}

func (DB2G db2g) charToString(bytes []byte, column *column, conn *DmConnection) string {
	if column.colType == VARCHAR2 {
		bytes = DB2G.processVarchar2(bytes, int(column.prec))
	} else if column.colType == CLOB {
		clob := newClobFromDB(bytes, conn, column, true)
		clobLen, _ := clob.GetLength()
		clobStr, _ := clob.getSubString(1, int32(clobLen))
		return clobStr
	}
	return Dm_build_650.Dm_build_902(bytes, conn.serverEncoding, conn)
}

func (DB2G db2g) charToFloat64(bytes []byte, column *column, conn *DmConnection) (float64, error) {
	str := DB2G.charToString(bytes, column, conn)
	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, ECGO_DATA_CONVERTION_ERROR.throw()
	}

	return val, nil
}

func (DB2G db2g) charToDeciaml(bytes []byte, column *column, conn *DmConnection) (*DmDecimal, error) {
	str := DB2G.charToString(bytes, column, conn)
	return NewDecimalFromString(str)
}

func (DB2G db2g) BinaryToInt64(bytes []byte, column *column, conn *DmConnection) (int64, error) {
	if column.colType == BLOB {
		blob := newBlobFromDB(bytes, conn, column, true)
		blobLen, err := blob.GetLength()
		if err != nil {
			return 0, err
		}
		bytes, err = blob.getBytes(1, int32(blobLen))
		if err != nil {
			return 0, err
		}
	}
	var n, b int64 = 0, 0

	startIndex := 0
	var length int
	if len(bytes) > 8 {
		length = 8
		for j := 0; j < len(bytes)-8; j++ {
			if bytes[j] != 0 {
				return 0, ECGO_DATA_CONVERTION_ERROR.throw()
			}

			startIndex = len(bytes) - 8
			length = 8
		}
	} else {
		length = len(bytes)
	}

	for j := startIndex; j < startIndex+length; j++ {
		b = int64(0xff & bytes[j])
		n = b | (n << 8)
	}

	return n, nil
}

func (DB2G db2g) decToDecimal(bytes []byte, prec int, scale int, compatibleOracle bool) (*DmDecimal, error) {

	if compatibleOracle {
		prec = -1
		scale = -1
	}
	return newDecimal(bytes, prec, scale)
}

func (DB2G db2g) toBytes(bytes []byte, column *column, conn *DmConnection) ([]byte, error) {
	retBytes := Dm_build_650.Dm_build_801(bytes, 0, len(bytes))
	switch column.colType {
	case CLOB:
		clob := newClobFromDB(retBytes, conn, column, true)
		str, err := clob.getSubString(1, int32(clob.length))
		if err != nil {
			return nil, err
		}

		return Dm_build_650.Dm_build_866(str, conn.getServerEncoding(), conn), nil
	case BLOB:
		blob := newBlobFromDB(retBytes, conn, column, true)
		bs, err := blob.getBytes(1, int32(blob.length))
		if err != nil {
			return nil, err
		}

		return bs, nil
	}
	return nil, ECGO_DATA_CONVERTION_ERROR.throw()
}

func (DB2G db2g) toString(bytes []byte, column *column, conn *DmConnection) string {
	switch column.colType {
	case CHAR, VARCHAR, VARCHAR2:
		return DB2G.charToString(bytes, column, conn)
	case BIT, BOOLEAN, TINYINT:
		return strconv.FormatInt(int64(bytes[0]), 10)
	case SMALLINT:
		return strconv.FormatInt(int64(Dm_build_650.Dm_build_874(bytes)), 10)
	case INT:
		return strconv.FormatInt(int64(Dm_build_650.Dm_build_877(bytes)), 10)
	case BIGINT:
		return strconv.FormatInt(int64(Dm_build_650.Dm_build_880(bytes)), 10)
	case REAL:
		return strconv.FormatFloat(float64(Dm_build_650.Dm_build_883(bytes)), 'f', -1, 32)
	case DOUBLE:
		return strconv.FormatFloat(float64(Dm_build_650.Dm_build_886(bytes)), 'f', -1, 64)
	case DECIMAL:

	case BINARY, VARBINARY:
		util.StringUtil.BytesToHexString(bytes, false)
	case BLOB:

	case CLOB:

	case DATE:
		dt := decode(bytes, column.isBdta, *column, int(conn.dmConnector.localTimezone), int(conn.DbTimezone))
		if conn.FormatDate != "" {
			return dtToStringByOracleFormat(dt, conn.FormatDate, column.scale, int(conn.OracleDateLanguage))
		}
	case TIME:
		dt := decode(bytes, column.isBdta, *column, int(conn.dmConnector.localTimezone), int(conn.DbTimezone))
		if conn.FormatTime != "" {
			return dtToStringByOracleFormat(dt, conn.FormatTime, column.scale, int(conn.OracleDateLanguage))
		}
	case DATETIME, DATETIME2:
		dt := decode(bytes, column.isBdta, *column, int(conn.dmConnector.localTimezone), int(conn.DbTimezone))
		if conn.FormatTimestamp != "" {
			return dtToStringByOracleFormat(dt, conn.FormatTimestamp, column.scale, int(conn.OracleDateLanguage))
		}
	case TIME_TZ:
		dt := decode(bytes, column.isBdta, *column, int(conn.dmConnector.localTimezone), int(conn.DbTimezone))
		if conn.FormatTimeTZ != "" {
			return dtToStringByOracleFormat(dt, conn.FormatTimeTZ, column.scale, int(conn.OracleDateLanguage))
		}
	case DATETIME_TZ, DATETIME2_TZ:
		dt := decode(bytes, column.isBdta, *column, int(conn.dmConnector.localTimezone), int(conn.DbTimezone))
		if conn.FormatTimestampTZ != "" {
			return dtToStringByOracleFormat(dt, conn.FormatTimestampTZ, column.scale, int(conn.OracleDateLanguage))
		}
	case INTERVAL_DT:
		return newDmIntervalDTByBytes(bytes).String()
	case INTERVAL_YM:
		return newDmIntervalYMByBytes(bytes).String()
	case ARRAY:

	case SARRAY:

	case CLASS:

	case PLTYPE_RECORD:

	}
	return ""
}

func (DB2G db2g) toBool(bytes []byte, column *column, conn *DmConnection) (bool, error) {
	switch column.colType {
	case BIT, BOOLEAN, TINYINT:
		return bytes[0] != 0, nil
	case SMALLINT:
		return Dm_build_650.Dm_build_747(bytes, 0) != 0, nil
	case INT:
		return Dm_build_650.Dm_build_752(bytes, 0) != 0, nil
	case BIGINT:
		return Dm_build_650.Dm_build_757(bytes, 0) != 0, nil
	case REAL:
		return Dm_build_650.Dm_build_762(bytes, 0) != 0, nil
	case DOUBLE:
		return Dm_build_650.Dm_build_766(bytes, 0) != 0, nil
	case DECIMAL:

	case CHAR, VARCHAR, VARCHAR2, CLOB:
		return G2DB.toBool(DB2G.charToString(bytes, column, conn))
	}

	return false, ECGO_DATA_CONVERTION_ERROR.throw()
}

func (DB2G db2g) toByte(bytes []byte, column *column, conn *DmConnection) (byte, error) {
	switch column.colType {
	case BIT, BOOLEAN, TINYINT:

		if bytes == nil || len(bytes) == 0 {
			return 0, nil
		} else {
			return bytes[0], nil
		}
	case SMALLINT:
		tval := Dm_build_650.Dm_build_747(bytes, 0)
		if tval < int16(BYTE_MIN) || tval > int16(BYTE_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return byte(tval), nil
	case INT:
		tval := Dm_build_650.Dm_build_752(bytes, 0)
		if tval < int32(BYTE_MIN) || tval > int32(BYTE_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return byte(tval), nil
	case BIGINT:
		tval := Dm_build_650.Dm_build_757(bytes, 0)
		if tval < int64(BYTE_MIN) || tval > int64(BYTE_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return byte(tval), nil
	case REAL:
		tval := Dm_build_650.Dm_build_762(bytes, 0)
		if tval < float32(BYTE_MIN) || tval > float32(BYTE_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return byte(tval), nil
	case DOUBLE:
		tval := Dm_build_650.Dm_build_766(bytes, 0)
		if tval < float64(BYTE_MIN) || tval > float64(BYTE_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return byte(tval), nil
	case DECIMAL:

	case CHAR, VARCHAR, VARCHAR2, CLOB:
		tval, err := DB2G.charToFloat64(bytes, column, conn)
		if err != nil {
			return 0, err
		}

		if tval < float64(BYTE_MIN) || tval > float64(BYTE_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return byte(tval), nil
	case BINARY, VARBINARY, BLOB:
		{
			tval, err := DB2G.BinaryToInt64(bytes, column, conn)
			if err != nil {
				return 0, err
			}

			if tval < int64(BYTE_MIN) || tval > int64(BYTE_MAX) {
				return 0, ECGO_DATA_OVERFLOW.throw()
			}
			return byte(tval), nil
		}
	}

	return 0, ECGO_DATA_CONVERTION_ERROR.throw()
}

func (DB2G db2g) toInt8(bytes []byte, column *column, conn *DmConnection) (int8, error) {
	switch column.colType {
	case BIT, BOOLEAN, TINYINT:
		if bytes == nil || len(bytes) == 0 {
			return 0, nil
		}

		return int8(bytes[0]), nil
	case SMALLINT:
		tval := Dm_build_650.Dm_build_747(bytes, 0)
		if tval < int16(INT8_MIN) || tval < int16(INT8_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return int8(tval), nil
	case INT:

		tval := Dm_build_650.Dm_build_752(bytes, 0)
		if tval < int32(INT8_MIN) || tval > int32(INT8_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return int8(tval), nil
	case BIGINT:
		tval := Dm_build_650.Dm_build_757(bytes, 0)
		if tval < int64(INT8_MIN) || tval > int64(INT8_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return int8(tval), nil
	case REAL:
		tval := Dm_build_650.Dm_build_762(bytes, 0)
		if tval < float32(INT8_MIN) || tval > float32(INT8_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return int8(tval), nil
	case DOUBLE:
		tval := Dm_build_650.Dm_build_766(bytes, 0)
		if tval < float64(INT8_MIN) || tval > float64(INT8_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return int8(tval), nil
	case DECIMAL:

	case CHAR, VARCHAR, VARCHAR2, CLOB:
		tval, err := DB2G.charToFloat64(bytes, column, conn)
		if err != nil {
			return 0, err
		}

		if tval < float64(INT8_MIN) || tval > float64(INT8_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return int8(tval), nil
	case BINARY, VARBINARY, BLOB:
		{
			tval, err := DB2G.BinaryToInt64(bytes, column, conn)
			if err != nil {
				return 0, err
			}

			if tval < int64(INT8_MIN) || tval > int64(INT8_MAX) {
				return 0, ECGO_DATA_OVERFLOW.throw()
			}
			return int8(tval), nil
		}
	}

	return 0, ECGO_DATA_CONVERTION_ERROR.throw()
}

func (DB2G db2g) toInt16(bytes []byte, column *column, conn *DmConnection) (int16, error) {
	switch column.colType {
	case BIT, BOOLEAN, TINYINT:
		if bytes == nil || len(bytes) == 0 {
			return 0, nil
		}

		return int16(bytes[0]), nil
	case SMALLINT:
		return Dm_build_650.Dm_build_747(bytes, 0), nil
	case INT:

		tval := Dm_build_650.Dm_build_752(bytes, 0)
		if tval < int32(INT16_MIN) || tval > int32(INT16_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return int16(tval), nil
	case BIGINT:
		tval := Dm_build_650.Dm_build_757(bytes, 0)
		if tval < int64(INT16_MIN) || tval > int64(INT16_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return int16(tval), nil
	case REAL:
		tval := Dm_build_650.Dm_build_762(bytes, 0)
		if tval < float32(INT16_MIN) || tval > float32(INT16_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return int16(tval), nil
	case DOUBLE:
		tval := Dm_build_650.Dm_build_766(bytes, 0)
		if tval < float64(INT16_MIN) || tval > float64(INT16_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return int16(tval), nil
	case DECIMAL:

	case CHAR, VARCHAR, VARCHAR2, CLOB:
		tval, err := DB2G.charToFloat64(bytes, column, conn)
		if err != nil {
			return 0, err
		}

		if tval < float64(INT16_MIN) || tval > float64(INT16_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return int16(tval), nil
	case BINARY, VARBINARY, BLOB:
		{
			tval, err := DB2G.BinaryToInt64(bytes, column, conn)
			if err != nil {
				return 0, err
			}

			if tval < int64(INT16_MIN) || tval > int64(INT16_MAX) {
				return 0, ECGO_DATA_OVERFLOW.throw()
			}
			return int16(tval), nil
		}
	}

	return 0, ECGO_DATA_CONVERTION_ERROR.throw()
}

func (DB2G db2g) toUInt16(bytes []byte, column *column, conn *DmConnection) (uint16, error) {
	switch column.colType {
	case BIT, BOOLEAN, TINYINT:
		if bytes == nil || len(bytes) == 0 {
			return 0, nil
		}

		return uint16(bytes[0]), nil
	case SMALLINT:
		return uint16(Dm_build_650.Dm_build_747(bytes, 0)), nil
	case INT:
		tval := Dm_build_650.Dm_build_752(bytes, 0)
		if tval < int32(UINT16_MIN) || tval > int32(UINT16_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return uint16(tval), nil
	case BIGINT:
		tval := Dm_build_650.Dm_build_757(bytes, 0)
		if tval < int64(UINT16_MIN) || tval > int64(UINT16_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return uint16(tval), nil
	case REAL:
		tval := Dm_build_650.Dm_build_762(bytes, 0)
		if tval < float32(UINT16_MIN) || tval > float32(UINT16_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return uint16(tval), nil
	case DOUBLE:
		tval := Dm_build_650.Dm_build_766(bytes, 0)
		if tval < float64(UINT16_MIN) || tval > float64(UINT16_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return uint16(tval), nil
	case DECIMAL:

	case CHAR, VARCHAR, VARCHAR2, CLOB:
		tval, err := DB2G.charToFloat64(bytes, column, conn)
		if err != nil {
			return 0, err
		}

		if tval < float64(UINT16_MIN) || tval > float64(UINT16_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return uint16(tval), nil
	case BINARY, VARBINARY, BLOB:
		{
			tval, err := DB2G.BinaryToInt64(bytes, column, conn)
			if err != nil {
				return 0, err
			}

			if tval < int64(UINT16_MIN) || tval > int64(UINT16_MAX) {
				return 0, ECGO_DATA_OVERFLOW.throw()
			}
			return uint16(tval), nil
		}
	}

	return 0, ECGO_DATA_CONVERTION_ERROR.throw()
}

func (DB2G db2g) toInt32(bytes []byte, column *column, conn *DmConnection) (int32, error) {
	switch column.colType {
	case BIT, BOOLEAN, TINYINT:
		if bytes == nil || len(bytes) == 0 {
			return 0, nil
		}

		return int32(bytes[0]), nil
	case SMALLINT:
		return int32(Dm_build_650.Dm_build_747(bytes, 0)), nil
	case INT:
		return Dm_build_650.Dm_build_752(bytes, 0), nil
	case BIGINT:
		tval := Dm_build_650.Dm_build_757(bytes, 0)
		if tval < int64(INT32_MIN) || tval > int64(INT32_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return int32(tval), nil
	case REAL:
		tval := Dm_build_650.Dm_build_762(bytes, 0)
		if tval < float32(INT32_MIN) || tval > float32(INT32_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return int32(tval), nil
	case DOUBLE:
		tval := Dm_build_650.Dm_build_766(bytes, 0)
		if tval < float64(INT32_MIN) || tval > float64(INT32_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return int32(tval), nil
	case DECIMAL:

	case CHAR, VARCHAR, VARCHAR2, CLOB:
		tval, err := DB2G.charToFloat64(bytes, column, conn)
		if err != nil {
			return 0, err
		}

		if tval < float64(INT32_MIN) || tval > float64(INT32_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return int32(tval), nil
	case BINARY, VARBINARY, BLOB:
		{
			tval, err := DB2G.BinaryToInt64(bytes, column, conn)
			if err != nil {
				return 0, err
			}

			if tval < int64(INT32_MIN) || tval > int64(INT32_MAX) {
				return 0, ECGO_DATA_OVERFLOW.throw()
			}
			return int32(tval), nil
		}
	}

	return 0, ECGO_DATA_CONVERTION_ERROR.throw()
}

func (DB2G db2g) toUInt32(bytes []byte, column *column, conn *DmConnection) (uint32, error) {
	switch column.colType {
	case BIT, BOOLEAN, TINYINT:
		if bytes == nil || len(bytes) == 0 {
			return 0, nil
		}

		return uint32(bytes[0]), nil
	case SMALLINT:
		return uint32(Dm_build_650.Dm_build_747(bytes, 0)), nil
	case INT:
		return uint32(Dm_build_650.Dm_build_752(bytes, 0)), nil
	case BIGINT:
		tval := Dm_build_650.Dm_build_757(bytes, 0)
		if tval < int64(UINT32_MIN) || tval > int64(UINT32_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return uint32(tval), nil
	case REAL:
		tval := Dm_build_650.Dm_build_762(bytes, 0)
		if tval < float32(UINT32_MIN) || tval > float32(UINT32_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return uint32(tval), nil
	case DOUBLE:
		tval := Dm_build_650.Dm_build_766(bytes, 0)
		if tval < float64(UINT32_MIN) || tval > float64(UINT32_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return uint32(tval), nil
	case DECIMAL:

	case CHAR, VARCHAR, VARCHAR2, CLOB:
		tval, err := DB2G.charToFloat64(bytes, column, conn)
		if err != nil {
			return 0, err
		}

		if tval < float64(UINT32_MIN) || tval > float64(UINT32_MAX) {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return uint32(tval), nil
	case BINARY, VARBINARY, BLOB:
		{
			tval, err := DB2G.BinaryToInt64(bytes, column, conn)
			if err != nil {
				return 0, err
			}

			if tval < int64(UINT32_MIN) || tval > int64(UINT32_MAX) {
				return 0, ECGO_DATA_OVERFLOW.throw()
			}
			return uint32(tval), nil
		}
	}

	return 0, ECGO_DATA_CONVERTION_ERROR.throw()
}

func (DB2G db2g) toInt64(bytes []byte, column *column, conn *DmConnection) (int64, error) {
	switch column.colType {
	case BOOLEAN, BIT, TINYINT:
		if bytes == nil || len(bytes) == 0 {
			return int64(0), nil
		} else {
			return int64(bytes[0]), nil
		}
	case SMALLINT:
		return int64(Dm_build_650.Dm_build_874(bytes)), nil
	case INT:
		return int64(Dm_build_650.Dm_build_877(bytes)), nil
	case BIGINT:
		return int64(Dm_build_650.Dm_build_880(bytes)), nil
	case REAL:
		return int64(Dm_build_650.Dm_build_883(bytes)), nil
	case DOUBLE:
		return int64(Dm_build_650.Dm_build_886(bytes)), nil

	case CHAR, VARCHAR2, VARCHAR, CLOB:
		tval, err := DB2G.charToFloat64(bytes, column, conn)
		if err != nil {
			return 0, err
		}

		if int64(tval) < INT64_MIN || int64(tval) > INT64_MAX {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return int64(tval), nil
	case BINARY, VARBINARY, BLOB:
		tval, err := DB2G.BinaryToInt64(bytes, column, conn)
		if err != nil {
			return 0, err
		}

		return tval, nil
	}
	return 0, ECGO_DATA_CONVERTION_ERROR.throw()
}

func (DB2G db2g) toUInt64(bytes []byte, column *column, conn *DmConnection) (uint64, error) {
	switch column.colType {
	case BOOLEAN, BIT, TINYINT:
		if bytes == nil || len(bytes) == 0 {
			return uint64(0), nil
		} else {
			return uint64(bytes[0]), nil
		}
	case SMALLINT:
		return uint64(Dm_build_650.Dm_build_874(bytes)), nil
	case INT:
		return uint64(Dm_build_650.Dm_build_877(bytes)), nil
	case BIGINT:
		return uint64(Dm_build_650.Dm_build_880(bytes)), nil
	case REAL:
		return uint64(Dm_build_650.Dm_build_883(bytes)), nil
	case DOUBLE:
		return uint64(Dm_build_650.Dm_build_886(bytes)), nil

	case CHAR, VARCHAR2, VARCHAR, CLOB:
		tval, err := DB2G.charToFloat64(bytes, column, conn)
		if err != nil {
			return 0, err
		}

		if uint64(tval) < UINT64_MIN || uint64(tval) > UINT64_MAX {
			return 0, ECGO_DATA_OVERFLOW.throw()
		}
		return uint64(tval), nil
	case BINARY, VARBINARY, BLOB:
		tval, err := DB2G.BinaryToInt64(bytes, column, conn)
		if err != nil {
			return 0, err
		}

		return uint64(tval), nil
	}
	return 0, ECGO_DATA_CONVERTION_ERROR.throw()
}

func (DB2G db2g) toInt(bytes []byte, column *column, conn *DmConnection) (int, error) {
	if strconv.IntSize == 32 {
		tmp, err := DB2G.toInt32(bytes, column, conn)
		return int(tmp), err
	} else {
		tmp, err := DB2G.toInt64(bytes, column, conn)
		return int(tmp), err
	}
}

func (DB2G db2g) toUInt(bytes []byte, column *column, conn *DmConnection) (uint, error) {
	if strconv.IntSize == 32 {
		tmp, err := DB2G.toUInt32(bytes, column, conn)
		return uint(tmp), err
	} else {
		tmp, err := DB2G.toUInt64(bytes, column, conn)
		return uint(tmp), err
	}
}

func (DB2G db2g) toFloat32(bytes []byte, column *column, conn *DmConnection) (float32, error) {
	switch column.colType {
	case BIT, BOOLEAN, TINYINT:
		if bytes == nil || len(bytes) == 0 {
			return 0, nil
		}
		return float32(bytes[0]), nil
	case SMALLINT:
		return float32(Dm_build_650.Dm_build_747(bytes, 0)), nil
	case INT:
		return float32(Dm_build_650.Dm_build_752(bytes, 0)), nil
	case BIGINT:
		return float32(Dm_build_650.Dm_build_757(bytes, 0)), nil
	case REAL:
		return Dm_build_650.Dm_build_762(bytes, 0), nil
	case DOUBLE:
		dval := Dm_build_650.Dm_build_766(bytes, 0)
		return float32(dval), nil
	case DECIMAL:
		dval, err := DB2G.decToDecimal(bytes, int(column.prec), int(column.scale), conn.CompatibleOracle())
		if err != nil {
			return 0, err
		}
		return float32(dval.ToFloat64()), nil
	case CHAR, VARCHAR2, VARCHAR, CLOB:
		dval, err := DB2G.charToDeciaml(bytes, column, conn)
		if err != nil {
			return 0, err
		}
		return float32(dval.ToFloat64()), nil
	}
	return 0, ECGO_DATA_CONVERTION_ERROR.throw()
}

func (DB2G db2g) toFloat64(bytes []byte, column *column, conn *DmConnection) (float64, error) {
	switch column.colType {
	case BIT, BOOLEAN, TINYINT:
		if bytes == nil || len(bytes) == 0 {
			return 0, nil
		}
		return float64(bytes[0]), nil
	case SMALLINT:
		return float64(Dm_build_650.Dm_build_747(bytes, 0)), nil
	case INT:
		return float64(Dm_build_650.Dm_build_752(bytes, 0)), nil
	case BIGINT:
		return float64(Dm_build_650.Dm_build_757(bytes, 0)), nil
	case REAL:
		return float64(Dm_build_650.Dm_build_762(bytes, 0)), nil
	case DOUBLE:
		return Dm_build_650.Dm_build_766(bytes, 0), nil
	case DECIMAL:
		dval, err := DB2G.decToDecimal(bytes, int(column.prec), int(column.scale), conn.CompatibleOracle())
		if err != nil {
			return 0, err
		}
		return dval.ToFloat64(), nil
	case CHAR, VARCHAR2, VARCHAR, CLOB:
		dval, err := DB2G.charToDeciaml(bytes, column, conn)
		if err != nil {
			return 0, err
		}
		return dval.ToFloat64(), nil
	}

	return 0, ECGO_DATA_CONVERTION_ERROR.throw()
}

func (DB2G db2g) toDmBlob(value []byte, column *column, conn *DmConnection) *DmBlob {

	switch column.colType {
	case BLOB:
		return newBlobFromDB(value, conn, column, conn.lobFetchAll())
	default:
		return newBlobOfLocal(value, conn)
	}
}

func (DB2G db2g) toDmClob(value []byte, conn *DmConnection, column *column) *DmClob {

	switch column.colType {
	case CLOB:
		return newClobFromDB(value, conn, column, conn.lobFetchAll())
	default:
		return newClobOfLocal(DB2G.toString(value, column, conn), conn)
	}
}

func (DB2G db2g) toDmDecimal(value []byte, column *column, conn *DmConnection) (*DmDecimal, error) {

	switch column.colType {
	case BIT, BOOLEAN, TINYINT:
		if value == nil || len(value) == 0 {
			return NewDecimalFromInt64(0)
		} else {
			return NewDecimalFromInt64(int64(value[0]))
		}
	case SMALLINT:
		return NewDecimalFromInt64(int64(Dm_build_650.Dm_build_747(value, 0)))
	case INT:
		return NewDecimalFromInt64(int64(Dm_build_650.Dm_build_752(value, 0)))
	case BIGINT:
		return NewDecimalFromInt64(Dm_build_650.Dm_build_757(value, 0))
	case REAL:
		return NewDecimalFromFloat64(float64(Dm_build_650.Dm_build_762(value, 0)))
	case DOUBLE:
		return NewDecimalFromFloat64(Dm_build_650.Dm_build_766(value, 0))
	case DECIMAL:
		return decodeDecimal(value, int(column.prec), int(column.scale))
	case CHAR, VARCHAR, VARCHAR2, CLOB:
		return DB2G.charToDeciaml(value, column, conn)
	}

	return nil, ECGO_DATA_CONVERTION_ERROR
}

func (DB2G db2g) toTime(bytes []byte, column *column, conn *DmConnection) (time.Time, error) {
	switch column.colType {
	case DATE, TIME, TIME_TZ, DATETIME_TZ, DATETIME, DATETIME2_TZ, DATETIME2:
		dt := decode(bytes, column.isBdta, *column, int(conn.dmConnector.localTimezone), int(conn.DbTimezone))
		return toTimeFromDT(dt, int(conn.dmConnector.localTimezone)), nil
	case CHAR, VARCHAR2, VARCHAR, CLOB:
		return toTimeFromString(DB2G.charToString(bytes, column, conn), int(conn.dmConnector.localTimezone)), nil
	}
	return time.Now(), ECGO_DATA_CONVERTION_ERROR.throw()
}

func (DB2G db2g) toObject(bytes []byte, column *column, conn *DmConnection) (interface{}, error) {

	switch column.colType {
	case BIT, BOOLEAN:
		return bytes[0] != 0, nil

	case TINYINT:

		return Dm_build_650.Dm_build_743(bytes, 0), nil
	case SMALLINT:
		return Dm_build_650.Dm_build_747(bytes, 0), nil
	case INT:
		return Dm_build_650.Dm_build_752(bytes, 0), nil
	case BIGINT:
		return Dm_build_650.Dm_build_757(bytes, 0), nil
	case DECIMAL:
		return DB2G.decToDecimal(bytes, int(column.prec), int(column.scale), conn.CompatibleOracle())
	case REAL:
		return Dm_build_650.Dm_build_762(bytes, 0), nil
	case DOUBLE:
		return Dm_build_650.Dm_build_766(bytes, 0), nil
	case DATE, TIME, DATETIME, TIME_TZ, DATETIME_TZ, DATETIME2, DATETIME2_TZ:
		dt := decode(bytes, column.isBdta, *column, int(conn.dmConnector.localTimezone), int(conn.DbTimezone))
		return toTimeFromDT(dt, int(conn.dmConnector.localTimezone)), nil
	case BINARY, VARBINARY:
		return bytes, nil
	case BLOB:
		blob := newBlobFromDB(bytes, conn, column, conn.lobFetchAll())

		if util.StringUtil.EqualsIgnoreCase(column.typeName, "LONGVARBINARY") {

			l, err := blob.GetLength()
			if err != nil {
				return nil, err
			}
			return blob.getBytes(1, int32(l))
		} else {
			return blob, nil
		}
	case CHAR, VARCHAR, VARCHAR2:
		val := DB2G.charToString(bytes, column, conn)
		if column.mask == MASK_BFILE {

		}

		return val, nil
	case CLOB:
		clob := newClobFromDB(bytes, conn, column, conn.lobFetchAll())
		if util.StringUtil.EqualsIgnoreCase(column.typeName, "LONGVARCHAR") {

			l, err := clob.GetLength()
			if err != nil {
				return nil, err
			}
			return clob.getSubString(1, int32(l))
		} else {
			return clob, nil
		}
	case INTERVAL_YM:
		return newDmIntervalYMByBytes(bytes), nil
	case INTERVAL_DT:
		return newDmIntervalDTByBytes(bytes), nil
	case ARRAY:
		return TypeDataSV.bytesToArray(bytes, nil, column.typeDescriptor)
	case SARRAY:
		return TypeDataSV.bytesToSArray(bytes, nil, column.typeDescriptor)
	case CLASS:

	case PLTYPE_RECORD:

	default:
		return nil, ECGO_DATA_CONVERTION_ERROR.throw()
	}

	return nil, ECGO_DATA_CONVERTION_ERROR.throw()
}

func (DB2G db2g) toComplexType(bytes []byte, column *column, conn *DmConnection) (interface{}, error) {
	switch column.colType {
	case BLOB:
		if !isComplexType(int(column.colType), int(column.scale)) {
			return nil, ECGO_DATA_CONVERTION_ERROR.throw()
		}
		blob := newBlobFromDB(bytes, conn, column, true)
		return TypeDataSV.objBlobToObj(blob, column.typeDescriptor)
	case ARRAY:
		return TypeDataSV.bytesToArray(bytes, nil, column.typeDescriptor)
	case SARRAY:
		return TypeDataSV.bytesToSArray(bytes, nil, column.typeDescriptor)
	case CLASS:
		return TypeDataSV.bytesToObj(bytes, nil, column.typeDescriptor)
	case PLTYPE_RECORD:
		return nil, ECGO_DATA_CONVERTION_ERROR.throw()
	default:
		return nil, ECGO_DATA_CONVERTION_ERROR.throw()
	}
}
