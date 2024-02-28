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
	QUA_Y  = 0
	QUA_YM = 1
	QUA_MO = 2
)

type DmIntervalYM struct {
	leadScale      int
	isLeadScaleSet bool
	_type          byte
	years          int
	months         int
	scaleForSvr    int

	Valid bool
}

func newDmIntervalYM() *DmIntervalYM {
	return &DmIntervalYM{
		Valid: true,
	}
}

func NewDmIntervalYMByString(str string) (ym *DmIntervalYM, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = ECGO_INVALID_TIME_INTERVAL.throw()
		}
	}()
	ym = newDmIntervalYM()
	ym.isLeadScaleSet = false
	if err = ym.parseIntervYMString(strings.TrimSpace(str)); err != nil {
		return nil, err
	}
	return ym, nil
}

func newDmIntervalYMByBytes(bytes []byte) *DmIntervalYM {
	ym := newDmIntervalYM()

	ym.scaleForSvr = int(Dm_build_650.Dm_build_752(bytes, 8))
	ym.leadScale = (ym.scaleForSvr >> 4) & 0x0000000F
	ym._type = bytes[9]
	switch ym._type {
	case QUA_Y:
		ym.years = int(Dm_build_650.Dm_build_752(bytes, 0))
	case QUA_YM:
		ym.years = int(Dm_build_650.Dm_build_752(bytes, 0))
		ym.months = int(Dm_build_650.Dm_build_752(bytes, 4))
	case QUA_MO:
		ym.months = int(Dm_build_650.Dm_build_752(bytes, 4))
	}
	return ym
}

func (ym *DmIntervalYM) GetYear() int {
	return ym.years
}

func (ym *DmIntervalYM) GetMonth() int {
	return ym.months
}

func (ym *DmIntervalYM) GetYMType() byte {
	return ym._type
}

func (ym *DmIntervalYM) String() string {
	if !ym.Valid {
		return ""
	}
	str := "INTERVAL "
	var year, month string
	var l int
	var destLen int

	switch ym._type {
	case QUA_Y:
		year = strconv.FormatInt(int64(math.Abs(float64(ym.years))), 10)
		if ym.years < 0 {
			str += "-"
		}

		if ym.leadScale > len(year) {
			l = len(year)
			destLen = ym.leadScale

			for destLen > l {
				year = "0" + year
				destLen--
			}
		}

		str += "'" + year + "' YEAR(" + strconv.FormatInt(int64(ym.leadScale), 10) + ")"
	case QUA_YM:
		year = strconv.FormatInt(int64(math.Abs(float64(ym.years))), 10)
		month = strconv.FormatInt(int64(math.Abs(float64(ym.months))), 10)

		if ym.years < 0 || ym.months < 0 {
			str += "-"
		}

		if ym.leadScale > len(year) {
			l = len(year)
			destLen = ym.leadScale

			for destLen > l {
				year = "0" + year
				destLen--
			}
		}

		if len(month) < 2 {
			month = "0" + month
		}

		str += "'" + year + "-" + month + "' YEAR(" + strconv.FormatInt(int64(ym.leadScale), 10) + ") TO MONTH"
	case QUA_MO:

		month = strconv.FormatInt(int64(math.Abs(float64(ym.months))), 10)
		if ym.months < 0 {
			str += "-"
		}

		if ym.leadScale > len(month) {
			l = len(month)
			destLen = ym.leadScale
			for destLen > l {
				month = "0" + month
				destLen--
			}
		}

		str += "'" + month + "' MONTH(" + strconv.FormatInt(int64(ym.leadScale), 10) + ")"
	}
	return str
}

func (dest *DmIntervalYM) Scan(src interface{}) error {
	if dest == nil {
		return ECGO_STORE_IN_NIL_POINTER.throw()
	}
	switch src := src.(type) {
	case nil:
		*dest = *new(DmIntervalYM)

		(*dest).Valid = false
		return nil
	case *DmIntervalYM:
		*dest = *src
		return nil
	case string:
		ret, err := NewDmIntervalYMByString(src)
		if err != nil {
			return err
		}
		*dest = *ret
		return nil
	default:
		return UNSUPPORTED_SCAN
	}
}

func (ym DmIntervalYM) Value() (driver.Value, error) {
	if !ym.Valid {
		return nil, nil
	}
	return ym, nil
}

func (ym *DmIntervalYM) parseIntervYMString(str string) error {
	str = strings.ToUpper(str)
	ret := strings.Split(str, " ")
	l := len(ret)
	if l < 3 || !util.StringUtil.EqualsIgnoreCase(ret[0], "INTERVAL") || !(strings.HasPrefix(ret[2], "YEAR") || strings.HasPrefix(ret[2], "MONTH")) {
		return ECGO_INVALID_TIME_INTERVAL.throw()
	}
	ym._type = QUA_YM
	yearId := strings.Index(str, "YEAR")
	monthId := strings.Index(str, "MONTH")
	toId := strings.Index(str, "TO")
	var err error
	if toId == -1 {
		if yearId != -1 && monthId == -1 {
			ym._type = QUA_Y
			ym.leadScale, err = ym.getLeadPrec(str, yearId)
			if err != nil {
				return err
			}
		} else if monthId != -1 && yearId == -1 {
			ym._type = QUA_MO
			ym.leadScale, err = ym.getLeadPrec(str, monthId)
			if err != nil {
				return err
			}
		} else {
			return ECGO_INVALID_TIME_INTERVAL.throw()
		}
	} else {
		if yearId == -1 || monthId == -1 {
			return ECGO_INVALID_TIME_INTERVAL.throw()
		}
		ym._type = QUA_YM
		ym.leadScale, err = ym.getLeadPrec(str, yearId)
		if err != nil {
			return err
		}
	}

	ym.scaleForSvr = (int(ym._type) << 8) + (ym.leadScale << 4)
	timeVals, err := ym.getTimeValue(ret[1], int(ym._type))
	if err != nil {
		return err
	}
	ym.years = timeVals[0]
	ym.months = timeVals[1]
	return ym.checkScale(ym.leadScale)
}

func (ym *DmIntervalYM) getLeadPrec(str string, startIndex int) (int, error) {
	if ym.isLeadScaleSet {
		return ym.leadScale, nil
	}

	leftBtId := strings.Index(str[startIndex:], "(")
	rightBtId := strings.Index(str[startIndex:], ")")
	leadPrec := 0

	if rightBtId == -1 && leftBtId == -1 {
		leftBtId += startIndex
		rightBtId += startIndex
		l := strings.Index(str, "'")
		var r int
		var dataStr string
		if l != -1 {
			r = strings.Index(str[l+1:], "'")
			if r != -1 {
				r += l + 1
			}
		} else {
			r = -1
		}

		if r != -1 {
			dataStr = strings.TrimSpace(str[l+1 : r])
		} else {
			dataStr = ""
		}

		if dataStr != "" {
			sign := dataStr[0]
			if sign == '+' || sign == '-' {
				dataStr = strings.TrimSpace(dataStr[1:])
			}
			end := strings.Index(dataStr, "-")

			if end != -1 {
				dataStr = dataStr[:end]
			}

			leadPrec = len(dataStr)
		} else {
			leadPrec = 2
		}
	} else if rightBtId != -1 && leftBtId != -1 && rightBtId > leftBtId+1 {
		leftBtId += startIndex
		rightBtId += startIndex
		strPrec := strings.TrimSpace(str[leftBtId+1 : rightBtId])
		temp, err := strconv.ParseInt(strPrec, 10, 32)
		if err != nil {
			return 0, err
		}

		leadPrec = int(temp)
	} else {
		return 0, ECGO_INVALID_TIME_INTERVAL.throw()
	}

	return leadPrec, nil
}

func (ym *DmIntervalYM) checkScale(prec int) error {
	switch ym._type {
	case QUA_Y:
		if prec < len(strconv.FormatInt(int64(math.Abs(float64(ym.years))), 10)) {
			return ECGO_INVALID_TIME_INTERVAL.throw()
		}
	case QUA_YM:
		if prec < len(strconv.FormatInt(int64(math.Abs(float64(ym.years))), 10)) {
			return ECGO_INVALID_TIME_INTERVAL.throw()
		}

		if int64(math.Abs(float64(ym.months))) > 11 {
			return ECGO_INVALID_TIME_INTERVAL.throw()
		}

	case QUA_MO:
		if prec < len(strconv.FormatInt(int64(math.Abs(float64(ym.months))), 10)) {
			return ECGO_INVALID_TIME_INTERVAL.throw()
		}
	}
	return nil
}

func (ym *DmIntervalYM) getTimeValue(subStr string, _type int) ([]int, error) {
	hasQuate := false
	if subStr[0] == '\'' && subStr[len(subStr)-1] == '\'' {
		hasQuate = true
		subStr = strings.TrimSpace(subStr[1 : len(subStr)-1])
	}

	negative := false
	if strings.Index(subStr, "-") == 0 {
		negative = true
		subStr = subStr[1:]
	} else if strings.Index(subStr, "+") == 0 {
		negative = false
		subStr = subStr[1:]
	}

	if subStr[0] == '\'' && subStr[len(subStr)-1] == '\'' {
		hasQuate = true
		subStr = strings.TrimSpace(subStr[1 : len(subStr)-1])
	}

	if !hasQuate {
		return nil, ECGO_INVALID_TIME_INTERVAL.throw()
	}

	lastSignIndex := strings.LastIndex(subStr, "-")

	list := make([]string, 2)
	if lastSignIndex == -1 || lastSignIndex == 0 {
		list[0] = subStr
		list[1] = ""
	} else {
		list[0] = subStr[0:lastSignIndex]
		list[1] = subStr[lastSignIndex+1:]
	}

	var yearVal, monthVal int64
	var err error
	if ym._type == QUA_YM {
		yearVal, err = strconv.ParseInt(list[0], 10, 32)
		if err != nil {
			return nil, err
		}

		if util.StringUtil.EqualsIgnoreCase(list[1], "") {
			monthVal = 0
		} else {
			monthVal, err = strconv.ParseInt(list[1], 10, 32)
			if err != nil {
				return nil, err
			}
		}

		if negative {
			yearVal *= -1
			monthVal *= -1
		}

		if yearVal > int64(math.Pow10(ym.leadScale))-1 || yearVal < 1-int64(math.Pow10(ym.leadScale)) {
			return nil, ECGO_INVALID_TIME_INTERVAL.throw()
		}
	} else if ym._type == QUA_Y {
		yearVal, err = strconv.ParseInt(list[0], 10, 32)
		if err != nil {
			return nil, err
		}
		monthVal = 0

		if negative {
			yearVal *= -1
		}

		if yearVal > int64(math.Pow10(ym.leadScale))-1 || yearVal < 1-int64(math.Pow10(ym.leadScale)) {
			return nil, ECGO_INVALID_TIME_INTERVAL.throw()
		}
	} else {
		yearVal = 0
		monthVal, err = strconv.ParseInt(list[0], 10, 32)
		if err != nil {
			return nil, err
		}
		if negative {
			monthVal *= -1
		}

		if monthVal > int64(math.Pow10(ym.leadScale))-1 || monthVal < 1-int64(math.Pow10(ym.leadScale)) {
			return nil, ECGO_INVALID_TIME_INTERVAL.throw()
		}
	}

	ret := make([]int, 2)
	ret[0] = int(yearVal)
	ret[1] = int(monthVal)

	return ret, nil
}

func (ym *DmIntervalYM) encode(scale int) ([]byte, error) {
	if scale == 0 {
		scale = ym.scaleForSvr
	}
	year, month := ym.years, ym.months
	if err := ym.checkScale(ym.leadScale); err != nil {
		return nil, err
	}
	if scale != ym.scaleForSvr {
		convertYM, err := ym.convertTo(scale)
		if err != nil {
			return nil, err
		}
		year = convertYM.years
		month = convertYM.months
	} else {
		if err := ym.checkScale(ym.leadScale); err != nil {
			return nil, err
		}
	}

	bytes := make([]byte, 12)
	Dm_build_650.Dm_build_666(bytes, 0, int32(year))
	Dm_build_650.Dm_build_666(bytes, 4, int32(month))
	Dm_build_650.Dm_build_666(bytes, 8, int32(scale))
	return bytes, nil
}

func (ym *DmIntervalYM) convertTo(scale int) (*DmIntervalYM, error) {
	destType := (scale & 0x0000FF00) >> 8
	leadPrec := (scale >> 4) & 0x0000000F
	totalMonths := ym.years*12 + ym.months
	year := 0
	month := 0
	switch destType {
	case QUA_Y:
		year = totalMonths / 12

		if totalMonths%12 >= 6 {
			year++
		} else if totalMonths%12 <= -6 {
			year--
		}
		if leadPrec < len(strconv.Itoa(int(math.Abs(float64(year))))) {
			return nil, ECGO_INVALID_TIME_INTERVAL.throw()
		}
	case QUA_YM:
		year = totalMonths / 12
		month = totalMonths % 12
		if leadPrec < len(strconv.Itoa(int(math.Abs(float64(year))))) {
			return nil, ECGO_INVALID_TIME_INTERVAL.throw()
		}
	case QUA_MO:
		month = totalMonths
		if leadPrec < len(strconv.Itoa(int(math.Abs(float64(month))))) {
			return nil, ECGO_INVALID_TIME_INTERVAL.throw()
		}
	}
	return &DmIntervalYM{
		_type:       byte(destType),
		years:       year,
		months:      month,
		scaleForSvr: scale,
		leadScale:   (scale >> 4) & 0x0000000F,
		Valid:       true,
	}, nil
}

func (ym *DmIntervalYM) checkValid() error {
	if !ym.Valid {
		return ECGO_IS_NULL.throw()
	}
	return nil
}

func (d *DmIntervalYM) GormDataType() string {
	return "INTERVAL YEAR TO MONTH"
}
