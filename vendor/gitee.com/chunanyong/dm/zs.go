/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package dm

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"gitee.com/chunanyong/dm/util"
)

type oracleDateFormat struct {
	PM                bool
	TZNegative        bool
	pattern           string
	language          int
	scale             int32
	FormatElementList []interface{}
	YearElement       yearElement
	MonthElement      monthElement
	MonElement        monElement
	MMElement         mmElement
	DDElement         ddElement
	HH24Element       hh24Element
	HH12Element       hh12Element
	MIElement         miElement
	SSElement         ssElement
	FElement          fElement
	TZHElement        tzhElement
	TZMElement        tzmElement
	AMElement         amElement
}

type element interface {
	/**
	 * 从字符串中解析出对应的值,
	 * @param str 完整的字符串
	 * @param offset 当前偏移
	 * @return 解析后的offset
	 */
	parse(str string, offset int, dt []int) (int, error)

	/**
	 * 将时间值value格式化成字符串
	 */
	format(dt []int) string
}

type yearElement struct {
	OracleDateFormat *oracleDateFormat
	len              int
}

func (YearElement yearElement) parse(str string, offset int, dt []int) (int, error) {
	strLen := 0
	for i := offset; i < offset+YearElement.len && i < len(str); i++ {
		if !unicode.IsLetter(rune(str[i])) && !unicode.IsDigit(rune(str[i])) {
			break
		}
		strLen++
	}
	str = str[offset : offset+strLen]
	if YearElement.len < 4 {
		today := strconv.FormatInt(int64(dt[OFFSET_YEAR]), 10)
		i, err := strconv.ParseInt(today[:4-YearElement.len]+str, 10, 32)
		if err != nil {
			return 0, err
		}
		dt[OFFSET_YEAR] = int(i)
	} else {
		i, err := strconv.ParseInt(str, 10, 32)
		if err != nil {
			return 0, err
		}
		dt[OFFSET_YEAR] = int(i)
	}

	return offset + strLen, nil
}

func (YearElement yearElement) format(dt []int) string {
	return YearElement.OracleDateFormat.formatInt(dt[OFFSET_YEAR], YearElement.len)
}

type monthElement struct {
	OracleDateFormat *oracleDateFormat
	upperCase        bool
	lowerCase        bool
}

var monthNameList = []string{"", "January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}

func (MonthElement monthElement) parse(str string, offset int, dt []int) (int, error) {

	if MonthElement.OracleDateFormat.language == LANGUAGE_CN {
		index := strings.IndexRune(str[offset:], '月')
		if index == -1 {
			return -1, ECGO_INVALID_DATETIME_FORMAT.throw()
		}
		index += offset

		mon, err := strconv.ParseInt(str[offset:index], 10, 32)
		if err != nil {
			return -1, ECGO_INVALID_DATETIME_FORMAT.throw()
		}

		if mon > 12 || mon < 1 {
			return -1, ECGO_INVALID_DATETIME_VALUE.throw()
		}
		dt[OFFSET_MONTH] = int(mon)
		return index + utf8.RuneLen('月'), nil
	} else {
		str = str[offset:]
		mon := 0
		for i := 1; i < len(monthNameList); i++ {
			if util.StringUtil.StartWithIgnoreCase(str, monthNameList[i]) {
				mon = i
				break
			}
		}
		if mon == 0 {
			return -1, ECGO_INVALID_DATETIME_FORMAT.throw()
		}
		dt[OFFSET_MONTH] = mon
		return offset + len(monthNameList[mon]), nil
	}
}

func (MonthElement monthElement) format(dt []int) string {
	value := dt[OFFSET_MONTH]

	if MonthElement.OracleDateFormat.language == LANGUAGE_CN {
		return strconv.FormatInt(int64(value), 10) + "月"
	}

	if MonthElement.upperCase {
		return strings.ToUpper(monthNameList[value])
	} else if MonthElement.lowerCase {
		return strings.ToLower(monthNameList[value])
	} else {
		return monthNameList[value]
	}

}

type monElement struct {
	OracleDateFormat *oracleDateFormat
	upperCase        bool
	lowerCase        bool
}

var monNameList []string = []string{"", "Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

func (MonElement monElement) parse(str string, offset int, dt []int) (int, error) {

	if MonElement.OracleDateFormat.language == LANGUAGE_CN {
		index := strings.IndexRune(str[offset:], '月') + offset
		if index == -1+offset {
			return -1, ECGO_INVALID_DATETIME_FORMAT.throw()
		}

		mon, err := strconv.ParseInt(str[offset:index], 10, 32)
		if err != nil {
			return -1, err
		}

		if mon > 12 || mon < 1 {
			return -1, ECGO_INVALID_DATETIME_VALUE.throw()
		}
		dt[OFFSET_MONTH] = int(mon)
		return index + utf8.RuneLen('月'), nil
	} else {
		str = str[offset : offset+3]
		mon := 0
		for i := 1; i < len(monNameList); i++ {
			if util.StringUtil.EqualsIgnoreCase(str, monNameList[i]) {
				mon = i
				break
			}
		}
		if mon == 0 {
			return -1, ECGO_INVALID_DATETIME_FORMAT.throw()
		}
		dt[OFFSET_MONTH] = mon
		return offset + 3, nil
	}

}

func (MonElement monElement) format(dt []int) string {
	value := dt[OFFSET_MONTH]
	language := int(0)
	if language == LANGUAGE_CN {
		return strconv.FormatInt(int64(value), 10) + "月"
	}

	if MonElement.upperCase {
		return strings.ToUpper(monNameList[value])
	} else if MonElement.lowerCase {
		return strings.ToLower(monNameList[value])
	} else {
		return monNameList[value]
	}
}

type mmElement struct {
	OracleDateFormat *oracleDateFormat
}

func (MMElement mmElement) parse(str string, offset int, dt []int) (int, error) {
	strLen := 0
	for i := offset; i < offset+2 && i < len(str); i++ {
		if !unicode.IsLetter(rune(str[i])) && !unicode.IsDigit(rune(str[i])) {
			break
		}
		strLen++
	}
	str = str[offset : offset+strLen]
	month, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return -1, ECGO_INVALID_DATETIME_FORMAT.throw()
	}

	if month > 12 || month < 1 {
		return -1, ECGO_INVALID_DATETIME_VALUE.throw()
	}
	dt[OFFSET_MONTH] = int(month)
	return offset + strLen, nil
}

func (MMElement mmElement) format(dt []int) string {
	return MMElement.OracleDateFormat.formatInt(dt[OFFSET_MONTH], 2)
}

type ddElement struct {
	OracleDateFormat *oracleDateFormat
}

func (DDElement ddElement) parse(str string, offset int, dt []int) (int, error) {
	strLen := 0
	for i := offset; i < offset+2 && i < len(str); i++ {
		if !unicode.IsLetter(rune(str[i])) && !unicode.IsDigit(rune(str[i])) {
			break
		}
		strLen++
	}
	str = str[offset : offset+strLen]
	day, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return -1, err
	}

	if day > 31 || day < 1 {
		return -1, ECGO_INVALID_DATETIME_VALUE.throw()
	}
	dt[OFFSET_DAY] = int(day)
	return offset + strLen, nil
}

func (DDElement ddElement) format(dt []int) string {
	return DDElement.OracleDateFormat.formatInt(dt[OFFSET_DAY], 2)
}

type hh24Element struct {
	OracleDateFormat *oracleDateFormat
}

func (HH24Element hh24Element) parse(str string, offset int, dt []int) (int, error) {
	strLen := 0
	for i := offset; i < offset+2 && i < len(str); i++ {
		if !unicode.IsLetter(rune(str[i])) && !unicode.IsDigit(rune(str[i])) {
			break
		}
		strLen++
	}
	str = str[offset : offset+strLen]
	hour, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return -1, err
	}

	if hour > 23 || hour < 0 {
		return -1, ECGO_INVALID_DATETIME_VALUE.throw()
	}
	dt[OFFSET_HOUR] = int(hour) // 0-23
	return offset + strLen, nil
}

func (HH24Element hh24Element) format(dt []int) string {
	return HH24Element.OracleDateFormat.formatInt(dt[OFFSET_HOUR], 2) // 0-23
}

type hh12Element struct {
	OracleDateFormat *oracleDateFormat
}

func (HH12Element hh12Element) parse(str string, offset int, dt []int) (int, error) {
	strLen := 0
	for i := offset; i < offset+2 && i < len(str); i++ {
		if !unicode.IsLetter(rune(str[i])) && !unicode.IsDigit(rune(str[i])) {
			break
		}
		strLen++
	}
	str = str[offset : offset+strLen]
	hour, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return -1, err
	}

	if hour > 12 || hour < 1 {
		return -1, ECGO_INVALID_DATETIME_VALUE.throw()
	}
	dt[OFFSET_HOUR] = int(hour)
	return offset + strLen, nil
}

func (HH12Element hh12Element) format(dt []int) string {
	var ret string
	value := dt[OFFSET_HOUR]
	if value > 12 || value == 0 {
		ret = HH12Element.OracleDateFormat.formatInt(int(math.Abs(float64(value-12))), 2) // 1-12
	} else {
		ret = HH12Element.OracleDateFormat.formatInt(value, 2)
	}
	return ret
}

type miElement struct {
	OracleDateFormat *oracleDateFormat
}

func (MIElement miElement) parse(str string, offset int, dt []int) (int, error) {
	strLen := 0
	for i := offset; i < offset+2 && i < len(str); i++ {
		if !unicode.IsLetter(rune(str[i])) && !unicode.IsDigit(rune(str[i])) {
			break
		}
		strLen++
	}
	str = str[offset : offset+strLen]
	minute, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return -1, err
	}

	if minute > 59 || minute < 0 {
		return -1, ECGO_INVALID_DATETIME_VALUE.throw()
	}
	dt[OFFSET_MINUTE] = int(minute) // 0-59
	return offset + strLen, nil
}

func (MIElement miElement) format(dt []int) string {
	return MIElement.OracleDateFormat.formatInt(dt[OFFSET_MINUTE], 2) // 0-59
}

type ssElement struct {
	OracleDateFormat *oracleDateFormat
}

func (SSElement ssElement) parse(str string, offset int, dt []int) (int, error) {
	strLen := 0
	for i := offset; i < offset+2 && i < len(str); i++ {
		if !unicode.IsLetter(rune(str[i])) && !unicode.IsDigit(rune(str[i])) {
			break
		}
		strLen++
	}
	str = str[offset : offset+strLen]
	second, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return -1, err
	}

	if second > 59 || second < 0 {
		return -1, ECGO_INVALID_DATETIME_VALUE.throw()
	}
	dt[OFFSET_SECOND] = int(second) // 0-59
	return offset + strLen, nil
}

func (SSElement ssElement) format(dt []int) string {
	return SSElement.OracleDateFormat.formatInt(dt[OFFSET_SECOND], 2) // 0-59
}

type fElement struct {
	OracleDateFormat *oracleDateFormat
	len              int
}

func (FElement fElement) parse(str string, offset int, dt []int) (int, error) {
	strLen := 0
	maxLen := 0
	if FElement.len > 0 {
		maxLen = FElement.len
	} else {
		maxLen = NANOSECOND_DIGITS
	}
	for i := offset; i < offset+maxLen && i < len(str); i++ {
		if !unicode.IsLetter(rune(str[i])) && !unicode.IsDigit(rune(str[i])) {
			break
		}
		strLen++
	}
	str = str[offset : offset+strLen]
	ms, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return -1, err
	}

	if strLen < NANOSECOND_DIGITS {
		ms *= int64(math.Pow10(NANOSECOND_DIGITS - strLen))
	} else {
		ms /= int64(math.Pow10(strLen - NANOSECOND_DIGITS))
	}

	dt[OFFSET_NANOSECOND] = int(ms)
	return offset + strLen, nil
}

func (FElement fElement) format(dt []int) string {
	msgLen := 0
	if FElement.len > 0 {
		msgLen = FElement.len
	} else {
		msgLen = int(FElement.OracleDateFormat.scale)
	}
	return FElement.OracleDateFormat.formatMilliSecond(dt[OFFSET_NANOSECOND], msgLen)
}

type tzhElement struct {
	OracleDateFormat *oracleDateFormat
}

func (TZHElement tzhElement) parse(str string, offset int, dt []int) (int, error) {
	if str[offset] == '+' {
		offset += 1
	} else if str[offset] == '-' {
		offset += 1
		TZHElement.OracleDateFormat.TZNegative = true
	}

	strLen := 0
	for i := offset; i < offset+2 && i < len(str); i++ {
		if !unicode.IsLetter(rune(str[i])) && !unicode.IsDigit(rune(str[i])) {
			break
		}
		strLen++
	}
	str = str[offset : offset+strLen]

	tzh, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return -1, err
	}

	if tzh > 23 || tzh < 0 {
		return -1, ECGO_INVALID_DATETIME_VALUE.throw()
	}

	tzh *= 60
	if dt[OFFSET_TIMEZONE] == int(INVALID_VALUE) {
		dt[OFFSET_TIMEZONE] = int(tzh)
	} else {
		dt[OFFSET_TIMEZONE] += int(tzh)
	}

	return offset + strLen, nil
}

func (TZHElement tzhElement) format(dt []int) string {
	var value int
	if dt[OFFSET_TIMEZONE] != int(INVALID_VALUE) {
		value = int(math.Abs(float64(dt[OFFSET_TIMEZONE]))) / 60
	} else {
		value = 0
	}

	return TZHElement.OracleDateFormat.formatInt(value, 2)
}

type tzmElement struct {
	OracleDateFormat *oracleDateFormat
}

func (TZMElement tzmElement) parse(str string, offset int, dt []int) (int, error) {
	if str[offset] == '+' {
		offset += 1
	} else if str[offset] == '-' {
		offset += 1
		TZMElement.OracleDateFormat.TZNegative = true
	}

	strLen := 0
	for i := offset; i < offset+2 && i < len(str); i++ {
		if !unicode.IsLetter(rune(str[i])) && !unicode.IsDigit(rune(str[i])) {
			break
		}
		strLen++
	}
	str = str[offset : offset+strLen]

	tzm, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return -1, err
	}
	if tzm > 59 || tzm < 0 {
		return -1, ECGO_INVALID_DATETIME_VALUE.throw()
	}

	if dt[OFFSET_TIMEZONE] == INVALID_VALUE {
		dt[OFFSET_TIMEZONE] = int(tzm)
	} else {
		dt[OFFSET_TIMEZONE] += int(tzm)
	}
	return offset + strLen, nil
}

func (TZMElement tzmElement) format(dt []int) string {
	var value int
	if dt[OFFSET_TIMEZONE] != int(INVALID_VALUE) {
		value = int(math.Abs(float64(dt[OFFSET_TIMEZONE]))) % 60
	} else {
		value = 0
	}

	return TZMElement.OracleDateFormat.formatInt(value, 2)
}

type amElement struct {
	OracleDateFormat *oracleDateFormat
}

func (AMElement amElement) parse(str string, offset int, dt []int) (int, error) {
	runeStr := ([]rune(str))[offset : offset+2]

	if AMElement.OracleDateFormat.language == LANGUAGE_CN {
		if util.StringUtil.EqualsIgnoreCase("下午", string(runeStr)) {
			AMElement.OracleDateFormat.PM = true
			return offset + utf8.RuneLen('下') + utf8.RuneLen('午'), nil
		} else {
			AMElement.OracleDateFormat.PM = false
			return offset + utf8.RuneLen('上') + utf8.RuneLen('午'), nil
		}

	} else if util.StringUtil.EqualsIgnoreCase("PM", string(runeStr)) {
		AMElement.OracleDateFormat.PM = true
	} else {
		AMElement.OracleDateFormat.PM = false
	}

	return offset + 2, nil
}

func (AMElement amElement) format(dt []int) string {
	hour := dt[OFFSET_HOUR]
	language := int(0)
	if language == LANGUAGE_CN {
		if hour > 12 {
			return "下午"
		} else {
			return "上午"
		}
	}

	if hour > 12 {
		return "PM"
	} else {
		return "AM"
	}
}

/**
 * 将int值格式化成指定长度，长度不足前面补0，长度超过的取末尾指定长度
 */
func (OracleDateFormat *oracleDateFormat) formatInt(value int, len int) string {
	pow := int(math.Pow10(len))
	if value >= pow {
		value %= pow
	}
	value += pow
	return strconv.FormatInt(int64(value), 10)[1:]
}

/**
 * 格式化毫秒值
 * @param ms
 * @param len <= 6
 */
func (OracleDateFormat *oracleDateFormat) formatMilliSecond(ms int, len int) string {
	var ret string
	if ms < 10 {
		ret = "00000" + strconv.FormatInt(int64(ms), 10)
	} else if ms < 100 {
		ret = "0000" + strconv.FormatInt(int64(ms), 10)
	} else if ms < 1000 {
		ret = "000" + strconv.FormatInt(int64(ms), 10)
	} else if ms < 10000 {
		ret = "00" + strconv.FormatInt(int64(ms), 10)
	} else if ms < 100000 {
		ret = "0" + strconv.FormatInt(int64(ms), 10)
	} else {
		ret = strconv.FormatInt(int64(ms), 10)
	}

	if len < 6 {
		ret = ret[:len]
	}
	return ret
}

func getFormat() *oracleDateFormat {
	format := new(oracleDateFormat)
	format.PM = false
	format.TZNegative = false
	format.YearElement = yearElement{format, 4}
	format.MonthElement = monthElement{format, false, false}
	format.MonElement = monElement{format, false, false}
	format.MMElement = mmElement{format}
	format.DDElement = ddElement{format}
	format.HH24Element = hh24Element{format}
	format.HH12Element = hh12Element{format}
	format.MIElement = miElement{format}
	format.SSElement = ssElement{format}
	format.FElement = fElement{format, -1}
	format.TZHElement = tzhElement{format}
	format.TZMElement = tzmElement{format}
	format.AMElement = amElement{format}

	return format
}

func (OracleDateFormat *oracleDateFormat) parse(str string) (ret []int, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = ECGO_INVALID_DATETIME_FORMAT.throw()
		}
	}()
	OracleDateFormat.TZNegative = false
	OracleDateFormat.PM = false
	dt := make([]int, DT_LEN)
	// oracle默认年月日为 当前时间
	today := time.Now()
	dt[OFFSET_YEAR] = today.Year()
	dt[OFFSET_MONTH] = int(today.Month())
	dt[OFFSET_DAY] = today.Day()
	dt[OFFSET_TIMEZONE] = INVALID_VALUE
	offset := 0
	str = strings.TrimSpace(str)
	for _, obj := range OracleDateFormat.FormatElementList {
		// 跳过空格
		for str[offset] == ' ' && fmt.Sprintf("%+v", obj) != " " {
			offset++
		}
		if e, ok := obj.(element); ok {
			offset, err = e.parse(str, offset, dt)
			if err != nil {
				return nil, err
			}
		} else {
			offset += len(obj.(string))
		}
	}
	if offset < len(str) {
		//[6103]:文字与格式字符串不匹配.
		return nil, ECGO_INVALID_DATETIME_VALUE.throw()
	}

	// 12小时制时间转换
	if OracleDateFormat.PM {
		dt[OFFSET_HOUR] = (dt[OFFSET_HOUR] + 12) % 24
	}

	// 时区符号保留
	if OracleDateFormat.TZNegative {
		dt[OFFSET_TIMEZONE] = -dt[OFFSET_TIMEZONE]
	}

	// check day
	if dt[OFFSET_DAY] > getDaysOfMonth(dt[OFFSET_YEAR], dt[OFFSET_MONTH]) || dt[OFFSET_DAY] < 1 {
		return nil, ECGO_INVALID_DATETIME_VALUE.throw()
	}
	// check timezone 兼容oracle
	if dt[OFFSET_TIMEZONE] != INVALID_VALUE && (dt[OFFSET_TIMEZONE] > 14*60 || dt[OFFSET_TIMEZONE] <= -13*60) {
		return nil, ECGO_INVALID_DATETIME_VALUE.throw()
	}
	return dt, nil
}

func parse(str string, pattern string, language int) ([]int, error) {
	f := getFormat()
	f.setPattern(pattern)
	f.language = language
	return f.parse(str)
}

func (OracleDateFormat *oracleDateFormat) setPattern(pattern string) {
	if pattern != OracleDateFormat.pattern {
		OracleDateFormat.pattern = pattern
		OracleDateFormat.FormatElementList = OracleDateFormat.FormatElementList[:0]
		OracleDateFormat.analysePattern(pattern)
	}
}

func format(dt []int, pattern string, scale int32, language int) string {
	f := getFormat()
	f.setPattern(pattern)
	f.language = language
	f.scale = scale
	ret := f.format(dt)
	return ret
}

func (OracleDateFormat *oracleDateFormat) format(dt []int) string {
	sf := strings.Builder{}
	tzStart := false
	for _, obj := range OracleDateFormat.FormatElementList {
		_, ok1 := obj.(tzhElement)
		_, ok2 := obj.(tzmElement)
		if !tzStart && (ok1 || ok2) {
			tzStart = true
			if dt[OFFSET_TIMEZONE] < 0 {
				sf.WriteString("-")
			} else {
				sf.WriteString("+")
			}
		}

		if e, ok := obj.(element); ok {
			sf.WriteString(e.format(dt))
		} else {
			sf.WriteString(obj.(string))
		}
	}
	return sf.String()
}

/**
 * 解析格式串
 */
func (OracleDateFormat *oracleDateFormat) analysePattern(pattern string) ([]interface{}, error) {

	// 按分隔符split
	pattern = strings.TrimSpace(pattern)
	l := len(pattern)
	var splitPatterns []string
	starti := 0
	var curChar rune
	for i := 0; i < l; i++ {
		curChar = rune(pattern[i])
		if !unicode.IsDigit(curChar) && !unicode.IsLetter(curChar) {
			if i > starti {
				splitPatterns = append(splitPatterns, pattern[starti:i])
			}

			splitPatterns = append(splitPatterns, string(curChar))
			starti = i + 1
		} else if i == l-1 {
			splitPatterns = append(splitPatterns, pattern[starti:i+1])
		}
	}

	// 每个串按照从完整串，然后依次去掉一个末尾字符 来进行尝试规约
	for _, subPattern := range splitPatterns {
		if len(subPattern) != 1 || unicode.IsDigit(rune(subPattern[0])) || unicode.IsLetter(rune(subPattern[0])) {
			fmtWord := subPattern
			for subPattern != "" {
				i := len(subPattern)
				for ; i > 0; i-- {
					fmtWord = subPattern[0:i]
					element, err := OracleDateFormat.getFormatElement(fmtWord)
					if err != nil {
						return nil, err
					}
					if element != nil {
						// 忽略时区前面的+-号
						if element == OracleDateFormat.TZHElement || element == OracleDateFormat.TZMElement {
							var lastFormatElement string = OracleDateFormat.FormatElementList[len(OracleDateFormat.FormatElementList)-1].(string)
							if util.StringUtil.Equals("+", lastFormatElement) || util.StringUtil.Equals("-", lastFormatElement) {
								OracleDateFormat.FormatElementList = OracleDateFormat.FormatElementList[:len(OracleDateFormat.FormatElementList)-2]
							}
						}
						OracleDateFormat.FormatElementList = append(OracleDateFormat.FormatElementList, element)
						if i == len(subPattern) {
							subPattern = ""
						} else {
							subPattern = subPattern[i:len(subPattern)]
						}
						break
					}
				}

				if i == 0 {
					// 非标识符串
					OracleDateFormat.FormatElementList = append(OracleDateFormat.FormatElementList, subPattern)
					break
				}
			}

		} else {
			OracleDateFormat.FormatElementList = append(OracleDateFormat.FormatElementList, subPattern)
		}
	}
	return OracleDateFormat.FormatElementList, nil
}

func (OracleDateFormat *oracleDateFormat) getFormatElement(word string) (element, error) {
	if util.StringUtil.EqualsIgnoreCase("HH", word) || util.StringUtil.EqualsIgnoreCase("HH12", word) {
		return OracleDateFormat.HH12Element, nil
	} else if util.StringUtil.EqualsIgnoreCase("HH24", word) {
		return OracleDateFormat.HH24Element, nil
	} else if util.StringUtil.EqualsIgnoreCase("MI", word) {
		return OracleDateFormat.MIElement, nil
	} else if util.StringUtil.EqualsIgnoreCase("SS", word) {
		return OracleDateFormat.SSElement, nil
	} else if util.StringUtil.EqualsIgnoreCase("AM", word) || util.StringUtil.EqualsIgnoreCase("A.M.", word) || util.StringUtil.EqualsIgnoreCase("PM", word) || util.StringUtil.EqualsIgnoreCase("P.M.", word) {
		return OracleDateFormat.AMElement, nil
	} else if util.StringUtil.Equals("MONTH", word) {
		OracleDateFormat.MonthElement.upperCase = true
		OracleDateFormat.MonthElement.lowerCase = false
		return OracleDateFormat.MonthElement, nil
	} else if util.StringUtil.Equals("month", word) {
		OracleDateFormat.MonthElement.upperCase = false
		OracleDateFormat.MonthElement.lowerCase = true
		return OracleDateFormat.MonthElement, nil
	} else if util.StringUtil.EqualsIgnoreCase("Month", word) {
		OracleDateFormat.MonthElement.upperCase = false
		OracleDateFormat.MonthElement.lowerCase = false
		return OracleDateFormat.MonthElement, nil
	} else if util.StringUtil.Equals("MON", word) {
		OracleDateFormat.MonElement.upperCase = true
		OracleDateFormat.MonElement.lowerCase = false
		return OracleDateFormat.MonElement, nil
	} else if util.StringUtil.Equals("mon", word) {
		OracleDateFormat.MonElement.upperCase = false
		OracleDateFormat.MonElement.lowerCase = true
		return OracleDateFormat.MonElement, nil
	} else if util.StringUtil.EqualsIgnoreCase("Mon", word) {
		OracleDateFormat.MonElement.upperCase = false
		OracleDateFormat.MonElement.lowerCase = false
		return OracleDateFormat.MonElement, nil
	} else if util.StringUtil.EqualsIgnoreCase("MM", word) {
		return OracleDateFormat.MMElement, nil
	} else if util.StringUtil.EqualsIgnoreCase("DD", word) {
		return OracleDateFormat.DDElement, nil
	} else if util.StringUtil.EqualsIgnoreCase("TZH", word) {
		return OracleDateFormat.TZHElement, nil
	} else if util.StringUtil.EqualsIgnoreCase("TZM", word) {
		return OracleDateFormat.TZMElement, nil
	} else if strings.Index(word, "Y") == 0 || strings.Index(word, "y") == 0 {
		OracleDateFormat.YearElement.len = len(word)
		return OracleDateFormat.YearElement, nil
	} else if strings.Index(word, "F") == 0 || strings.Index(word, "f") == 0 {

		word = strings.ToUpper(word)
		numIndex := strings.LastIndex(word, "F") + 1
		var count int64
		var err error
		if numIndex < len(word) {
			count, err = strconv.ParseInt(word[numIndex:len(word)], 10, 32)
			if err != nil {
				return nil, err
			}
		} else {
			count = -1
		}

		OracleDateFormat.FElement.len = int(count)
		return OracleDateFormat.FElement, nil
	}

	return nil, nil
}
