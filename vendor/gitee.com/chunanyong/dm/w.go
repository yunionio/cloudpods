/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package dm

import (
	"database/sql/driver"
	"strings"
	"time"
)

const (
	Seconds_1900_1970 = 2209017600

	OFFSET_YEAR = 0

	OFFSET_MONTH = 1

	OFFSET_DAY = 2

	OFFSET_HOUR = 3

	OFFSET_MINUTE = 4

	OFFSET_SECOND = 5

	OFFSET_NANOSECOND = 6

	OFFSET_TIMEZONE = 7

	DT_LEN = 8

	INVALID_VALUE = int(INT32_MIN)

	NANOSECOND_DIGITS = 9

	NANOSECOND_POW = 1000000000
)

type DmTimestamp struct {
	dt                  []int
	dtype               int
	scale               int
	oracleFormatPattern string
	oracleDateLanguage  int

	// Valid为false代表DmArray数据在数据库中为NULL
	Valid bool
}

func newDmTimestampFromDt(dt []int, dtype int, scale int) *DmTimestamp {
	dmts := new(DmTimestamp)
	dmts.Valid = true
	dmts.dt = dt
	dmts.dtype = dtype
	dmts.scale = scale
	return dmts
}

func newDmTimestampFromBytes(bytes []byte, column column, conn *DmConnection) *DmTimestamp {
	dmts := new(DmTimestamp)
	dmts.Valid = true
	dmts.dt = decode(bytes, column.isBdta, column, int(conn.dmConnector.localTimezone), int(conn.DbTimezone))

	if isLocalTimeZone(int(column.colType), int(column.scale)) {
		dmts.scale = getLocalTimeZoneScale(int(column.colType), int(column.scale))
	} else {
		dmts.scale = int(column.scale)
	}

	dmts.dtype = int(column.colType)
	dmts.scale = int(column.scale)
	dmts.oracleDateLanguage = int(conn.OracleDateLanguage)
	switch column.colType {
	case DATE:
		dmts.oracleFormatPattern = conn.FormatDate
	case TIME:
		dmts.oracleFormatPattern = conn.FormatTime
	case TIME_TZ:
		dmts.oracleFormatPattern = conn.FormatTimeTZ
	case DATETIME, DATETIME2:
		dmts.oracleFormatPattern = conn.FormatTimestamp
	case DATETIME_TZ, DATETIME2_TZ:
		dmts.oracleFormatPattern = conn.FormatTimestampTZ
	}
	return dmts
}

func NewDmTimestampFromString(str string) (*DmTimestamp, error) {
	dt := make([]int, DT_LEN)
	dtype, err := toDTFromString(strings.TrimSpace(str), dt)
	if err != nil {
		return nil, err
	}

	if dtype == DATE {
		return newDmTimestampFromDt(dt, dtype, 0), nil
	}
	return newDmTimestampFromDt(dt, dtype, 6), nil
}

func NewDmTimestampFromTime(time time.Time) *DmTimestamp {
	dt := toDTFromTime(time)
	return newDmTimestampFromDt(dt, DATETIME, 6)
}

func (dmTimestamp *DmTimestamp) ToTime() time.Time {
	return toTimeFromDT(dmTimestamp.dt, 0)
}

// 获取年月日时分秒毫秒时区
func (dmTimestamp *DmTimestamp) GetDt() []int {
	return dmTimestamp.dt
}

func (dmTimestamp *DmTimestamp) CompareTo(ts DmTimestamp) int {
	if dmTimestamp.ToTime().Equal(ts.ToTime()) {
		return 0
	} else if dmTimestamp.ToTime().Before(ts.ToTime()) {
		return -1
	} else {
		return 1
	}
}

func (dmTimestamp *DmTimestamp) String() string {
	if dmTimestamp.oracleFormatPattern != "" {
		return dtToStringByOracleFormat(dmTimestamp.dt, dmTimestamp.oracleFormatPattern, int32(dmTimestamp.scale), dmTimestamp.oracleDateLanguage)
	}
	return dtToString(dmTimestamp.dt, dmTimestamp.dtype, dmTimestamp.scale)
}

func (dest *DmTimestamp) Scan(src interface{}) error {
	if dest == nil {
		return ECGO_STORE_IN_NIL_POINTER.throw()
	}
	switch src := src.(type) {
	case nil:
		*dest = *new(DmTimestamp)
		// 将Valid标志置false表示数据库中该列为NULL
		(*dest).Valid = false
		return nil
	case *DmTimestamp:
		*dest = *src
		return nil
	case time.Time:
		ret := NewDmTimestampFromTime(src)
		*dest = *ret
		return nil
	case string:
		ret, err := NewDmTimestampFromString(src)
		if err != nil {
			return err
		}
		*dest = *ret
		return nil
	default:
		return UNSUPPORTED_SCAN.throw()
	}
}

func (dmTimestamp DmTimestamp) Value() (driver.Value, error) {
	if !dmTimestamp.Valid {
		return nil, nil
	}
	return dmTimestamp, nil
}

//func (dmTimestamp *DmTimestamp) toBytes() ([]byte, error) {
//	return encode(dmTimestamp.dt, dmTimestamp.dtype, dmTimestamp.scale, dmTimestamp.dt[OFFSET_TIMEZONE])
//}

/**
 * 获取当前对象的年月日时分秒，如果原来没有decode会先decode;
 */
func (dmTimestamp *DmTimestamp) getDt() []int {
	return dmTimestamp.dt
}

func (dmTimestamp *DmTimestamp) getTime() int64 {
	sec := toTimeFromDT(dmTimestamp.dt, 0).Unix()
	return sec + int64(dmTimestamp.dt[OFFSET_NANOSECOND])
}

func (dmTimestamp *DmTimestamp) setTime(time int64) {
	timeInMillis := (time / 1000) * 1000
	nanos := (int64)((time % 1000) * 1000000)
	if nanos < 0 {
		nanos = 1000000000 + nanos
		timeInMillis = (((time / 1000) - 1) * 1000)
	}
	dmTimestamp.dt = toDTFromUnix(timeInMillis, nanos)
}

func (dmTimestamp *DmTimestamp) setTimezone(tz int) error {
	// DM中合法的时区取值范围为-12:59至+14:00
	if tz <= -13*60 || tz > 14*60 {
		return ECGO_INVALID_DATETIME_FORMAT.throw()
	}
	dmTimestamp.dt[OFFSET_TIMEZONE] = tz
	return nil
}

func (dmTimestamp *DmTimestamp) getNano() int64 {
	return int64(dmTimestamp.dt[OFFSET_NANOSECOND] * 1000)
}

func (dmTimestamp *DmTimestamp) setNano(nano int64) {
	dmTimestamp.dt[OFFSET_NANOSECOND] = (int)(nano / 1000)
}

func (dmTimestamp *DmTimestamp) string() string {
	if dmTimestamp.oracleFormatPattern != "" {
		return dtToStringByOracleFormat(dmTimestamp.dt, dmTimestamp.oracleFormatPattern, int32(dmTimestamp.scale), dmTimestamp.oracleDateLanguage)
	}
	return dtToString(dmTimestamp.dt, dmTimestamp.dtype, dmTimestamp.scale)
}

func (dmTimestamp *DmTimestamp) checkValid() error {
	if !dmTimestamp.Valid {
		return ECGO_IS_NULL.throw()
	}
	return nil
}

/* for gorm v2 */
func (d *DmTimestamp) GormDataType() string {
	return "TIMESTAMP"
}