/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package dm

import (
	"bytes"
	"io"
	"math/big"
	"strconv"
	"strings"
	"time"

	"gitee.com/chunanyong/dm/util"
)

var G2DB g2db

type g2db struct {
}

func (G2DB g2db) checkTinyint(val interface{}) error {
	switch v := val.(type) {
	case float64:
		if v < float64(INT8_MIN) || v > float64(INT8_MAX) {
			return ECGO_DATA_OVERFLOW.throw()
		}
	case DmDecimal:
		if v.ToBigInt().Cmp(big.NewInt(int64(INT8_MIN))) < 0 ||
			v.ToBigInt().Cmp(big.NewInt(int64(INT8_MAX))) > 0 {
			return ECGO_DATA_OVERFLOW.throw()
		}
	}
	return nil
}

func (G2DB g2db) checkSmallint(val interface{}) error {
	switch v := val.(type) {
	case float64:
		if v < float64(INT16_MIN) || v > float64(INT16_MAX) {
			return ECGO_DATA_OVERFLOW.throw()
		}
	case DmDecimal:
		if v.ToBigInt().Cmp(big.NewInt(int64(INT16_MIN))) < 0 ||
			v.ToBigInt().Cmp(big.NewInt(int64(INT16_MAX))) > 0 {
			return ECGO_DATA_OVERFLOW.throw()
		}
	}
	return nil
}

func (G2DB g2db) checkInt(val interface{}) error {
	switch v := val.(type) {
	case float64:
		if v < float64(INT32_MIN) || v > float64(INT32_MAX) {
			return ECGO_DATA_OVERFLOW.throw()
		}
	case DmDecimal:
		if v.ToBigInt().Cmp(big.NewInt(int64(INT32_MIN))) < 0 ||
			v.ToBigInt().Cmp(big.NewInt(int64(INT32_MAX))) > 0 {
			return ECGO_DATA_OVERFLOW.throw()
		}
	}
	return nil
}

func (G2DB g2db) checkBigint(val interface{}) error {
	switch v := val.(type) {
	case float64:
		if v < float64(INT64_MIN) || v > float64(INT64_MAX) {
			return ECGO_DATA_OVERFLOW.throw()
		}
	case DmDecimal:
		if v.ToBigInt().Cmp(big.NewInt(INT64_MIN)) < 0 ||
			v.ToBigInt().Cmp(big.NewInt(INT64_MAX)) > 0 {
			return ECGO_DATA_OVERFLOW.throw()
		}
	}
	return nil
}

func (G2DB g2db) checkReal(val interface{}) error {
	switch v := val.(type) {
	case float64:
		if v < float64(FLOAT32_MIN) || v > float64(FLOAT32_MAX) {
			return ECGO_DATA_OVERFLOW.throw()
		}
	case DmDecimal:
		if v.ToBigFloat().Cmp(big.NewFloat(float64(FLOAT32_MIN))) < 0 ||
			v.ToBigFloat().Cmp(big.NewFloat(float64(FLOAT32_MAX))) > 0 {
			return ECGO_DATA_OVERFLOW.throw()
		}
	}
	return nil
}

func (G2DB g2db) fromBool(val bool, param parameter, conn *DmConnection) ([]byte, error) {
	switch param.colType {
	case BOOLEAN, BIT, TINYINT, SMALLINT, INT, BIGINT, REAL, DOUBLE, DECIMAL, CHAR,
		VARCHAR2, VARCHAR, CLOB:
		if val {
			return G2DB.fromInt64(1, param, conn)
		} else {
			return G2DB.fromInt64(0, param, conn)
		}
	case BINARY, VARBINARY, BLOB:
		if val {
			return Dm_build_650.Dm_build_828(byte(1)), nil
		} else {
			return Dm_build_650.Dm_build_828(byte(0)), nil
		}
	}
	return nil, ECGO_DATA_CONVERTION_ERROR.throw()
}

func (G2DB g2db) fromInt64(val int64, param parameter, conn *DmConnection) ([]byte, error) {

	switch param.colType {
	case BOOLEAN, BIT:
		if val == 0 {
			return Dm_build_650.Dm_build_828(byte(0)), nil
		}

		return Dm_build_650.Dm_build_828(byte(1)), nil

	case TINYINT:
		err := G2DB.checkTinyint(float64(val))

		if err != nil {
			return nil, err
		}

		return Dm_build_650.Dm_build_828(byte(val)), nil
	case SMALLINT:
		err := G2DB.checkSmallint(float64(val))

		if err != nil {
			return nil, err
		}

		return Dm_build_650.Dm_build_834(int16(val)), nil
	case INT:
		err := G2DB.checkInt(float64(val))

		if err != nil {
			return nil, err
		}

		return Dm_build_650.Dm_build_837(int32(val)), nil
	case BIGINT:
		err := G2DB.checkBigint(float64(val))

		if err != nil {
			return nil, err
		}

		return Dm_build_650.Dm_build_840(int64(val)), nil
	case REAL:
		err := G2DB.checkReal(float64(val))

		if err != nil {
			return nil, err
		}

		return Dm_build_650.Dm_build_843(float32(val)), nil
	case DOUBLE:
		return Dm_build_650.Dm_build_846(float64(val)), nil
	case DECIMAL:
		d, err := newDecimal(big.NewInt(val), int(param.prec), int(param.scale))
		if err != nil {
			return nil, err
		}
		return d.encodeDecimal()
	case CHAR, VARCHAR2, VARCHAR, CLOB:
		return Dm_build_650.Dm_build_866(strconv.FormatInt(val, 10), conn.getServerEncoding(), conn), nil
	case BINARY, VARBINARY, BLOB:
		return G2DB.ToBinary(val, int(param.prec)), nil
	case DATE, TIME, DATETIME, DATETIME2:
		if err := G2DB.checkInt(float64(val)); err != nil {
			return nil, err
		}
		return toDate(val, param.column, *conn)
	}
	return nil, ECGO_DATA_CONVERTION_ERROR.throw()
}

func (G2DB g2db) ToBinary(x int64, prec int) []byte {
	b := make([]byte, 8)
	b[7] = byte(x)
	b[6] = byte(x >> 8)
	b[5] = byte(x >> 16)
	b[4] = byte(x >> 24)
	b[3] = byte(x >> 32)
	b[2] = byte(x >> 40)
	b[1] = byte(x >> 48)
	b[0] = byte(x >> 56)

	if prec > 0 && prec < len(b) {
		dest := make([]byte, prec)
		copy(dest, b[len(b)-prec:])
		return dest
	}
	return b
}

func (G2DB g2db) fromFloat32(val float32, param parameter, conn *DmConnection) ([]byte, error) {
	switch param.colType {
	case BOOLEAN, BIT:
		if val == 0.0 {
			return Dm_build_650.Dm_build_828(0), nil
		}
		return Dm_build_650.Dm_build_828(1), nil
	case TINYINT:
		if err := G2DB.checkTinyint(float64(val)); err != nil {
			return nil, err
		}
		return Dm_build_650.Dm_build_828(byte(val)), nil
	case SMALLINT:
		if err := G2DB.checkSmallint(float64(val)); err != nil {
			return nil, err
		}
		return Dm_build_650.Dm_build_834(int16(val)), nil
	case INT:
		if err := G2DB.checkInt(float64(val)); err != nil {
			return nil, err
		}
		return Dm_build_650.Dm_build_837(int32(val)), nil
	case BIGINT:
		if err := G2DB.checkBigint(float64(val)); err != nil {
			return nil, err
		}
		return Dm_build_650.Dm_build_840(int64(val)), nil
	case REAL:
		if err := G2DB.checkReal(float64(val)); err != nil {
			return nil, err
		}
		return Dm_build_650.Dm_build_843(val), nil
	case DOUBLE:
		return Dm_build_650.Dm_build_846(float64(val)), nil
	case DECIMAL:
		d, err := newDecimal(big.NewFloat(float64(val)), int(param.prec), int(param.scale))
		if err != nil {
			return nil, err
		}
		return d.encodeDecimal()
	case CHAR, VARCHAR2, VARCHAR, CLOB:
		return Dm_build_650.Dm_build_866(strconv.FormatFloat(float64(val), 'f', -1, 32), conn.getServerEncoding(), conn), nil
	}
	return nil, ECGO_DATA_CONVERTION_ERROR.throw()
}

func (G2DB g2db) fromFloat64(val float64, param parameter, conn *DmConnection) ([]byte, error) {

	switch param.colType {
	case BOOLEAN, BIT:
		if val == 0.0 {
			return Dm_build_650.Dm_build_828(0), nil
		}
		return Dm_build_650.Dm_build_828(1), nil

	case TINYINT:
		err := G2DB.checkTinyint(val)

		if err != nil {
			return nil, err
		}

		return Dm_build_650.Dm_build_828(byte(val)), nil
	case SMALLINT:
		err := G2DB.checkSmallint(val)

		if err != nil {
			return nil, err
		}

		return Dm_build_650.Dm_build_834(int16(val)), nil
	case INT:
		err := G2DB.checkInt(val)

		if err != nil {
			return nil, err
		}

		return Dm_build_650.Dm_build_837(int32(val)), nil
	case BIGINT:
		err := G2DB.checkBigint(val)

		if err != nil {
			return nil, err
		}

		return Dm_build_650.Dm_build_840(int64(val)), nil
	case REAL:
		err := G2DB.checkReal(val)

		if err != nil {
			return nil, err
		}

		return Dm_build_650.Dm_build_843(float32(val)), nil
	case DOUBLE:
		return Dm_build_650.Dm_build_846(float64(val)), nil
	case DECIMAL:
		d, err := newDecimal(big.NewFloat(val), int(param.prec), int(param.scale))
		if err != nil {
			return nil, err
		}
		return d.encodeDecimal()
	case CHAR, VARCHAR2, VARCHAR, CLOB:
		return Dm_build_650.Dm_build_866(strconv.FormatFloat(val, 'f', -1, 64), conn.getServerEncoding(), conn), nil
	}
	return nil, ECGO_DATA_CONVERTION_ERROR.throw()
}

func (G2DB g2db) fromBytes(val []byte, param parameter, conn *DmConnection) (interface{}, error) {
	switch param.colType {
	case CHAR, VARCHAR2, VARCHAR:
		return G2DB.toVarchar(val)
	case CLOB:
		b, err := G2DB.toVarchar(val)
		if err != nil {
			return nil, err
		}
		return G2DB.changeOffRowData(param, b, conn.getServerEncoding())
	case BINARY, VARBINARY:
		return val, nil
	case BLOB:
		return G2DB.bytes2Blob(val, param, conn)
	case ARRAY, CLASS, PLTYPE_RECORD, SARRAY:
		if param.typeDescriptor == nil {
			return nil, ECGO_DATA_CONVERTION_ERROR.throw()
		}
		return TypeDataSV.objBlobToBytes(val, param.typeDescriptor)
	}
	return nil, ECGO_DATA_CONVERTION_ERROR.throw()
}

func (G2DB g2db) toVarchar(bsArr []byte) ([]byte, error) {
	if bsArr == nil || len(bsArr) == 0 {
		return make([]byte, 0), nil
	}

	realLen := len(bsArr) * 2
	bsRet := make([]byte, realLen)
	for i := 0; i < len(bsArr); i++ {
		bsTemp, err := G2DB.toChar(bsArr[i])
		if err != nil {
			return nil, err
		}

		bsRet[i*2] = bsTemp[0]
		bsRet[i*2+1] = bsTemp[1]
	}

	return bsRet, nil
}

func (G2DB g2db) toChar(bt byte) ([]byte, error) {
	bytes := make([]byte, 2)
	var err error

	bytes[0], err = G2DB.getCharByNumVal((bt >> 4) & 0x0F)
	if err != nil {
		return nil, err
	}

	bytes[1], err = G2DB.getCharByNumVal(bt & 0x0F)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (G2DB g2db) getCharByNumVal(val byte) (byte, error) {
	if val >= 0 && val <= 9 {
		return (byte)(val + '0'), nil
	}

	if val >= 0x0a && val <= 0x0F {
		return (byte)(val + 'A' - 0x0a), nil
	}
	return 0, ECGO_INVALID_HEX.throw()
}

func (G2DB g2db) fromString(val string, param parameter, conn *DmConnection) (interface{}, error) {
	switch param.colType {
	case BOOLEAN, BIT:
		ret, err := G2DB.toBool(val)
		if err != nil {
			return nil, err
		}

		if ret {
			return Dm_build_650.Dm_build_828(byte(1)), nil
		} else {
			return Dm_build_650.Dm_build_828(byte(0)), nil
		}

	case TINYINT, SMALLINT, INT, BIGINT:
		f, ok := new(big.Float).SetString(val)
		if !ok {
			return nil, ECGO_DATA_CONVERTION_ERROR.throw()
		}
		if f.Sign() < 0 {
			f.Sub(f, big.NewFloat(0.5))
		} else {
			f.Add(f, big.NewFloat(0.5))
		}
		z, _ := f.Int(nil)
		return G2DB.fromBigInt(z, param, conn)
	case REAL, DOUBLE, DECIMAL:
		f, ok := new(big.Float).SetString(val)
		if ok {
			return G2DB.fromBigFloat(f, param, conn)
		} else {
			return nil, ECGO_DATA_CONVERTION_ERROR.throw()
		}

	case CHAR, VARCHAR2, VARCHAR:
		if param.mask == MASK_BFILE && !isValidBFileStr(val) {
			return nil, ECGO_INVALID_BFILE_STR.throw()
		}
		return Dm_build_650.Dm_build_866(val, conn.getServerEncoding(), conn), nil
	case CLOB:
		return G2DB.string2Clob(val, param, conn)
	case BINARY, VARBINARY:
		return util.StringUtil.HexStringToBytes(val), nil
	case BLOB:
		return G2DB.bytes2Blob(util.StringUtil.HexStringToBytes(val), param, conn)
	case DATE:
		if conn.FormatDate != "" {
			dt, err := parse(val, conn.FormatDate, int(conn.OracleDateLanguage))
			if err != nil {
				return nil, err
			}

			return encode(dt, param.column, int(conn.dmConnector.localTimezone), int(conn.DbTimezone))
		}

		return encodeByString(val, param.column, *conn)
	case TIME:
		if conn.FormatTime != "" {
			dt, err := parse(val, conn.FormatTime, int(conn.OracleDateLanguage))
			if err != nil {
				return nil, err
			}

			return encode(dt, param.column, int(conn.dmConnector.localTimezone), int(conn.DbTimezone))
		}

		return encodeByString(val, param.column, *conn)
	case DATETIME, DATETIME2:
		if conn.FormatTimestamp != "" {
			dt, err := parse(val, conn.FormatTimestamp, int(conn.OracleDateLanguage))
			if err != nil {
				return nil, err
			}

			return encode(dt, param.column, int(conn.dmConnector.localTimezone), int(conn.DbTimezone))
		}

		return encodeByString(val, param.column, *conn)
	case TIME_TZ:
		dt, err := parse(val, conn.FormatTimeTZ, int(conn.OracleDateLanguage))
		if err != nil {
			return nil, err
		}

		if conn.FormatTimeTZ != "" {
			return encode(dt, param.column, int(conn.dmConnector.localTimezone), int(conn.DbTimezone))
		}

		return encodeByString(val, param.column, *conn)
	case DATETIME_TZ, DATETIME2_TZ:
		if conn.FormatTimestampTZ != "" {
			dt, err := parse(val, conn.FormatTimestampTZ, int(conn.OracleDateLanguage))
			if err != nil {
				return nil, err
			}

			return encode(dt, param.column, int(conn.dmConnector.localTimezone), int(conn.DbTimezone))
		}

		return encodeByString(val, param.column, *conn)
	case INTERVAL_DT:
		dt, err := NewDmIntervalDTByString(val)
		if err != nil {
			return nil, err
		}
		return dt.encode(int(param.scale))

	case INTERVAL_YM:
		ym, err := NewDmIntervalYMByString(val)
		if err != nil {
			return nil, err
		}
		return ym.encode(int(param.scale))
	}
	return nil, ECGO_DATA_CONVERTION_ERROR.throw()
}

func (G2DB g2db) toBool(str string) (bool, error) {
	str = strings.TrimSpace(str)
	if util.StringUtil.Equals(str, "0") {
		return false, nil
	} else if util.StringUtil.Equals(str, "1") {
		return true, nil
	}

	return strings.ToLower(str) == "true", nil
}

func (G2DB g2db) fromBigInt(val *big.Int, param parameter, conn *DmConnection) ([]byte, error) {
	var ret []byte
	switch param.colType {
	case BOOLEAN, BIT:
		if val.Sign() == 0 {
			ret = Dm_build_650.Dm_build_828(0)
		} else {
			ret = Dm_build_650.Dm_build_828(1)
		}
	case TINYINT:
		err := G2DB.checkTinyint(float64(val.Int64()))

		if err != nil {
			return nil, err
		}

		ret = Dm_build_650.Dm_build_828(byte(val.Int64()))
	case SMALLINT:
		err := G2DB.checkSmallint(float64(val.Int64()))

		if err != nil {
			return nil, err
		}

		ret = Dm_build_650.Dm_build_834(int16(val.Int64()))
	case INT:
		err := G2DB.checkInt(float64(val.Int64()))

		if err != nil {
			return nil, err
		}

		ret = Dm_build_650.Dm_build_837(int32(val.Int64()))
	case BIGINT:
		err := G2DB.checkBigint(float64(val.Int64()))

		if err != nil {
			return nil, err
		}

		ret = Dm_build_650.Dm_build_840(val.Int64())
	case REAL:
		err := G2DB.checkReal(float64(val.Int64()))

		if err != nil {
			return nil, err
		}

		ret = Dm_build_650.Dm_build_843(float32(val.Int64()))
	case DOUBLE:
		ret = Dm_build_650.Dm_build_846(float64(val.Int64()))
	case DECIMAL, BINARY, VARBINARY, BLOB:
		d, err := newDecimal(val, int(param.prec), int(param.scale))
		if err != nil {
			return nil, err
		}
		ret, err = d.encodeDecimal()
		if err != nil {
			return nil, err
		}
	case CHAR, VARCHAR2, VARCHAR, CLOB:
		ret = Dm_build_650.Dm_build_866(val.String(), conn.getServerEncoding(), conn)
	default:
		return nil, ECGO_DATA_CONVERTION_ERROR.throw()
	}
	return ret, nil
}

func (G2DB g2db) fromBigFloat(val *big.Float, param parameter, conn *DmConnection) ([]byte, error) {
	var ret []byte
	switch param.colType {
	case BOOLEAN, BIT:
		if val.Sign() == 0 {
			ret = Dm_build_650.Dm_build_828(0)
		} else {
			ret = Dm_build_650.Dm_build_828(1)
		}
	case TINYINT:
		f, _ := val.Float64()

		err := G2DB.checkTinyint(f)

		if err != nil {
			return nil, err
		}

		ret = Dm_build_650.Dm_build_828(byte(f))
	case SMALLINT:
		f, _ := val.Float64()

		err := G2DB.checkSmallint(f)

		if err != nil {
			return nil, err
		}

		ret = Dm_build_650.Dm_build_834(int16(f))
	case INT:
		f, _ := val.Float64()

		err := G2DB.checkInt(f)

		if err != nil {
			return nil, err
		}

		ret = Dm_build_650.Dm_build_837(int32(f))
	case BIGINT:
		f, _ := val.Float64()

		err := G2DB.checkBigint(f)

		if err != nil {
			return nil, err
		}

		ret = Dm_build_650.Dm_build_840(int64(f))
	case REAL:
		f, _ := val.Float64()

		err := G2DB.checkReal(f)

		if err != nil {
			return nil, err
		}

		ret = Dm_build_650.Dm_build_843(float32(f))
	case DOUBLE:
		f, _ := val.Float64()
		ret = Dm_build_650.Dm_build_846(f)
	case DECIMAL:
		d, err := newDecimal(val, int(param.prec), int(param.scale))
		if err != nil {
			return nil, err
		}
		ret, err = d.encodeDecimal()
		if err != nil {
			return nil, err
		}
	case CHAR, VARCHAR2, VARCHAR, CLOB:
		ret = Dm_build_650.Dm_build_866(val.Text('f', int(param.scale)), conn.getServerEncoding(), conn)
	default:
		return nil, ECGO_DATA_CONVERTION_ERROR.throw()
	}
	return ret, nil
}

func (G2DB g2db) fromDecimal(val DmDecimal, param parameter, conn *DmConnection) ([]byte, error) {
	var ret []byte
	switch param.colType {
	case BOOLEAN, BIT:
		if val.Sign() == 0 {
			ret = Dm_build_650.Dm_build_828(0)
		} else {
			ret = Dm_build_650.Dm_build_828(1)
		}
	case TINYINT:
		if err := G2DB.checkTinyint(val); err != nil {
			return nil, err
		}
		ret = Dm_build_650.Dm_build_828(byte(val.ToBigInt().Int64()))
	case SMALLINT:
		if err := G2DB.checkSmallint(val); err != nil {
			return nil, err
		}
		ret = Dm_build_650.Dm_build_834(int16(val.ToBigInt().Int64()))
	case INT:
		if err := G2DB.checkInt(val); err != nil {
			return nil, err
		}
		ret = Dm_build_650.Dm_build_837(int32(val.ToBigInt().Int64()))
	case BIGINT:
		if err := G2DB.checkBigint(val); err != nil {
			return nil, err
		}
		ret = Dm_build_650.Dm_build_840(int64(val.ToBigInt().Int64()))
	case REAL:
		if err := G2DB.checkReal(val); err != nil {
			return nil, err
		}
		f, _ := val.ToBigFloat().Float32()
		ret = Dm_build_650.Dm_build_843(f)
	case DOUBLE:
		f, _ := val.ToBigFloat().Float64()
		ret = Dm_build_650.Dm_build_846(f)
	case DECIMAL:
		var err error
		ret, err = val.encodeDecimal()
		if err != nil {
			return nil, err
		}
	case CHAR, VARCHAR2, VARCHAR, CLOB:
		ret = Dm_build_650.Dm_build_866(val.ToBigFloat().Text('f', -1), conn.getServerEncoding(), conn)
	default:
		return nil, ECGO_DATA_CONVERTION_ERROR.throw()
	}
	return ret, nil
}

func (G2DB g2db) fromTime(val time.Time, param parameter, conn *DmConnection) ([]byte, error) {

	switch param.colType {
	case DATE, DATETIME, DATETIME_TZ, TIME, TIME_TZ, DATETIME2, DATETIME2_TZ:
		return encodeByTime(val, param.column, *conn)
	case CHAR, VARCHAR2, VARCHAR, CLOB:
		return Dm_build_650.Dm_build_866(val.Format("2006-01-02 15:04:05.999999999 -07:00"), conn.getServerEncoding(), conn), nil
	}

	return nil, ECGO_DATA_CONVERTION_ERROR.throw()
}

func (G2DB g2db) fromDmIntervalDT(val DmIntervalDT, param parameter, conn *DmConnection) ([]byte, error) {
	switch param.colType {
	case CHAR, VARCHAR2, VARCHAR, CLOB:
		return Dm_build_650.Dm_build_866(val.String(), conn.getServerEncoding(), conn), nil
	case INTERVAL_DT:
		return val.encode(int(param.scale))
	default:
		return nil, ECGO_DATA_CONVERTION_ERROR.throw()
	}
}

func (G2DB g2db) fromDmdbIntervalYM(val DmIntervalYM, param parameter, conn *DmConnection) ([]byte, error) {

	switch param.colType {
	case CHAR, VARCHAR, VARCHAR2, CLOB:
		return Dm_build_650.Dm_build_866(val.String(), conn.getServerEncoding(), conn), nil
	case INTERVAL_YM:
		return val.encode(int(param.scale))
	default:
		return nil, ECGO_DATA_CONVERTION_ERROR.throw()
	}
}

func (G2DB g2db) fromBlob(val DmBlob, param parameter, conn *DmConnection) (interface{}, error) {
	var ret interface{}
	switch param.colType {
	case BINARY, VARBINARY:
		len, err := val.GetLength()
		if err != nil {
			return nil, err
		}
		ret, err = val.getBytes(1, int32(len))
		if err != nil {
			return nil, err
		}
	case BLOB:
		var err error
		ret, err = G2DB.blob2Blob(val, param, conn)
		if err != nil {
			return nil, err
		}
	case ARRAY, CLASS, PLTYPE_RECORD, SARRAY:

	default:
		return nil, ECGO_DATA_CONVERTION_ERROR.throw()
	}
	return ret, nil
}

func (G2DB g2db) fromClob(val DmClob, param parameter, conn *DmConnection) (interface{}, error) {
	var ret interface{}
	switch param.colType {
	case CHAR, VARCHAR, VARCHAR2:
		var len int64
		var s string
		var err error
		len, err = val.GetLength()
		if err != nil {
			return nil, err
		}
		s, err = val.getSubString(1, int32(len))
		if err != nil {
			return nil, err
		}
		ret = []byte(s)
	case CLOB:
		var err error
		ret, err = G2DB.clob2Clob(val, param, conn)
		if err != nil {
			return nil, err
		}
	default:
		return nil, ECGO_DATA_CONVERTION_ERROR.throw()
	}
	return ret, nil
}

func (G2DB g2db) fromReader(val io.Reader, param parameter, conn *DmConnection) (interface{}, error) {
	var ret interface{}
	switch param.colType {
	case CHAR, VARCHAR2, VARCHAR:
		var bytesBuf = new(bytes.Buffer)
		if _, err := bytesBuf.ReadFrom(val); err != nil {
			return nil, err
		}
		return Dm_build_650.Dm_build_866(string(bytesBuf.Bytes()), conn.getServerEncoding(), conn), nil
	case BINARY, VARBINARY:
		var bytesBuf = new(bytes.Buffer)
		if _, err := bytesBuf.ReadFrom(val); err != nil {
			return nil, err
		}
		return util.StringUtil.HexStringToBytes(string(bytesBuf.Bytes())), nil
	case BLOB, CLOB:
		var binder = newOffRowReaderBinder(val, conn.getServerEncoding())
		if binder.offRow {
			ret = binder
		} else {
			ret = binder.readAll()
		}
	default:
		return nil, ECGO_DATA_CONVERTION_ERROR.throw()
	}
	return ret, nil
}

func (G2DB g2db) string2Clob(val string, param parameter, conn *DmConnection) (interface{}, error) {
	return G2DB.changeOffRowData(param, Dm_build_650.Dm_build_866(val, conn.getServerEncoding(), conn), conn.getServerEncoding())
}

func (G2DB g2db) bytes2Blob(val []byte, param parameter, conn *DmConnection) (interface{}, error) {
	return G2DB.changeOffRowData(param, val, conn.getServerEncoding())
}

func (G2DB g2db) clob2Clob(val DmClob, param parameter, conn *DmConnection) (interface{}, error) {
	var clobLen int64
	var err error
	if clobLen, err = val.GetLength(); err != nil {
		return nil, err
	}
	if G2DB.isOffRow(param.colType, clobLen) {
		return newOffRowClobBinder(val, conn.getServerEncoding()), nil
	} else {
		var length int64
		var str string
		if length, err = val.GetLength(); err != nil {
			return nil, err
		}
		if str, err = val.getSubString(1, int32(length)); err != nil {
			return nil, err
		}
		return Dm_build_650.Dm_build_866(str, conn.getServerEncoding(), conn), nil
	}
}

func (G2DB g2db) blob2Blob(val DmBlob, param parameter, conn *DmConnection) (interface{}, error) {
	var clobLen int64
	var err error
	if clobLen, err = val.GetLength(); err != nil {
		return nil, err
	}
	if G2DB.isOffRow(param.colType, clobLen) {
		return newOffRowBlobBinder(val, conn.getServerEncoding()), nil
	} else {
		var length int64
		if length, err = val.GetLength(); err != nil {
			return nil, err
		}
		return val.getBytes(1, int32(length))
	}
}

func (G2DB g2db) changeOffRowData(paramDesc parameter, paramData []byte, encoding string) (interface{}, error) {
	if G2DB.isOffRow(paramDesc.colType, int64(len(paramData))) {
		return newOffRowBytesBinder(paramData, encoding), nil
	} else {
		return paramData, nil
	}
}

func (G2DB g2db) isOffRow(dtype int32, length int64) bool {
	return (dtype == BLOB || dtype == CLOB) && length > Dm_build_124
}

func (G2DB g2db) fromObject(mem interface{}, param parameter, conn *DmConnection) ([]byte, error) {
	switch v := mem.(type) {
	case bool:
		return G2DB.fromBool(v, param, conn)
	case string:
		val, err := G2DB.fromString(v, param, conn)
		if err != nil {
			return nil, err
		}
		return val.([]byte), err
	case byte:
		return G2DB.fromInt64(int64(v), param, conn)
	case int:
		return G2DB.fromInt64(int64(v), param, conn)
	case int16:
		return G2DB.fromInt64(int64(v), param, conn)
	case int32:
		return G2DB.fromInt64(int64(v), param, conn)
	case int64:
		return G2DB.fromInt64(v, param, conn)
	case float32:
		return G2DB.fromFloat64(float64(v), param, conn)
	case float64:
		return G2DB.fromFloat64(v, param, conn)
	case time.Time:
		return G2DB.fromTime(v, param, conn)
	case DmDecimal:
		return G2DB.fromDecimal(v, param, conn)
	case DmIntervalDT:
		return G2DB.fromDmIntervalDT(v, param, conn)
	case DmIntervalYM:
		return G2DB.fromDmdbIntervalYM(v, param, conn)
	case DmBlob:
		length, _ := v.GetLength()
		return v.getBytes(1, int32(length))
	case DmClob:
		length, _ := v.GetLength()
		str, err := v.getSubString(1, int32(length))
		if err != nil {
			return nil, err
		}
		return Dm_build_650.Dm_build_866(str, conn.getServerEncoding(), conn), nil
	default:
		return nil, ECGO_UNSUPPORTED_TYPE.throw()
	}

}

func (G2DB g2db) toInt32(val int32) []byte {
	bytes := make([]byte, 4)
	Dm_build_650.Dm_build_666(bytes, 0, val)
	return bytes
}

func (G2DB g2db) toInt64(val int64) []byte {
	bytes := make([]byte, 8)
	Dm_build_650.Dm_build_671(bytes, 0, val)
	return bytes
}

func (G2DB g2db) toFloat32(val float32) []byte {
	bytes := make([]byte, 4)
	Dm_build_650.Dm_build_676(bytes, 0, val)
	return bytes
}

func (G2DB g2db) toFloat64(val float64) []byte {
	bytes := make([]byte, 8)
	Dm_build_650.Dm_build_681(bytes, 0, val)
	return bytes
}

func (G2DB g2db) toDecimal(val string, prec int, scale int) ([]byte, error) {
	d, err := decodeDecimal([]byte(val), prec, scale)
	if err != nil {
		return nil, err
	}
	return d.encodeDecimal()
}

func (G2DB g2db) fromArray(x *DmArray, param parameter, connection *DmConnection) (interface{}, error) {
	var ret interface{}
	var err error
	switch param.colType {
	case SARRAY:
		ret, err = TypeDataSV.sarrayToBytes(x, param.typeDescriptor)
	case CLASS, ARRAY:
		ret, err = TypeDataSV.arrayToBytes(x, param.typeDescriptor)
	case BLOB:
		ret, err = TypeDataSV.toBytesFromDmArray(x, param.typeDescriptor)
		if err == nil {
			ret, err = G2DB.bytes2Blob(ret.([]byte), param, connection)
		}
	default:
		err = ECGO_DATA_CONVERTION_ERROR.throw()
	}
	return ret, err
}

func (G2DB g2db) fromStruct(x *DmStruct, param parameter, connection *DmConnection) (interface{}, error) {
	var ret interface{}
	var err error
	switch param.colType {
	case CLASS:
		ret, err = TypeDataSV.structToBytes(x, param.typeDescriptor)
	case PLTYPE_RECORD:
		ret, err = TypeDataSV.recordToBytes(x, param.typeDescriptor)
	case BLOB:
		ret, err = TypeDataSV.toBytesFromDmStruct(x, param.typeDescriptor)
		if err == nil {
			ret, err = G2DB.bytes2Blob(ret.([]byte), param, connection)
		}

	default:
		err = ECGO_DATA_CONVERTION_ERROR.throw()
	}
	return ret, err
}

func isValidBFileStr(s string) bool {
	strs := strings.Split(strings.TrimSpace(s), ":")
	if len(strs) != 2 {
		return false
	}
	if len(strs[0]) > Dm_build_52 || len(strs[1]) > Dm_build_53 {
		return false
	}
	return true
}
