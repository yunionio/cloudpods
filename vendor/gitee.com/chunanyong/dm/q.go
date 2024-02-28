/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package dm

import (
	"database/sql/driver"
	"math"
	"strconv"
	"strings"

	"gitee.com/chunanyong/dm/util"
)

const (
	LOADPREC_DEFAULT = 2

	LOADPREC_MAX = 9

	SECDPREC_DEFAULT = 6

	SECDPREC_MAX = 6

	QUA_D byte = 3

	QUA_DH byte = 4

	QUA_DHM byte = 5

	QUA_DHMS byte = 6

	QUA_H byte = 7

	QUA_HM byte = 8

	QUA_HMS byte = 9

	QUA_M byte = 10

	QUA_MS byte = 11

	QUA_S byte = 12
)

type DmIntervalDT struct {
	_type byte

	leadScale int

	secScale int

	negative bool

	days int

	hours int

	minutes int

	seconds int

	fraction int

	scaleForSvr int

	Valid bool
}

func (dt *DmIntervalDT) init() {
	dt._type = QUA_D
	dt.leadScale = 2
	dt.secScale = 6
	dt.negative = false
	dt.days = 0
	dt.hours = 0
	dt.minutes = 0
	dt.seconds = 0
	dt.fraction = 0
	dt.scaleForSvr = 0
	dt.Valid = true
}

func newDmIntervalDTByBytes(bytes []byte) *DmIntervalDT {
	dt := new(DmIntervalDT)
	dt.init()

	dt._type = bytes[21]
	dt.scaleForSvr = int(Dm_build_650.Dm_build_752(bytes, 20))
	dt.leadScale = (dt.scaleForSvr >> 4) & 0x0000000F
	dt.secScale = dt.scaleForSvr & 0x0000000F

	switch dt._type {
	case QUA_D:
		dt.days = int(Dm_build_650.Dm_build_752(bytes, 0))
	case QUA_DH:
		dt.days = int(Dm_build_650.Dm_build_752(bytes, 0))
		dt.hours = int(Dm_build_650.Dm_build_752(bytes, 4))
	case QUA_DHM:
		dt.days = int(Dm_build_650.Dm_build_752(bytes, 0))
		dt.hours = int(Dm_build_650.Dm_build_752(bytes, 4))
		dt.minutes = int(Dm_build_650.Dm_build_752(bytes, 8))
	case QUA_DHMS:
		dt.days = int(Dm_build_650.Dm_build_752(bytes, 0))
		dt.hours = int(Dm_build_650.Dm_build_752(bytes, 4))
		dt.minutes = int(Dm_build_650.Dm_build_752(bytes, 8))
		dt.seconds = int(Dm_build_650.Dm_build_752(bytes, 12))
		dt.fraction = int(Dm_build_650.Dm_build_752(bytes, 16))
	case QUA_H:
		dt.hours = int(Dm_build_650.Dm_build_752(bytes, 4))
	case QUA_HM:
		dt.hours = int(Dm_build_650.Dm_build_752(bytes, 4))
		dt.minutes = int(Dm_build_650.Dm_build_752(bytes, 8))
	case QUA_HMS:
		dt.hours = int(Dm_build_650.Dm_build_752(bytes, 4))
		dt.minutes = int(Dm_build_650.Dm_build_752(bytes, 8))
		dt.seconds = int(Dm_build_650.Dm_build_752(bytes, 12))
		dt.fraction = int(Dm_build_650.Dm_build_752(bytes, 16))
	case QUA_M:
		dt.minutes = int(Dm_build_650.Dm_build_752(bytes, 8))
	case QUA_MS:
		dt.minutes = int(Dm_build_650.Dm_build_752(bytes, 8))
		dt.seconds = int(Dm_build_650.Dm_build_752(bytes, 12))
		dt.fraction = int(Dm_build_650.Dm_build_752(bytes, 16))
	case QUA_S:
		dt.seconds = int(Dm_build_650.Dm_build_752(bytes, 12))
		dt.fraction = int(Dm_build_650.Dm_build_752(bytes, 16))
	}
	if dt.days < 0 {
		dt.days = -dt.days
		dt.negative = true
	}
	if dt.hours < 0 {
		dt.hours = -dt.hours
		dt.negative = true
	}
	if dt.minutes < 0 {
		dt.minutes = -dt.minutes
		dt.negative = true
	}
	if dt.seconds < 0 {
		dt.seconds = -dt.seconds
		dt.negative = true
	}
	if dt.fraction < 0 {
		dt.fraction = -dt.fraction
		dt.negative = true
	}

	return dt
}

func NewDmIntervalDTByString(str string) (dt *DmIntervalDT, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = ECGO_INVALID_TIME_INTERVAL.throw()
		}
	}()
	dt = new(DmIntervalDT)
	dt.init()

	if str == "" {
		return nil, ECGO_INVALID_TIME_INTERVAL.throw()
	}

	leadStr := strings.TrimSpace(strings.ToUpper(str))

	if !(strings.Index(leadStr, "INTERVAL ") == 0) {
		return nil, ECGO_INVALID_TIME_INTERVAL.throw()
	}

	leadStr = strings.TrimSpace(leadStr[strings.Index(leadStr, " "):])

	endIndex := 0
	var valueStr string

	if endIndex = strings.Index(leadStr[1:], "'"); leadStr[0] == '\'' && endIndex != -1 {
		endIndex += 1
		valueStr = strings.TrimSpace(leadStr[1:endIndex])
		valueStr = dt.checkSign(valueStr)
		leadStr = strings.TrimSpace(leadStr[endIndex+1:])
	}

	if valueStr == "" {
		leadStr = dt.checkSign(leadStr)
		if endIndex = strings.Index(leadStr[1:], "'"); leadStr[0] != '\'' || endIndex == -1 {
			return nil, ECGO_INVALID_TIME_INTERVAL.throw()
		}
		endIndex += 1
		valueStr = strings.TrimSpace(leadStr[1:endIndex])
		leadStr = strings.TrimSpace(leadStr[endIndex+1:])
	}

	strLeadPrec := ""
	strSecPrec := ""

	leadPrecIndex := 0
	secPrecIndex := 0
	toIndex := 0

	if leadPrecIndex = strings.Index(leadStr, "DAY"); leadPrecIndex != -1 {
		toIndex = strings.Index(leadStr[leadPrecIndex:], "TO")

		if toIndex == -1 {
			strLeadPrec = strings.TrimSpace(leadStr[leadPrecIndex:])
			if err := dt.setDay(valueStr); err != nil {
				return nil, ECGO_INVALID_TIME_INTERVAL.throw()
			}
		} else {
			toIndex += leadPrecIndex
			strLeadPrec = strings.TrimSpace(leadStr[leadPrecIndex:toIndex])

			if strings.Index(leadStr[toIndex:], "HOUR") != -1 {
				if err := dt.setDayToHour(valueStr); err != nil {
					return nil, ECGO_INVALID_TIME_INTERVAL.throw()
				}
			} else if strings.Index(leadStr[toIndex:], "MINUTE") != -1 {
				if err := dt.setDayToMinute(valueStr); err != nil {
					return nil, ECGO_INVALID_TIME_INTERVAL.throw()
				}
			} else if secPrecIndex = strings.Index(leadStr[toIndex:], "SECOND"); secPrecIndex != -1 {
				secPrecIndex += toIndex
				strSecPrec = leadStr[secPrecIndex:]
				if err := dt.setDayToSecond(valueStr); err != nil {
					return nil, ECGO_INVALID_TIME_INTERVAL.throw()
				}
			} else {
				return nil, ECGO_INVALID_TIME_INTERVAL.throw()
			}
		}

		if err := dt.setPrecForSvr(leadStr, strLeadPrec, strSecPrec); err != nil {
			return nil, ECGO_INVALID_TIME_INTERVAL.throw()
		}
		return dt, nil
	}

	if leadPrecIndex = strings.Index(leadStr, "HOUR"); leadPrecIndex != -1 {
		toIndex = strings.Index(leadStr[leadPrecIndex:], "TO")

		if toIndex == -1 {
			toIndex += leadPrecIndex
			strLeadPrec = leadStr[leadPrecIndex:]
			if err := dt.setHour(valueStr); err != nil {
				return nil, ECGO_INVALID_TIME_INTERVAL.throw()
			}
		} else {
			strLeadPrec = leadStr[leadPrecIndex:toIndex]

			if strings.Index(leadStr[toIndex:], "MINUTE") != -1 {
				if err := dt.setHourToMinute(valueStr); err != nil {
					return nil, ECGO_INVALID_TIME_INTERVAL.throw()
				}
			} else if secPrecIndex = strings.Index(leadStr[toIndex:], "SECOND"); secPrecIndex != -1 {
				secPrecIndex += toIndex
				strSecPrec = leadStr[secPrecIndex:]
				if err := dt.setHourToSecond(valueStr); err != nil {
					return nil, ECGO_INVALID_TIME_INTERVAL.throw()
				}
			} else {
				return nil, ECGO_INVALID_TIME_INTERVAL.throw()
			}
		}

		if err := dt.setPrecForSvr(leadStr, strLeadPrec, strSecPrec); err != nil {
			return nil, ECGO_INVALID_TIME_INTERVAL.throw()
		}
		return dt, nil
	}

	if leadPrecIndex = strings.Index(leadStr, "MINUTE"); leadPrecIndex != -1 {
		toIndex = strings.Index(leadStr, "TO")

		if toIndex == -1 {
			toIndex += leadPrecIndex
			strLeadPrec = leadStr[leadPrecIndex:]
			if err := dt.setMinute(valueStr); err != nil {
				return nil, ECGO_INVALID_TIME_INTERVAL.throw()
			}
		} else {
			strLeadPrec = leadStr[leadPrecIndex:toIndex]

			if secPrecIndex = strings.Index(leadStr[toIndex:], "SECOND"); secPrecIndex != -1 {
				strSecPrec = leadStr[secPrecIndex:]
				if err := dt.setMinuteToSecond(valueStr); err != nil {
					return nil, ECGO_INVALID_TIME_INTERVAL.throw()
				}
			} else {
				return nil, ECGO_INVALID_TIME_INTERVAL.throw()
			}
		}

		if err := dt.setPrecForSvr(leadStr, strLeadPrec, strSecPrec); err != nil {
			return nil, ECGO_INVALID_TIME_INTERVAL.throw()
		}
		return dt, nil
	}

	if leadPrecIndex = strings.Index(leadStr, "SECOND"); leadPrecIndex != -1 {
		if err := dt.setSecond(valueStr); err != nil {
			return nil, ECGO_INVALID_TIME_INTERVAL.throw()
		}

		leadStr = strings.TrimSpace(leadStr[leadPrecIndex:])

		colonIndex := strings.Index(leadStr, ",")
		if colonIndex != -1 {
			strLeadPrec = strings.TrimSpace(leadStr[:colonIndex]) + ")"
			strSecPrec = "(" + strings.TrimSpace(leadStr[:colonIndex+1])
		}

		if err := dt.setPrecForSvr(leadStr, strLeadPrec, strSecPrec); err != nil {
			return nil, ECGO_INVALID_TIME_INTERVAL.throw()
		}
		return dt, nil
	}

	return nil, ECGO_INVALID_TIME_INTERVAL.throw()
}

func (dt *DmIntervalDT) GetDay() int {
	return dt.days
}

func (dt *DmIntervalDT) GetHour() int {
	return dt.hours
}

func (dt *DmIntervalDT) GetMinute() int {
	return dt.minutes
}

func (dt *DmIntervalDT) GetSecond() int {
	return dt.seconds
}

func (dt *DmIntervalDT) GetMsec() int {
	return dt.fraction
}

func (dt *DmIntervalDT) GetDTType() byte {
	return dt._type
}

func (dt *DmIntervalDT) String() string {
	if !dt.Valid {
		return ""
	}
	var l, destLen int
	var dStr, hStr, mStr, sStr, nStr string
	interval := "INTERVAL "

	switch dt._type {
	case QUA_D:
		dStr := strconv.FormatInt(int64(float64(dt.days)), 10)
		if dt.negative {
			interval += "-"
		}

		if len(dStr) < dt.leadScale {
			l = len(dStr)
			destLen = dt.leadScale

			for destLen > l {
				dStr = "0" + dStr
				destLen--
			}
		}

		interval += "'" + dStr + "' DAY(" + strconv.FormatInt(int64(dt.leadScale), 10) + ")"
	case QUA_DH:
		dStr = strconv.FormatInt(int64(float64(dt.days)), 10)
		hStr = strconv.FormatInt(int64(float64(dt.hours)), 10)

		if dt.negative {
			interval += "-"
		}

		if len(dStr) < dt.leadScale {
			l = len(dStr)
			destLen = dt.leadScale

			for destLen > l {
				dStr = "0" + dStr
				destLen--
			}
		}

		if len(hStr) < 2 {
			hStr = "0" + hStr
		}

		interval += "'" + dStr + " " + hStr + "' DAY(" + strconv.FormatInt(int64(dt.leadScale), 10) + ") TO HOUR"
	case QUA_DHM:
		dStr = strconv.FormatInt(int64(float64(dt.days)), 10)
		hStr = strconv.FormatInt(int64(float64(dt.hours)), 10)
		mStr = strconv.FormatInt(int64(float64(dt.minutes)), 10)

		if dt.negative {
			interval += "-"
		}

		if len(dStr) < dt.leadScale {
			l = len(dStr)
			destLen = dt.leadScale

			for destLen > l {
				dStr = "0" + dStr
				destLen--
			}
		}
		if len(hStr) < 2 {
			hStr = "0" + hStr
		}
		if len(mStr) < 2 {
			mStr = "0" + mStr
		}
		interval += "'" + dStr + " " + hStr + ":" + mStr + "' DAY(" + strconv.FormatInt(int64(dt.leadScale), 10) + ") TO MINUTE"
	case QUA_DHMS:
		dStr = strconv.FormatInt(int64(float64(dt.days)), 10)
		hStr = strconv.FormatInt(int64(float64(dt.hours)), 10)
		mStr = strconv.FormatInt(int64(float64(dt.minutes)), 10)
		sStr = strconv.FormatInt(int64(float64(dt.seconds)), 10)
		nStr = dt.getMsecString()
		if dt.negative {
			interval += "-"
		}

		if len(dStr) < dt.leadScale {
			l = len(dStr)
			destLen = dt.leadScale

			for destLen > l {
				dStr = "0" + dStr
				destLen--
			}
		}
		if len(hStr) < 2 {
			hStr = "0" + hStr
		}
		if len(mStr) < 2 {
			mStr = "0" + mStr
		}
		if len(sStr) < 2 {
			sStr = "0" + sStr
		}
		interval += "'" + dStr + " " + hStr + ":" + mStr + ":" + sStr
		if nStr != "" {
			interval += "." + nStr
		}

		interval += "' DAY(" + strconv.FormatInt(int64(dt.leadScale), 10) + ") TO SECOND(" + strconv.FormatInt(int64(dt.secScale), 10) + ")"
	case QUA_H:
		hStr = strconv.FormatInt(int64(float64(dt.hours)), 10)
		if dt.negative {
			interval += "-"
		}

		if len(hStr) < dt.leadScale {
			l = len(hStr)
			destLen = dt.leadScale

			for destLen > l {
				hStr = "0" + hStr
				destLen--
			}
		}

		interval += "'" + hStr + "' HOUR(" + strconv.FormatInt(int64(dt.leadScale), 10) + ")"
	case QUA_HM:
		hStr = strconv.FormatInt(int64(float64(dt.hours)), 10)
		mStr = strconv.FormatInt(int64(float64(dt.minutes)), 10)

		if dt.negative {
			interval += "-"
		}

		if len(hStr) < dt.leadScale {
			l = len(hStr)
			destLen = dt.leadScale

			for destLen > l {
				hStr = "0" + hStr
				destLen--
			}
		}
		if len(mStr) < 2 {
			mStr = "0" + mStr
		}

		interval += "'" + hStr + ":" + mStr + "' HOUR(" + strconv.FormatInt(int64(dt.leadScale), 10) + ") TO MINUTE"
	case QUA_HMS:
		nStr = dt.getMsecString()
		hStr = strconv.FormatInt(int64(float64(dt.hours)), 10)
		mStr = strconv.FormatInt(int64(float64(dt.minutes)), 10)
		sStr = strconv.FormatInt(int64(float64(dt.seconds)), 10)

		if dt.negative {
			interval += "-"
		}

		if len(hStr) < dt.leadScale {
			l = len(hStr)
			destLen = dt.leadScale

			for destLen > l {
				hStr = "0" + hStr
				destLen--
			}
		}
		if len(mStr) < 2 {
			mStr = "0" + mStr
		}
		if len(sStr) < 2 {
			sStr = "0" + sStr
		}

		interval += "'" + hStr + ":" + mStr + ":" + sStr
		if nStr != "" {
			interval += "." + nStr
		}

		interval += "' HOUR(" + strconv.FormatInt(int64(dt.leadScale), 10) + ") TO SECOND(" + strconv.FormatInt(int64(dt.secScale), 10) + ")"

	case QUA_M:
		mStr = strconv.FormatInt(int64(float64(dt.minutes)), 10)

		if dt.negative {
			interval += "-"
		}

		if len(mStr) < dt.leadScale {
			l = len(mStr)
			destLen = dt.leadScale

			for destLen > l {
				mStr = "0" + mStr
				destLen--
			}
		}

		interval += "'" + mStr + "' MINUTE(" + strconv.FormatInt(int64(dt.leadScale), 10) + ")"
	case QUA_MS:
		nStr = dt.getMsecString()
		mStr = strconv.FormatInt(int64(float64(dt.minutes)), 10)
		sStr = strconv.FormatInt(int64(float64(dt.seconds)), 10)

		if dt.negative {
			interval += "-"
		}

		if len(mStr) < dt.leadScale {
			l = len(mStr)
			destLen = dt.leadScale

			for destLen > l {
				mStr = "0" + mStr
				destLen--
			}
		}
		if len(sStr) < 2 {
			sStr = "0" + sStr
		}
		interval += "'" + mStr + ":" + sStr
		if nStr != "" {
			interval += "." + nStr
		}

		interval += "' MINUTE(" + strconv.FormatInt(int64(dt.leadScale), 10) + ") TO SECOND(" + strconv.FormatInt(int64(dt.secScale), 10) + ")"
	case QUA_S:
		nStr = dt.getMsecString()
		sStr = strconv.FormatInt(int64(float64(dt.seconds)), 10)

		if dt.negative {
			interval += "-"
		}

		if len(sStr) < dt.leadScale {
			l = len(sStr)
			destLen = dt.leadScale

			for destLen > l {
				sStr = "0" + sStr
				destLen--
			}
		}

		interval += "'" + sStr

		if nStr != "" {
			interval += "." + nStr
		}

		interval += "' SECOND(" + strconv.FormatInt(int64(dt.leadScale), 10) + ", " + strconv.FormatInt(int64(dt.secScale), 10) + ")"

	}

	return interval
}

func (dest *DmIntervalDT) Scan(src interface{}) error {
	if dest == nil {
		return ECGO_STORE_IN_NIL_POINTER.throw()
	}
	switch src := src.(type) {
	case nil:
		*dest = *new(DmIntervalDT)

		(*dest).Valid = false
		return nil
	case *DmIntervalDT:
		*dest = *src
		return nil
	case string:
		ret, err := NewDmIntervalDTByString(src)
		if err != nil {
			return err
		}
		*dest = *ret
		return nil
	default:
		return UNSUPPORTED_SCAN
	}
}

func (dt DmIntervalDT) Value() (driver.Value, error) {
	if !dt.Valid {
		return nil, nil
	}
	return dt, nil
}

func (dt *DmIntervalDT) checkScale(leadScale int) (int, error) {
	switch dt._type {
	case QUA_D:
		if leadScale == -1 {
			leadScale = len(strconv.FormatInt(int64(math.Abs(float64(dt.days))), 10))
		} else if leadScale < len(strconv.FormatInt(int64(math.Abs(float64(dt.days))), 10)) {
			return 0, ECGO_INVALID_TIME_INTERVAL.throw()
		}

	case QUA_DH:
		if leadScale == -1 {
			leadScale = len(strconv.FormatInt(int64(math.Abs(float64(dt.days))), 10))
		} else if leadScale < len(strconv.FormatInt(int64(math.Abs(float64(dt.days))), 10)) {
			return 0, ECGO_INVALID_TIME_INTERVAL.throw()
		}

		if int64(math.Abs(float64((dt.hours)))) > 23 {
			return 0, ECGO_INVALID_TIME_INTERVAL.throw()
		}

	case QUA_DHM:
		if leadScale == -1 {
			leadScale = len(strconv.FormatInt(int64(math.Abs(float64(dt.days))), 10))
		} else if leadScale < len(strconv.FormatInt(int64(math.Abs(float64(dt.days))), 10)) {
			return 0, ECGO_INVALID_TIME_INTERVAL.throw()
		}
		if int64(math.Abs(float64(dt.hours))) > 23 || int64(math.Abs(float64(dt.minutes))) > 59 {
			return 0, ECGO_INVALID_TIME_INTERVAL.throw()
		}

	case QUA_DHMS:
		if leadScale == -1 {
			leadScale = len(strconv.FormatInt(int64(math.Abs(float64(dt.days))), 10))
		} else if leadScale < len(strconv.FormatInt(int64(math.Abs(float64(dt.days))), 10)) {
			return 0, ECGO_INVALID_TIME_INTERVAL.throw()
		}
		if int64(math.Abs(float64(dt.hours))) > 23 || int64(math.Abs(float64(dt.minutes))) > 59 ||
			int64(math.Abs(float64(dt.seconds))) > 59 {
			return 0, ECGO_INVALID_TIME_INTERVAL.throw()
		}

	case QUA_H:
		if leadScale == -1 {
			leadScale = len(strconv.FormatInt(int64(math.Abs(float64(dt.hours))), 10))
		} else if leadScale < len(strconv.FormatInt(int64(math.Abs(float64(dt.hours))), 10)) {
			return 0, ECGO_INVALID_TIME_INTERVAL.throw()
		}

	case QUA_HM:
		if leadScale == -1 {
			leadScale = len(strconv.FormatInt(int64(math.Abs(float64(dt.hours))), 10))
		} else if leadScale < len(strconv.FormatInt(int64(math.Abs(float64(dt.hours))), 10)) {
			return 0, ECGO_INVALID_TIME_INTERVAL.throw()
		}
		if int64(math.Abs(float64(dt.minutes))) > 59 {
			return 0, ECGO_INVALID_TIME_INTERVAL.throw()
		}

	case QUA_HMS:
		if leadScale == -1 {
			leadScale = len(strconv.FormatInt(int64(math.Abs(float64(dt.hours))), 10))
		} else if leadScale < len(strconv.FormatInt(int64(math.Abs(float64(dt.hours))), 10)) {
			return 0, ECGO_INVALID_TIME_INTERVAL.throw()
		}
		if int64(math.Abs(float64(dt.minutes))) > 59 || int64(math.Abs(float64(dt.seconds))) > 59 {
			return 0, ECGO_INVALID_TIME_INTERVAL.throw()
		}

	case QUA_M:
		if leadScale == -1 {
			leadScale = len(strconv.FormatInt(int64(math.Abs(float64(dt.minutes))), 10))
		} else if leadScale < len(strconv.FormatInt(int64(math.Abs(float64(dt.minutes))), 10)) {
			return 0, ECGO_INVALID_TIME_INTERVAL.throw()
		}

	case QUA_MS:
		if leadScale == -1 {
			leadScale = len(strconv.FormatInt(int64(math.Abs(float64(dt.minutes))), 10))
		} else if leadScale < len(strconv.FormatInt(int64(math.Abs(float64(dt.minutes))), 10)) {
			return 0, ECGO_INVALID_TIME_INTERVAL.throw()
		}
		if int64(math.Abs(float64(dt.seconds))) > 59 {
			return 0, ECGO_INVALID_TIME_INTERVAL.throw()
		}
	case QUA_S:
		if leadScale == -1 {
			leadScale = len(strconv.FormatInt(int64(math.Abs(float64(dt.minutes))), 10))
		} else if leadScale < len(strconv.FormatInt(int64(math.Abs(float64(dt.minutes))), 10)) {
			return 0, ECGO_INVALID_TIME_INTERVAL.throw()
		}
	}

	if leadScale > LOADPREC_MAX {
		return 0, ECGO_INVALID_TIME_INTERVAL.throw()
	}
	return leadScale, nil
}

func (dt *DmIntervalDT) parsePrec(leadStr string) (int, error) {
	leftBtId := strings.Index(leadStr, "(")
	rightBtId := strings.Index(leadStr, ")")
	var prec int64 = -1

	if rightBtId != -1 && leftBtId != -1 && rightBtId > leftBtId+1 {
		strPrec := strings.TrimSpace(leadStr[leftBtId+1 : rightBtId])
		var err error
		prec, err = strconv.ParseInt(strPrec, 10, 32)
		if err != nil {
			return -1, err
		}
	}

	return int(prec), nil
}

func (dt *DmIntervalDT) setPrecForSvr(fullStr string, leadScale string, secScale string) error {
	prec, err := dt.parsePrec(leadScale)
	if err != nil {
		return err
	}

	prec, err = dt.checkScale(prec)
	if err != nil {
		return err
	}

	if prec < LOADPREC_DEFAULT {
		dt.leadScale = LOADPREC_DEFAULT
	} else {
		dt.leadScale = prec
	}

	prec, err = dt.parsePrec(secScale)
	if err != nil {
		return err
	}

	if prec >= 0 && prec < SECDPREC_MAX {
		dt.secScale = prec
	} else {
		dt.secScale = SECDPREC_DEFAULT
	}

	dt.scaleForSvr = (int(dt._type) << 8) + (dt.leadScale << 4) + dt.secScale
	return nil
}

func (dt *DmIntervalDT) checkSign(str string) string {

	if str[0] == '-' {
		str = strings.TrimSpace(str[1:])
		dt.negative = true
	} else if str[0] == '+' {
		str = strings.TrimSpace(str[1:])
		dt.negative = false
	}

	return str
}

func (dt *DmIntervalDT) setDay(value string) error {
	list := util.Split(value, " :.")
	if len(list) > 1 {
		return ECGO_INVALID_TIME_INTERVAL.throw()
	}
	dt._type = QUA_D
	i, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return err
	}

	if i < 0 {
		dt.days = int(-i)
		dt.negative = true
	} else {
		dt.days = int(i)
	}
	return nil
}

func (dt *DmIntervalDT) setHour(value string) error {
	list := util.Split(value, " :.")
	if len(list) > 1 {
		return ECGO_INVALID_TIME_INTERVAL.throw()
	}
	dt._type = QUA_H
	i, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return err
	}

	if i < 0 {
		dt.hours = int(-i)
		dt.negative = true
	} else {
		dt.hours = int(i)
	}
	return nil
}

func (dt *DmIntervalDT) setMinute(value string) error {
	list := util.Split(value, " :.")
	if len(list) > 1 {
		return ECGO_INVALID_TIME_INTERVAL.throw()
	}
	dt._type = QUA_M
	i, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return err
	}

	if i < 0 {
		dt.minutes = int(-i)
		dt.negative = true
	} else {
		dt.minutes = int(i)
	}
	return nil
}

func (dt *DmIntervalDT) setSecond(value string) error {
	list := util.Split(value, " :.")
	if len(list) > 2 {
		return ECGO_INVALID_TIME_INTERVAL.throw()
	}
	dt._type = QUA_S
	i, err := strconv.ParseInt(list[0], 10, 32)
	if err != nil {
		return err
	}

	nano := 0
	if len(list) > 1 {
		strNano := "0" + "." + list[1]
		d_v, err := strconv.ParseFloat(strNano, 64)
		if err != nil {
			return err
		}
		nx := math.Pow10(dt.secScale)
		nano = (int)(d_v * nx)
	}

	if i < 0 {
		dt.seconds = int(-i)
	} else {
		dt.seconds = int(i)
	}
	if nano < 0 {
		dt.fraction = -nano
	} else {
		dt.fraction = nano
	}
	if i < 0 || nano < 0 {
		dt.negative = true
	}
	return nil

}

func (dt *DmIntervalDT) setHourToSecond(value string) error {
	list := util.Split(value, " :.")
	if len(list) > 4 {
		return ECGO_INVALID_TIME_INTERVAL.throw()
	}
	dt._type = QUA_HMS

	h, err := strconv.ParseInt(list[0], 10, 32)
	if err != nil {
		return err
	}

	m, err := strconv.ParseInt(list[1], 10, 32)
	if err != nil {
		return err
	}

	s, err := strconv.ParseInt(list[2], 10, 32)
	if err != nil {
		return err
	}
	nano := 0
	if len(list) > 3 {
		strNano := "0" + "." + list[3]
		d_v, err := strconv.ParseFloat(strNano, 64)
		if err != nil {
			return err
		}
		nx := math.Pow10(dt.secScale)
		nano = (int)(d_v * nx)
	}

	if h < 0 {
		dt.hours = int(-h)
	} else {
		dt.hours = int(h)
	}
	if m < 0 {
		dt.minutes = int(-m)
	} else {
		dt.minutes = int(m)
	}
	if s < 0 {
		dt.seconds = int(-s)
	} else {
		dt.seconds = int(s)
	}
	if nano < 0 {
		dt.fraction = -nano
	} else {
		dt.fraction = nano
	}
	if h < 0 || m < 0 || s < 0 || nano < 0 {
		dt.negative = true
	}
	return nil
}

func (dt *DmIntervalDT) setHourToMinute(value string) error {
	value = strings.TrimSpace(value)
	list := util.Split(value, " :.")
	if len(list) > 2 {
		return ECGO_INVALID_TIME_INTERVAL.throw()
	}
	dt._type = QUA_HM

	h, err := strconv.ParseInt(list[0], 10, 32)
	if err != nil {
		return err
	}

	m, err := strconv.ParseInt(list[1], 10, 32)
	if err != nil {
		return err
	}

	if h < 0 {
		dt.hours = int(-h)
	} else {
		dt.hours = int(h)
	}
	if m < 0 {
		dt.minutes = int(-m)
	} else {
		dt.minutes = int(m)
	}
	if h < 0 || m < 0 {
		dt.negative = true
	}
	return nil
}

func (dt *DmIntervalDT) setMinuteToSecond(value string) error {
	list := util.Split(value, " :.")
	if len(list) > 3 {
		return ECGO_INVALID_TIME_INTERVAL.throw()
	}
	dt._type = QUA_MS

	m, err := strconv.ParseInt(list[0], 10, 32)
	if err != nil {
		return err
	}

	s, err := strconv.ParseInt(list[1], 10, 32)
	if err != nil {
		return err
	}

	nano := 0
	if len(list) > 2 {
		strNano := "0" + "." + list[2]
		d_v, err := strconv.ParseFloat(strNano, 64)
		if err != nil {
			return err
		}

		nx := math.Pow10(dt.secScale)
		nano = (int)(d_v * nx)
	}

	if m < 0 {
		dt.minutes = int(-m)
	} else {
		dt.minutes = int(m)
	}
	if s < 0 {
		dt.seconds = int(-s)
	} else {
		dt.seconds = int(s)
	}
	if nano < 0 {
		dt.fraction = -nano
	} else {
		dt.fraction = nano
	}
	if m < 0 || s < 0 || nano < 0 {
		dt.negative = true
	}
	return nil
}

func (dt *DmIntervalDT) setDayToHour(value string) error {
	list := util.Split(value, " :.")
	if len(list) > 2 {
		return ECGO_INVALID_TIME_INTERVAL.throw()
	}
	dt._type = QUA_DH

	d, err := strconv.ParseInt(list[0], 10, 32)
	if err != nil {
		return err
	}

	h, err := strconv.ParseInt(list[1], 10, 32)
	if err != nil {
		return err
	}

	if d < 0 {
		dt.days = int(-d)
	} else {
		dt.days = int(d)
	}
	if h < 0 {
		dt.hours = int(-h)
	} else {
		dt.hours = int(h)
	}
	if d < 0 || h < 0 {
		dt.negative = true
	}
	return nil
}

func (dt *DmIntervalDT) setDayToMinute(value string) error {
	list := util.Split(value, " :.")
	if len(list) > 3 {
		return ECGO_INVALID_TIME_INTERVAL.throw()
	}
	dt._type = QUA_DHM

	d, err := strconv.ParseInt(list[0], 10, 32)
	if err != nil {
		return err
	}

	h, err := strconv.ParseInt(list[1], 10, 32)
	if err != nil {
		return err
	}

	m, err := strconv.ParseInt(list[2], 10, 32)
	if err != nil {
		return err
	}

	if d < 0 {
		dt.days = int(-d)
	} else {
		dt.days = int(d)
	}
	if h < 0 {
		dt.hours = int(-h)
	} else {
		dt.hours = int(h)
	}
	if m < 0 {
		dt.minutes = int(-m)
	} else {
		dt.minutes = int(m)
	}
	if d < 0 || h < 0 || m < 0 {
		dt.negative = true
	}
	return nil
}

func (dt *DmIntervalDT) setDayToSecond(value string) error {
	list := util.Split(value, " :.")
	if len(list) > 5 {
		return ECGO_INVALID_TIME_INTERVAL.throw()
	}
	dt._type = QUA_DHMS

	d, err := strconv.ParseInt(list[0], 10, 32)
	if err != nil {
		return err
	}

	h, err := strconv.ParseInt(list[1], 10, 32)
	if err != nil {
		return err
	}

	m, err := strconv.ParseInt(list[2], 10, 32)
	if err != nil {
		return err
	}

	s, err := strconv.ParseInt(list[3], 10, 32)
	if err != nil {
		return err
	}

	nano := 0
	if len(list) > 4 {
		strNano := "0" + "." + list[4]
		d_v, err := strconv.ParseFloat(strNano, 64)
		if err != nil {
			return err
		}

		nx := math.Pow10(dt.secScale)
		nano = (int)(d_v * nx)
	}

	if d < 0 {
		dt.days = int(-d)
	} else {
		dt.days = int(d)
	}
	if h < 0 {
		dt.hours = int(-h)
	} else {
		dt.hours = int(h)
	}
	if m < 0 {
		dt.minutes = int(-m)
	} else {
		dt.minutes = int(m)
	}
	if s < 0 {
		dt.seconds = int(-s)
	} else {
		dt.seconds = int(s)
	}
	if nano < 0 {
		dt.fraction = -nano
	} else {
		dt.fraction = nano
	}
	if d < 0 || h < 0 || m < 0 || s < 0 || nano < 0 {
		dt.negative = true
	}
	return nil
}

func (dt *DmIntervalDT) getMsecString() string {
	nano := strconv.Itoa(dt.fraction)

	for i := 6 - len(nano); i > 0; i-- {
		nano = "0" + nano
	}

	if len(nano) > dt.secScale {
		nano = nano[:dt.secScale]
	}

	return nano
}

func (dt *DmIntervalDT) encode(scale int) ([]byte, error) {
	if scale == 0 {
		scale = dt.scaleForSvr
	}
	day, hour, minute, second, f := dt.days, dt.hours, dt.minutes, dt.seconds, dt.fraction
	if scale != dt.scaleForSvr {
		convertDT, err := dt.convertTo(scale)
		if err != nil {
			return nil, err
		}
		day, hour, minute, second, f = convertDT.days, convertDT.hours, convertDT.minutes, convertDT.seconds, convertDT.fraction
	} else {
		loadPrec := (scale >> 4) & 0x0000000F
		if _, err := dt.checkScale(loadPrec); err != nil {
			return nil, err
		}
	}

	bytes := make([]byte, 24)
	if dt.negative {
		Dm_build_650.Dm_build_666(bytes, 0, int32(-day))
		Dm_build_650.Dm_build_666(bytes, 4, int32(-hour))
		Dm_build_650.Dm_build_666(bytes, 8, int32(-minute))
		Dm_build_650.Dm_build_666(bytes, 12, int32(-second))
		Dm_build_650.Dm_build_666(bytes, 16, int32(-f))
		Dm_build_650.Dm_build_666(bytes, 20, int32(scale))
	} else {
		Dm_build_650.Dm_build_666(bytes, 0, int32(day))
		Dm_build_650.Dm_build_666(bytes, 4, int32(hour))
		Dm_build_650.Dm_build_666(bytes, 8, int32(minute))
		Dm_build_650.Dm_build_666(bytes, 12, int32(second))
		Dm_build_650.Dm_build_666(bytes, 16, int32(f))
		Dm_build_650.Dm_build_666(bytes, 20, int32(scale))
	}
	return bytes, nil
}

func (dt *DmIntervalDT) convertTo(scale int) (*DmIntervalDT, error) {
	destType := (scale & 0x0000FF00) >> 8
	leadPrec := (scale >> 4) & 0x0000000F
	secScale := scale & 0x0000000F
	dayIndex := 0
	hourIndex := 1
	minuteIndex := 2
	secondIndex := 3
	fractionIndex := 4
	orgDT := make([]int, 5)
	destDT := make([]int, 5)

	switch dt._type {
	case QUA_D:
		orgDT[dayIndex] = dt.days
	case QUA_DH:
		orgDT[dayIndex] = dt.days
		orgDT[hourIndex] = dt.hours
	case QUA_DHM:
		orgDT[dayIndex] = dt.days
		orgDT[hourIndex] = dt.hours
		orgDT[minuteIndex] = dt.minutes
	case QUA_DHMS:
		orgDT[dayIndex] = dt.days
		orgDT[hourIndex] = dt.hours
		orgDT[minuteIndex] = dt.minutes
		orgDT[secondIndex] = dt.seconds
		orgDT[fractionIndex] = dt.fraction
	case QUA_H:
		orgDT[dayIndex] = dt.hours / 24
		orgDT[hourIndex] = dt.hours % 24
	case QUA_HM:
		orgDT[dayIndex] = dt.hours / 24
		orgDT[hourIndex] = dt.hours % 24
		orgDT[minuteIndex] = dt.minutes
	case QUA_HMS:
		orgDT[dayIndex] = dt.hours / 24
		orgDT[hourIndex] = dt.hours % 24
		orgDT[minuteIndex] = dt.minutes
		orgDT[secondIndex] = dt.seconds
		orgDT[fractionIndex] = dt.fraction
	case QUA_M:
		orgDT[dayIndex] = dt.minutes / (24 * 60)
		orgDT[hourIndex] = (dt.minutes % (24 * 60)) / 60
		orgDT[minuteIndex] = (dt.minutes % (24 * 60)) % 60
	case QUA_MS:
		orgDT[dayIndex] = dt.minutes / (24 * 60)
		orgDT[hourIndex] = (dt.minutes % (24 * 60)) / 60
		orgDT[minuteIndex] = (dt.minutes % (24 * 60)) % 60
		orgDT[secondIndex] = dt.seconds
		orgDT[fractionIndex] = dt.fraction
	case QUA_S:
		orgDT[dayIndex] = dt.seconds / (24 * 60 * 60)
		orgDT[hourIndex] = (dt.seconds % (24 * 60 * 60)) / (60 * 60)
		orgDT[minuteIndex] = ((dt.seconds % (24 * 60 * 60)) % (60 * 60)) / 60
		orgDT[secondIndex] = ((dt.seconds % (24 * 60 * 60)) % (60 * 60)) % 60
		orgDT[fractionIndex] = dt.fraction
	}

	switch byte(destType) {
	case QUA_D:
		destDT[dayIndex] = orgDT[dayIndex]
		if orgDT[hourIndex] >= 12 {
			incrementDay(QUA_D, destDT)
		}
		if leadPrec < len(strconv.Itoa(int(math.Abs(float64(destDT[dayIndex]))))) {
			return nil, ECGO_INTERVAL_OVERFLOW.throw()
		}
	case QUA_DH:
		destDT[dayIndex] = orgDT[dayIndex]
		destDT[hourIndex] = orgDT[hourIndex]
		if orgDT[minuteIndex] >= 30 {
			incrementHour(QUA_DH, destDT)
		}
		if leadPrec < len(strconv.Itoa(int(math.Abs(float64(destDT[dayIndex]))))) {
			return nil, ECGO_INTERVAL_OVERFLOW.throw()
		}
	case QUA_DHM:
		destDT[dayIndex] = orgDT[dayIndex]
		destDT[hourIndex] = orgDT[hourIndex]
		destDT[minuteIndex] = orgDT[minuteIndex]
		if orgDT[secondIndex] >= 30 {
			incrementMinute(QUA_DHM, destDT)
		}
		if leadPrec < len(strconv.Itoa(int(math.Abs(float64(destDT[dayIndex]))))) {
			return nil, ECGO_INTERVAL_OVERFLOW.throw()
		}
	case QUA_DHMS:
		destDT[dayIndex] = orgDT[dayIndex]
		destDT[hourIndex] = orgDT[hourIndex]
		destDT[minuteIndex] = orgDT[minuteIndex]
		destDT[secondIndex] = orgDT[secondIndex]
		destDT[fractionIndex] = orgDT[fractionIndex]
		dt.convertMSecond(QUA_DHMS, destDT, secScale)
		if leadPrec < len(strconv.Itoa(int(math.Abs(float64(destDT[dayIndex]))))) {
			return nil, ECGO_INTERVAL_OVERFLOW.throw()
		}
	case QUA_H:
		destDT[hourIndex] = orgDT[dayIndex]*24 + orgDT[hourIndex]
		if orgDT[minuteIndex] >= 30 {
			incrementHour(QUA_H, destDT)
		}
		if leadPrec < len(strconv.Itoa(int(math.Abs(float64(destDT[hourIndex]))))) {
			return nil, ECGO_INTERVAL_OVERFLOW.throw()
		}
	case QUA_HM:
		destDT[hourIndex] = orgDT[dayIndex]*24 + orgDT[hourIndex]
		destDT[minuteIndex] = orgDT[minuteIndex]
		if orgDT[secondIndex] >= 30 {
			incrementMinute(QUA_HM, destDT)
		}
		if leadPrec < len(strconv.Itoa(int(math.Abs(float64(destDT[hourIndex]))))) {
			return nil, ECGO_INTERVAL_OVERFLOW.throw()
		}
	case QUA_HMS:
		destDT[hourIndex] = orgDT[dayIndex]*24 + orgDT[hourIndex]
		destDT[minuteIndex] = orgDT[minuteIndex]
		destDT[secondIndex] = orgDT[secondIndex]
		destDT[fractionIndex] = orgDT[fractionIndex]
		dt.convertMSecond(QUA_HMS, destDT, secScale)
		if leadPrec < len(strconv.Itoa(int(math.Abs(float64(destDT[hourIndex]))))) {
			return nil, ECGO_INTERVAL_OVERFLOW.throw()
		}
	case QUA_M:
		destDT[minuteIndex] = orgDT[dayIndex]*24*60 + orgDT[hourIndex]*60 + orgDT[minuteIndex]
		if orgDT[secondIndex] >= 30 {
			incrementMinute(QUA_M, destDT)
		}
		if leadPrec < len(strconv.Itoa(int(math.Abs(float64(destDT[minuteIndex]))))) {
			return nil, ECGO_INTERVAL_OVERFLOW.throw()
		}
	case QUA_MS:
		destDT[minuteIndex] = orgDT[dayIndex]*24*60 + orgDT[hourIndex]*60 + orgDT[minuteIndex]
		destDT[secondIndex] = orgDT[secondIndex]
		destDT[fractionIndex] = orgDT[fractionIndex]
		dt.convertMSecond(QUA_MS, destDT, secScale)
		if leadPrec < len(strconv.Itoa(int(math.Abs(float64(destDT[minuteIndex]))))) {
			return nil, ECGO_INTERVAL_OVERFLOW.throw()
		}
	case QUA_S:
		destDT[secondIndex] = orgDT[dayIndex]*24*60*60 + orgDT[hourIndex]*60*60 + orgDT[minuteIndex]*60 + orgDT[secondIndex]
		destDT[fractionIndex] = orgDT[fractionIndex]
		dt.convertMSecond(QUA_S, destDT, secScale)
		if leadPrec < len(strconv.Itoa(int(math.Abs(float64(destDT[secondIndex]))))) {
			return nil, ECGO_INTERVAL_OVERFLOW.throw()
		}
	}

	return &DmIntervalDT{
		_type:       byte(destType),
		negative:    dt.negative,
		leadScale:   (scale >> 4) & 0x0000000F,
		secScale:    scale & 0x0000000F,
		scaleForSvr: scale,
		days:        destDT[dayIndex],
		hours:       destDT[hourIndex],
		minutes:     destDT[minuteIndex],
		seconds:     destDT[secondIndex],
		fraction:    destDT[fractionIndex],
		Valid:       true,
	}, nil
}

func (dt DmIntervalDT) convertMSecond(destType byte, destDT []int, destSecScale int) {
	fractionIndex := 4
	orgFraction := destDT[fractionIndex]
	if destSecScale == 0 || destSecScale < dt.secScale {
		n := int(math.Pow(10, 6-float64(destSecScale)-1))
		f := orgFraction / n / 10

		if (orgFraction/n)%10 >= 5 {
			f++
			f = f * n * 10
			if f == 1000000 {
				destDT[fractionIndex] = 0
				incrementSecond(destType, destDT)
				return
			}
		}
		destDT[fractionIndex] = f
	}
}

func incrementDay(destType byte, dt []int) {
	dayIndex := 0
	dt[dayIndex]++
}

func incrementHour(destType byte, dt []int) {
	hourIndex := 1
	dt[hourIndex]++
	if dt[hourIndex] == 24 && destType < QUA_H {
		incrementDay(destType, dt)
		dt[hourIndex] = 0
	}
}

func incrementMinute(destType byte, dt []int) {
	minuteIndex := 2
	dt[minuteIndex]++
	if dt[minuteIndex] == 60 && destType < QUA_M {
		incrementHour(destType, dt)
		dt[minuteIndex] = 0
	}
}

func incrementSecond(destType byte, dt []int) {
	secondIndex := 3
	dt[secondIndex]++
	if dt[secondIndex] == 60 && destType < QUA_S {
		incrementMinute(destType, dt)
		dt[secondIndex] = 0
	}
}

func (dt *DmIntervalDT) checkValid() error {
	if !dt.Valid {
		return ECGO_IS_NULL.throw()
	}
	return nil
}

func (d *DmIntervalDT) GormDataType() string {
	return "INTERVAL DAY TO SECOND"
}
