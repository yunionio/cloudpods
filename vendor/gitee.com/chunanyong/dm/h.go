/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */
package dm

import (
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func encodeByString(x string, column column, conn DmConnection) ([]byte, error) {
	dt := make([]int, DT_LEN)
	if _, err := toDTFromString(x, dt); err != nil {
		return nil, err
	}
	return encode(dt, column, int(conn.dmConnector.localTimezone), int(conn.DbTimezone))
}

func encodeByTime(x time.Time, column column, conn DmConnection) ([]byte, error) {
	dt := toDTFromTime(x)
	return encode(dt, column, int(conn.dmConnector.localTimezone), int(conn.DbTimezone))
}

func toTimeFromString(str string, ltz int) time.Time {
	dt := make([]int, DT_LEN)
	toDTFromString(str, dt)
	return toTimeFromDT(dt, ltz)
}

func toTimeFromDT(dt []int, ltz int) time.Time {
	var year, month, day, hour, minute, second, nsec, tz int

	year = dt[OFFSET_YEAR]

	if dt[OFFSET_MONTH] > 0 {
		month = dt[OFFSET_MONTH]
	} else {
		month = 1
	}

	if dt[OFFSET_DAY] > 0 {
		day = dt[OFFSET_DAY]
	} else {
		day = 1
	}

	hour = dt[OFFSET_HOUR]
	minute = dt[OFFSET_MINUTE]
	second = dt[OFFSET_SECOND]
	nsec = dt[OFFSET_NANOSECOND]
	if dt[OFFSET_TIMEZONE] == INVALID_VALUE {
		tz = ltz * 60
	} else {
		tz = dt[OFFSET_TIMEZONE] * 60
	}
	return time.Date(year, time.Month(month), day, hour, minute, second, nsec, time.FixedZone("", tz))
}

func decode(value []byte, isBdta bool, column column, ltz int, dtz int) []int {
	var dt []int
	if isBdta {
		dt = dmdtDecodeBdta(value)
	} else {
		dt = dmdtDecodeFast(value)
	}

	if column.mask == MASK_LOCAL_DATETIME {
		transformTZ(dt, dtz, ltz)
	}

	return dt
}

func dmdtDecodeFast(value []byte) []int {
	dt := make([]int, DT_LEN)
	dt[OFFSET_TIMEZONE] = INVALID_VALUE

	dtype := 0
	if len(value) == DATE_PREC {
		dtype = DATE
	} else if len(value) == TIME_PREC {
		dtype = TIME
	} else if len(value) == TIME_TZ_PREC {
		dtype = TIME_TZ
	} else if len(value) == DATETIME_PREC {
		dtype = DATETIME
	} else if len(value) == DATETIME2_PREC {
		dtype = DATETIME2
	} else if len(value) == DATETIME_TZ_PREC {
		dtype = DATETIME_TZ
	} else if len(value) == DATETIME2_TZ_PREC {
		dtype = DATETIME2_TZ
	}

	if dtype == DATE {

		dt[OFFSET_YEAR] = int(Dm_build_650.Dm_build_747(value, 0)) & 0x7FFF
		if dt[OFFSET_YEAR] > 9999 {
			dt[OFFSET_YEAR] = int(int16(dt[OFFSET_YEAR] | 0x8000))
		}

		dt[OFFSET_MONTH] = ((int(value[1]) >> 7) & 0x1) + ((int(value[2]) & 0x07) << 1)

		dt[OFFSET_DAY] = ((int(value[2]) & 0xF8) >> 3) & 0x1f
	} else if dtype == TIME {
		dt[OFFSET_HOUR] = int(value[0]) & 0x1F
		dt[OFFSET_MINUTE] = ((int(value[0]) >> 5) & 0x07) + ((int(value[1]) & 0x07) << 3)
		dt[OFFSET_SECOND] = ((int(value[1]) >> 3) & 0x1f) + ((int(value[2]) & 0x01) << 5)
		dt[OFFSET_NANOSECOND] = ((int(value[2]) >> 1) & 0x7f) + ((int(value[3]) & 0x00ff) << 7) + ((int(value[4]) & 0x1F) << 15)
		dt[OFFSET_NANOSECOND] *= 1000
	} else if dtype == TIME_TZ {
		dt[OFFSET_HOUR] = int(value[0]) & 0x1F
		dt[OFFSET_MINUTE] = ((int(value[0]) >> 5) & 0x07) + ((int(value[1]) & 0x07) << 3)
		dt[OFFSET_SECOND] = ((int(value[1]) >> 3) & 0x1f) + ((int(value[2]) & 0x01) << 5)
		dt[OFFSET_NANOSECOND] = ((int(value[2]) >> 1) & 0x7f) + ((int(value[3]) & 0x00ff) << 7) + ((int(value[4]) & 0x1F) << 15)
		dt[OFFSET_NANOSECOND] *= 1000
		dt[OFFSET_TIMEZONE] = int(Dm_build_650.Dm_build_747(value, 5))
	} else if dtype == DATETIME {

		dt[OFFSET_YEAR] = int(Dm_build_650.Dm_build_747(value, 0)) & 0x7FFF
		if dt[OFFSET_YEAR] > 9999 {
			dt[OFFSET_YEAR] = int(int16(dt[OFFSET_YEAR] | 0x8000))
		}

		dt[OFFSET_MONTH] = ((int(value[1]) >> 7) & 0x1) + ((int(value[2]) & 0x07) << 1)

		dt[OFFSET_DAY] = ((int(value[2]) & 0xF8) >> 3) & 0x1f

		dt[OFFSET_HOUR] = (int(value[3]) & 0x1F)

		dt[OFFSET_MINUTE] = ((int(value[3]) >> 5) & 0x07) + ((int(value[4]) & 0x07) << 3)

		dt[OFFSET_SECOND] = ((int(value[4]) >> 3) & 0x1f) + ((int(value[5]) & 0x01) << 5)

		dt[OFFSET_NANOSECOND] = ((int(value[5]) >> 1) & 0x7f) + ((int(value[6]) & 0x00ff) << 7) + ((int(value[7]) & 0x1F) << 15)
		dt[OFFSET_NANOSECOND] *= 1000
	} else if dtype == DATETIME_TZ {

		dt[OFFSET_YEAR] = int(Dm_build_650.Dm_build_747(value, 0)) & 0x7FFF
		if dt[OFFSET_YEAR] > 9999 {
			dt[OFFSET_YEAR] = int(int16(dt[OFFSET_YEAR] | 0x8000))
		}

		dt[OFFSET_MONTH] = ((int(value[1]) >> 7) & 0x1) + ((int(value[2]) & 0x07) << 1)

		dt[OFFSET_DAY] = ((int(value[2]) & 0xF8) >> 3) & 0x1f

		dt[OFFSET_HOUR] = (int(value[3]) & 0x1F)

		dt[OFFSET_MINUTE] = ((int(value[3]) >> 5) & 0x07) + ((int(value[4]) & 0x07) << 3)

		dt[OFFSET_SECOND] = ((int(value[4]) >> 3) & 0x1f) + ((int(value[5]) & 0x01) << 5)

		dt[OFFSET_NANOSECOND] = ((int(value[5]) >> 1) & 0x7f) + ((int(value[6]) & 0x00ff) << 7) + ((int(value[7]) & 0x1F) << 15)
		dt[OFFSET_NANOSECOND] *= 1000

		dt[OFFSET_TIMEZONE] = int(Dm_build_650.Dm_build_747(value, len(value)-2))
	} else if dtype == DATETIME2 {

		dt[OFFSET_YEAR] = int(Dm_build_650.Dm_build_747(value, 0)) & 0x7FFF
		if dt[OFFSET_YEAR] > 9999 {
			dt[OFFSET_YEAR] = int(int16(dt[OFFSET_YEAR] | 0x8000))
		}

		dt[OFFSET_MONTH] = ((int(value[1]) >> 7) & 0x1) + ((int(value[2]) & 0x07) << 1)

		dt[OFFSET_DAY] = ((int(value[2]) & 0xF8) >> 3) & 0x1f

		dt[OFFSET_HOUR] = (int(value[3]) & 0x1F)

		dt[OFFSET_MINUTE] = ((int(value[3]) >> 5) & 0x07) + ((int(value[4]) & 0x07) << 3)

		dt[OFFSET_SECOND] = ((int(value[4]) >> 3) & 0x1f) + ((int(value[5]) & 0x01) << 5)

		dt[OFFSET_NANOSECOND] = ((int(value[5]) >> 1) & 0x7f) + ((int(value[6]) & 0x00ff) << 7) + ((int(value[7]) & 0x00ff) << 15) + ((int(value[8]) & 0x7F) << 23)
	} else if dtype == DATETIME2_TZ {

		dt[OFFSET_YEAR] = int(Dm_build_650.Dm_build_747(value, 0)) & 0x7FFF
		if dt[OFFSET_YEAR] > 9999 {
			dt[OFFSET_YEAR] = int(int16(dt[OFFSET_YEAR] | 0x8000))
		}

		dt[OFFSET_MONTH] = ((int(value[1]) >> 7) & 0x1) + ((int(value[2]) & 0x07) << 1)

		dt[OFFSET_DAY] = ((int(value[2]) & 0xF8) >> 3) & 0x1f

		dt[OFFSET_HOUR] = (int(value[3]) & 0x1F)

		dt[OFFSET_MINUTE] = ((int(value[3]) >> 5) & 0x07) + ((int(value[4]) & 0x07) << 3)

		dt[OFFSET_SECOND] = ((int(value[4]) >> 3) & 0x1f) + ((int(value[5]) & 0x01) << 5)

		dt[OFFSET_NANOSECOND] = ((int(value[5]) >> 1) & 0x7f) + ((int(value[6]) & 0x00ff) << 7) + ((int(value[7]) & 0x00ff) << 15) + ((int(value[8]) & 0x7F) << 23)

		dt[OFFSET_TIMEZONE] = int(Dm_build_650.Dm_build_747(value, len(value)-2))
	}
	return dt
}

func dmdtDecodeBdta(value []byte) []int {
	dt := make([]int, DT_LEN)
	dt[OFFSET_YEAR] = int(Dm_build_650.Dm_build_747(value, 0))
	dt[OFFSET_MONTH] = int(value[2] & 0xFF)
	dt[OFFSET_DAY] = int(value[3] & 0xFF)
	dt[OFFSET_HOUR] = int(value[4] & 0xFF)
	dt[OFFSET_MINUTE] = int(value[5] & 0xFF)
	dt[OFFSET_SECOND] = int(value[6] & 0xFF)
	dt[OFFSET_NANOSECOND] = int((value[7] & 0xFF) + (value[8] << 8) + (value[9] << 16))
	dt[OFFSET_TIMEZONE] = int(Dm_build_650.Dm_build_747(value, 10))

	if len(value) > 12 {

		dt[OFFSET_NANOSECOND] += int(value[12] << 24)
	}
	return dt
}

func dtToStringByOracleFormat(dt []int, oracleFormatPattern string, scale int32, language int) string {
	return format(dt, oracleFormatPattern, scale, language)
}

func dtToString(dt []int, dtype int, scale int) string {
	switch dtype {
	case DATE:
		return formatYear(dt[OFFSET_YEAR]) + "-" + format2(dt[OFFSET_MONTH]) + "-" + format2(dt[OFFSET_DAY])

	case TIME:
		if scale > 0 {
			return format2(dt[OFFSET_HOUR]) + ":" + format2(dt[OFFSET_MINUTE]) + ":" + format2(dt[OFFSET_SECOND]) + "." + formatMilliSecond(dt[OFFSET_NANOSECOND], scale)
		} else {
			return format2(dt[OFFSET_HOUR]) + ":" + format2(dt[OFFSET_MINUTE]) + ":" + format2(dt[OFFSET_SECOND])
		}

	case TIME_TZ:
		if scale > 0 {
			return format2(dt[OFFSET_HOUR]) + ":" + format2(dt[OFFSET_MINUTE]) + ":" + format2(dt[OFFSET_SECOND]) + "." + formatMilliSecond(dt[OFFSET_NANOSECOND], scale) + " " + formatTZ(dt[OFFSET_TIMEZONE])
		} else {
			return format2(dt[OFFSET_HOUR]) + ":" + format2(dt[OFFSET_MINUTE]) + ":" + format2(dt[OFFSET_SECOND]) + " " + formatTZ(dt[OFFSET_TIMEZONE])
		}

	case DATETIME, DATETIME2:
		if scale > 0 {
			return formatYear(dt[OFFSET_YEAR]) + "-" + format2(dt[OFFSET_MONTH]) + "-" + format2(dt[OFFSET_DAY]) + " " + format2(dt[OFFSET_HOUR]) + ":" + format2(dt[OFFSET_MINUTE]) + ":" + format2(dt[OFFSET_SECOND]) + "." + formatMilliSecond(dt[OFFSET_NANOSECOND], scale)
		} else {
			return formatYear(dt[OFFSET_YEAR]) + "-" + format2(dt[OFFSET_MONTH]) + "-" + format2(dt[OFFSET_DAY]) + " " + format2(dt[OFFSET_HOUR]) + ":" + format2(dt[OFFSET_MINUTE]) + ":" + format2(dt[OFFSET_SECOND])
		}

	case DATETIME_TZ, DATETIME2_TZ:
		if scale > 0 {
			return formatYear(dt[OFFSET_YEAR]) + "-" + format2(dt[OFFSET_MONTH]) + "-" + format2(dt[OFFSET_DAY]) + " " + format2(dt[OFFSET_HOUR]) + ":" + format2(dt[OFFSET_MINUTE]) + ":" + format2(dt[OFFSET_SECOND]) + "." + formatMilliSecond(dt[OFFSET_NANOSECOND], scale) + " " + formatTZ(dt[OFFSET_TIMEZONE])
		} else {
			return formatYear(dt[OFFSET_YEAR]) + "-" + format2(dt[OFFSET_MONTH]) + "-" + format2(dt[OFFSET_DAY]) + " " + format2(dt[OFFSET_HOUR]) + ":" + format2(dt[OFFSET_MINUTE]) + ":" + format2(dt[OFFSET_SECOND]) + " " + formatTZ(dt[OFFSET_TIMEZONE])
		}
	}

	return ""
}

func formatYear(value int) string {
	if value >= 0 {
		if value < 10 {
			return "000" + strconv.FormatInt(int64(value), 10)
		} else if value < 100 {
			return "00" + strconv.FormatInt(int64(value), 10)
		} else if value < 1000 {
			return "0" + strconv.FormatInt(int64(value), 10)
		} else {
			return strconv.FormatInt(int64(value), 10)
		}
	} else {
		if value > -10 {
			return "-000" + strconv.FormatInt(int64(-value), 10)
		} else if value > -100 {
			return "-00" + strconv.FormatInt(int64(-value), 10)
		} else if value > -1000 {
			return "-0" + strconv.FormatInt(int64(-value), 10)
		} else {
			return strconv.FormatInt(int64(value), 10)
		}
	}
}

func format2(value int) string {
	if value < 10 {
		return "0" + strconv.FormatInt(int64(value), 10)
	} else {
		return strconv.FormatInt(int64(value), 10)
	}
}

func formatMilliSecond(ms int, prec int) string {
	var ret string
	if ms < 10 {
		ret = "00000000" + strconv.FormatInt(int64(ms), 10)
	} else if ms < 100 {
		ret = "0000000" + strconv.FormatInt(int64(ms), 10)
	} else if ms < 1000 {
		ret = "000000" + strconv.FormatInt(int64(ms), 10)
	} else if ms < 10000 {
		ret = "00000" + strconv.FormatInt(int64(ms), 10)
	} else if ms < 100000 {
		ret = "0000" + strconv.FormatInt(int64(ms), 10)
	} else if ms < 1000000 {
		ret = "000" + strconv.FormatInt(int64(ms), 10)
	} else if ms < 10000000 {
		ret = "00" + strconv.FormatInt(int64(ms), 10)
	} else if ms < 100000000 {
		ret = "0" + strconv.FormatInt(int64(ms), 10)
	} else {
		ret = strconv.FormatInt(int64(ms), 10)
	}

	if prec < NANOSECOND_DIGITS {
		ret = ret[:prec]
	}
	return ret
}

func formatTZ(tz int) string {
	tz_hour := int(math.Abs(float64(tz / 60)))
	tz_min := int(math.Abs(float64(tz % 60)))

	if tz >= 0 {
		return "+" + format2(tz_hour) + ":" + format2(tz_min)
	} else {
		return "-" + format2(tz_hour) + ":" + format2(tz_min)
	}
}

func toDTFromTime(x time.Time) []int {
	hour, min, sec := x.Clock()
	ts := make([]int, DT_LEN)
	ts[OFFSET_YEAR] = x.Year()
	ts[OFFSET_MONTH] = int(x.Month())
	ts[OFFSET_DAY] = x.Day()
	ts[OFFSET_HOUR] = hour
	ts[OFFSET_MINUTE] = min
	ts[OFFSET_SECOND] = sec
	ts[OFFSET_NANOSECOND] = (int)(x.Nanosecond())
	_, tz := x.Zone()
	ts[OFFSET_TIMEZONE] = tz / 60
	return ts
}

func toDTFromUnix(sec int64, nsec int64) []int {
	return toDTFromTime(time.Unix(sec, nsec))
}

func toDTFromString(s string, dt []int) (dtype int, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = ECGO_INVALID_DATETIME_FORMAT.throw()
		}
	}()
	date_s := ""
	time_s := ""
	nanos_s := ""
	tz_s := ""
	year := 0
	month := 0
	day := 0
	hour := 0
	minute := 0
	second := 0
	a_nanos := 0
	firstDash := -1
	secondDash := -1
	firstColon := -1
	secondColon := -1
	period := -1
	sign := 0
	ownTz := INVALID_VALUE
	dtype = -1

	zeros := "000000000"

	if s != "" && strings.TrimSpace(s) == "" {
		return 0, ECGO_INVALID_DATETIME_FORMAT.throw()
	}
	s = strings.TrimSpace(s)

	if strings.Index(s, "-") == 0 {
		s = strings.TrimSpace(s[1:])
		sign = 1
	}

	comps := strings.Split(s, " ")

	switch len(comps) {
	case 3:
		date_s = comps[0]
		time_s = comps[1]
		tz_s = comps[2]
		dtype = DATETIME_TZ

	case 2:
		if strings.Index(comps[0], ":") > 0 {
			time_s = comps[0]
			tz_s = comps[1]
			dtype = TIME_TZ
		} else {
			date_s = comps[0]
			time_s = comps[1]
			dtype = DATETIME
		}

	case 1:
		if strings.Index(comps[0], ":") > 0 {
			time_s = comps[0]
			dtype = TIME
		} else {
			date_s = comps[0]
			dtype = DATE
		}

	default:
		return 0, ECGO_INVALID_DATETIME_FORMAT.throw()
	}

	if date_s != "" {

		firstDash = strings.Index(date_s, "-")
		secondDash = strings.Index(date_s[firstDash+1:], "-")

		if firstDash < 0 || secondDash < 0 {
			firstDash = strings.Index(s, ".")
			secondDash = strings.Index(date_s[firstDash+1:], ".")
		}

		if firstDash < 0 || secondDash < 0 {
			firstDash = strings.Index(s, "/")
			secondDash = strings.Index(date_s[firstDash+1:], "/")
		}
		if secondDash > 0 {
			secondDash += firstDash + 1
		}

		if (firstDash > 0) && (secondDash > 0) && (secondDash < len(date_s)-1) {

			if sign == 1 {
				i, err := strconv.ParseInt(date_s[:firstDash], 10, 32)
				if err != nil {
					return 0, ECGO_INVALID_DATETIME_FORMAT.addDetailln(err.Error()).throw()
				}
				year = 0 - int(i) - 1900
			} else {
				i, err := strconv.ParseInt(date_s[:firstDash], 10, 32)
				if err != nil {
					return 0, ECGO_INVALID_DATETIME_FORMAT.addDetailln(err.Error()).throw()
				}
				year = int(i) - 1900
			}

			i, err := strconv.ParseInt(date_s[firstDash+1:secondDash], 10, 32)
			if err != nil {
				return 0, ECGO_INVALID_DATETIME_FORMAT.addDetailln(err.Error()).throw()
			}
			month = int(i) - 1

			i, err = strconv.ParseInt(date_s[secondDash+1:], 10, 32)
			if err != nil {
				return 0, ECGO_INVALID_DATETIME_FORMAT.addDetailln(err.Error()).throw()
			}
			day = int(i)

			if !checkDate(year+1900, month+1, day) {
				return -1, ECGO_INVALID_DATETIME_FORMAT.throw()
			}
		} else {
			return -1, ECGO_INVALID_DATETIME_FORMAT.throw()
		}
	}

	if time_s != "" {
		firstColon = strings.Index(time_s, ":")
		secondColon = strings.Index(time_s[firstColon+1:], ":")
		if secondColon > 0 {
			secondColon += firstColon + 1
		}

		period = strings.Index(time_s[secondColon+1:], ".")
		if period > 0 {
			period += secondColon + 1
		}

		if (firstColon > 0) && (secondColon > 0) && (secondColon < len(time_s)-1) {
			i, err := strconv.ParseInt(time_s[:firstColon], 10, 32)
			if err != nil {
				return 0, ECGO_INVALID_DATETIME_FORMAT.addDetailln(err.Error()).throw()
			}
			hour = int(i)

			i, err = strconv.ParseInt(time_s[firstColon+1:secondColon], 10, 32)
			if err != nil {
				return 0, ECGO_INVALID_DATETIME_FORMAT.addDetailln(err.Error()).throw()
			}
			minute = int(i)

			if period > 0 && period < len(time_s)-1 {
				i, err = strconv.ParseInt(time_s[secondColon+1:period], 10, 32)
				if err != nil {
					return 0, ECGO_INVALID_DATETIME_FORMAT.addDetailln(err.Error()).throw()
				}
				second = int(i)

				nanos_s = time_s[period+1:]
				if len(nanos_s) > NANOSECOND_DIGITS {
					return -1, ECGO_INVALID_DATETIME_FORMAT.throw()
				}
				if !unicode.IsDigit(rune(nanos_s[0])) {
					return -1, ECGO_INVALID_DATETIME_FORMAT.throw()
				}
				nanos_s = nanos_s + zeros[0:NANOSECOND_DIGITS-len(nanos_s)]

				i, err = strconv.ParseInt(nanos_s, 10, 32)
				if err != nil {
					return 0, ECGO_INVALID_DATETIME_FORMAT.addDetailln(err.Error()).throw()
				}
				a_nanos = int(i)
			} else if period > 0 {
				return -1, ECGO_INVALID_DATETIME_FORMAT.throw()
			} else {
				i, err = strconv.ParseInt(time_s[secondColon+1:], 10, 32)
				if err != nil {
					return 0, ECGO_INVALID_DATETIME_FORMAT.addDetailln(err.Error()).throw()
				}
				second = int(i)
			}

			if hour >= 24 || hour < 0 || minute >= 60 || minute < 0 || second >= 60 || second < 0 {
				return -1, ECGO_INVALID_DATETIME_FORMAT.throw()
			}
		} else {
			return -1, ECGO_INVALID_DATETIME_FORMAT.throw()
		}
	}

	if tz_s != "" {
		neg := false
		if strings.Index(tz_s, "-") == 0 {
			neg = true
		}

		if strings.Index(tz_s, "-") == 0 || strings.Index(tz_s, "+") == 0 {
			tz_s = strings.TrimSpace(tz_s[1:])
		}

		hm := strings.Split(tz_s, ":")
		var tzh, tzm int16 = 0, 0
		switch len(hm) {
		case 2:
			s, err := strconv.ParseInt(strings.TrimSpace(hm[0]), 10, 16)
			if err != nil {
				return -1, ECGO_INVALID_DATETIME_FORMAT.throw()
			}
			tzh = int16(s)

			s, err = strconv.ParseInt(strings.TrimSpace(hm[1]), 10, 16)
			if err != nil {
				return -1, ECGO_INVALID_DATETIME_FORMAT.throw()
			}
			tzm = int16(s)
		case 1:
			s, err := strconv.ParseInt(strings.TrimSpace(hm[0]), 10, 16)
			if err != nil {
				return -1, ECGO_INVALID_DATETIME_FORMAT.throw()
			}
			tzh = int16(s)
		default:
			return -1, ECGO_INVALID_DATETIME_FORMAT.throw()
		}

		ownTz = int(tzh*60 + tzm)
		if ownTz < 0 {
			return -1, ECGO_INVALID_DATETIME_FORMAT.throw()
		}

		if neg {
			ownTz *= -1
		}

		if ownTz <= -13*60 || ownTz > 14*60 {
			return -1, ECGO_INVALID_DATETIME_FORMAT.throw()
		}
	}

	dt[OFFSET_YEAR] = year + 1900
	dt[OFFSET_MONTH] = month + 1
	if day == 0 {
		dt[OFFSET_DAY] = 1
	} else {
		dt[OFFSET_DAY] = day
	}
	dt[OFFSET_HOUR] = hour
	dt[OFFSET_MINUTE] = minute
	dt[OFFSET_SECOND] = second
	dt[OFFSET_NANOSECOND] = a_nanos
	dt[OFFSET_TIMEZONE] = int(ownTz)
	return dtype, nil
}

func transformTZ(dt []int, defaultSrcTz int, destTz int) {
	srcTz := defaultSrcTz

	if srcTz != INVALID_VALUE && destTz != INVALID_VALUE && destTz != srcTz {
		dt = addMinute(dt, destTz-srcTz)

		dt[OFFSET_TIMEZONE] = destTz

	}
}

func encode(dt []int, column column, lTz int, dTz int) ([]byte, error) {
	if dt[OFFSET_TIMEZONE] != INVALID_VALUE {
		transformTZ(dt, dt[OFFSET_TIMEZONE], lTz)
	}

	if column.mask == MASK_LOCAL_DATETIME {
		transformTZ(dt, dt[OFFSET_TIMEZONE], dTz)
	}

	if dt[OFFSET_YEAR] < -4712 || dt[OFFSET_YEAR] > 9999 {
		return nil, ECGO_DATETIME_OVERFLOW.throw()
	}

	year := dt[OFFSET_YEAR]

	month := dt[OFFSET_MONTH]

	day := dt[OFFSET_DAY]

	hour := dt[OFFSET_HOUR]

	min := dt[OFFSET_MINUTE]

	sec := dt[OFFSET_SECOND]

	msec := dt[OFFSET_NANOSECOND]

	var tz int

	if dt[OFFSET_TIMEZONE] == INVALID_VALUE {
		tz = dTz
	} else {
		tz = dt[OFFSET_TIMEZONE]
	}

	var ret []byte

	if column.colType == DATE {
		ret = make([]byte, 3)

		ret[0] = (byte)(year & 0xFF)

		if year >= 0 {
			ret[1] = (byte)((year >> 8) | ((month & 0x01) << 7))
		} else {
			ret[1] = (byte)((year >> 8) & (((month & 0x01) << 7) | 0x7f))
		}

		ret[2] = (byte)(((month & 0x0E) >> 1) | (day << 3))
	} else if column.colType == DATETIME {
		msec /= 1000
		ret = make([]byte, 8)

		ret[0] = (byte)(year & 0xFF)

		if year >= 0 {
			ret[1] = (byte)((year >> 8) | ((month & 0x01) << 7))
		} else {
			ret[1] = (byte)((year >> 8) & (((month & 0x01) << 7) | 0x7f))
		}

		ret[2] = (byte)(((month & 0x0E) >> 1) | (day << 3))

		ret[3] = (byte)(hour | ((min & 0x07) << 5))

		ret[4] = (byte)(((min & 0x38) >> 3) | ((sec & 0x1F) << 3))

		ret[5] = (byte)(((sec & 0x20) >> 5) | ((msec & 0x7F) << 1))

		ret[6] = (byte)((msec >> 7) & 0xFF)

		ret[7] = (byte)((msec >> 15) & 0xFF)
	} else if column.colType == DATETIME2 {
		ret = make([]byte, 9)

		ret[0] = (byte)(year & 0xFF)

		if year >= 0 {
			ret[1] = (byte)((year >> 8) | ((month & 0x01) << 7))
		} else {
			ret[1] = (byte)((year >> 8) & (((month & 0x01) << 7) | 0x7f))
		}

		ret[2] = (byte)(((month & 0x0E) >> 1) | (day << 3))

		ret[3] = (byte)(hour | ((min & 0x07) << 5))

		ret[4] = (byte)(((min & 0x38) >> 3) | ((sec & 0x1F) << 3))

		ret[5] = (byte)(((sec & 0x20) >> 5) | ((msec & 0x7F) << 1))

		ret[6] = (byte)((msec >> 7) & 0xFF)

		ret[7] = (byte)((msec >> 15) & 0xFF)

		ret[8] = (byte)((msec >> 23) & 0xFF)
	} else if column.colType == DATETIME_TZ {
		msec /= 1000
		ret = make([]byte, 10)

		ret[0] = (byte)(year & 0xFF)

		if year >= 0 {
			ret[1] = (byte)((year >> 8) | ((month & 0x01) << 7))
		} else {
			ret[1] = (byte)((year >> 8) & (((month & 0x01) << 7) | 0x7f))
		}

		ret[2] = (byte)(((month & 0x0E) >> 1) | (day << 3))

		ret[3] = (byte)(hour | ((min & 0x07) << 5))

		ret[4] = (byte)(((min & 0x38) >> 3) | ((sec & 0x1F) << 3))

		ret[5] = (byte)(((sec & 0x20) >> 5) | ((msec & 0x7F) << 1))

		ret[6] = (byte)((msec >> 7) & 0xFF)

		ret[7] = (byte)((msec >> 15) & 0xFF)

		Dm_build_650.Dm_build_661(ret, 8, int16(tz))
	} else if column.colType == DATETIME2_TZ {
		ret = make([]byte, 11)

		ret[0] = (byte)(year & 0xFF)

		if year >= 0 {
			ret[1] = (byte)((year >> 8) | ((month & 0x01) << 7))
		} else {
			ret[1] = (byte)((year >> 8) & (((month & 0x01) << 7) | 0x7f))
		}

		ret[2] = (byte)(((month & 0x0E) >> 1) | (day << 3))

		ret[3] = (byte)(hour | ((min & 0x07) << 5))

		ret[4] = (byte)(((min & 0x38) >> 3) | ((sec & 0x1F) << 3))

		ret[5] = (byte)(((sec & 0x20) >> 5) | ((msec & 0x7F) << 1))

		ret[6] = (byte)((msec >> 7) & 0xFF)

		ret[7] = (byte)((msec >> 15) & 0xFF)

		ret[8] = (byte)((msec >> 23) & 0xFF)

		Dm_build_650.Dm_build_661(ret, 8, int16(tz))
	} else if column.colType == TIME {
		msec /= 1000
		ret = make([]byte, 5)

		ret[0] = (byte)(hour | ((min & 0x07) << 5))

		ret[1] = (byte)(((min & 0x38) >> 3) | ((sec & 0x1F) << 3))

		ret[2] = (byte)(((sec & 0x20) >> 5) | ((msec & 0x7F) << 1))

		ret[3] = (byte)((msec >> 7) & 0xFF)

		ret[4] = (byte)((msec >> 15) & 0xFF)
	} else if column.colType == TIME_TZ {
		msec /= 1000
		ret = make([]byte, 7)

		ret[0] = (byte)(hour | ((min & 0x07) << 5))

		ret[1] = (byte)(((min & 0x38) >> 3) | ((sec & 0x1F) << 3))

		ret[2] = (byte)(((sec & 0x20) >> 5) | ((msec & 0x7F) << 1))

		ret[3] = (byte)((msec >> 7) & 0xFF)

		ret[4] = (byte)((msec >> 15) & 0xFF)

		Dm_build_650.Dm_build_661(ret, 5, int16(tz))
	}

	return ret, nil
}

func toDate(x int64, column column, conn DmConnection) ([]byte, error) {
	switch column.colType {
	case DATETIME, DATETIME2:
		if x > 2958463*24*60*60 {
			return nil, ECGO_DATETIME_OVERFLOW.throw()
		}

		dt := toDTFromUnix(x-Seconds_1900_1970, 0)
		return encode(dt, column, int(conn.dmConnector.localTimezone), int(conn.DbTimezone))

	case TIME:
		dt := toDTFromUnix(x, 0)
		return encode(dt, column, int(conn.dmConnector.localTimezone), int(conn.DbTimezone))

	case DATE:
		if x > 2958463 {
			return nil, ECGO_DATETIME_OVERFLOW.throw()
		}

		dt := toDTFromUnix(x*24*60*60-Seconds_1900_1970, 0)
		if dt[OFFSET_YEAR] < -4712 || dt[OFFSET_YEAR] > 9999 {
			return nil, ECGO_DATETIME_OVERFLOW.throw()
		}
		return encode(dt, column, int(conn.dmConnector.localTimezone), int(conn.DbTimezone))

	default:
		return nil, ECGO_DATA_CONVERTION_ERROR.throw()
	}
}

func checkDate(year int, month int, day int) bool {
	if year > 9999 || year < -4712 || month > 12 || month < 1 {
		return false
	}

	monthDays := getDaysOfMonth(year, month)
	if day > monthDays || day < 1 {
		return false
	}
	return true
}

func getDaysOfMonth(year int, month int) int {
	switch month {
	case 1, 3, 5, 7, 8, 10, 12:
		return 31
	case 4, 6, 9, 11:
		return 30
	case 2:
		if isLeapYear(year) {
			return 29
		}
		return 28
	default:
		return 0
	}
}

func isLeapYear(year int) bool {
	return (year%4 == 0 && year%100 != 0) || year%400 == 0
}

func addYear(dt []int, n int) []int {
	dt[OFFSET_YEAR] += n
	return dt
}

func addMonth(dt []int, n int) []int {
	month := dt[OFFSET_MONTH] + n
	addYearValue := month / 12
	if month %= 12; month < 1 {
		month += 12
		addYearValue--
	}

	daysOfMonth := getDaysOfMonth(dt[OFFSET_YEAR], month)
	if dt[OFFSET_DAY] > daysOfMonth {
		dt[OFFSET_DAY] = daysOfMonth
	}

	dt[OFFSET_MONTH] = month
	addYear(dt, addYearValue)
	return dt
}

func addDay(dt []int, n int) []int {
	tmp := dt[OFFSET_DAY] + n
	monthDays := 0
	monthDays = getDaysOfMonth(dt[OFFSET_YEAR], dt[OFFSET_MONTH])
	for tmp > monthDays || tmp <= 0 {
		if tmp > monthDays {
			addMonth(dt, 1)
			tmp -= monthDays
		} else {
			addMonth(dt, -1)
			tmp += monthDays
		}
	}
	dt[OFFSET_DAY] = tmp
	return dt
}

func addHour(dt []int, n int) []int {
	hour := dt[OFFSET_HOUR] + n
	addDayValue := hour / 24
	if hour %= 24; hour < 0 {
		hour += 24
		addDayValue--
	}

	dt[OFFSET_HOUR] = hour
	addDay(dt, addDayValue)
	return dt
}

func addMinute(dt []int, n int) []int {
	minute := dt[OFFSET_MINUTE] + n
	addHourValue := minute / 60
	if minute %= 60; minute < 0 {
		minute += 60
		addHourValue--
	}

	dt[OFFSET_MINUTE] = minute
	addHour(dt, addHourValue)
	return dt
}
