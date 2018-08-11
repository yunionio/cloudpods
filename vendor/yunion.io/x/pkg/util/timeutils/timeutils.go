package timeutils

import (
	"fmt"
	"time"

	"yunion.io/x/pkg/util/regutils"
)

func UtcNow() time.Time {
	return time.Now().UTC()
}

func Utcify(now time.Time) time.Time {
	if now.IsZero() {
		val := time.Now().UTC()
		return val
	} else {
		return now.UTC()
	}
}

const (
	IsoTimeFormat         = "2006-01-02T15:04:05Z"
	IsoNoSecondTimeFormat = "2006-01-02T15:04Z"
	FullIsoTimeFormat     = "2006-01-02T15:04:05.000000Z"
	MysqlTimeFormat       = "2006-01-02 15:04:05"
	NormalTimeFormat      = "2006-01-02T15:04:05"
	CompactTimeFormat     = "20060102150405"
	DateFormat            = "2006-01-02"
	ShortDateFormat       = "20060102"
	RFC2882Format         = time.RFC1123
)

func IsoTime(now time.Time) string {
	return Utcify(now).Format(IsoTimeFormat)
}

func IsoNoSecondTime(now time.Time) string {
	return Utcify(now).Format(IsoNoSecondTimeFormat)
}

func FullIsoTime(now time.Time) string {
	return Utcify(now).Format(FullIsoTimeFormat)
}

func MysqlTime(now time.Time) string {
	return Utcify(now).Format(MysqlTimeFormat)
}

func CompactTime(now time.Time) string {
	return Utcify(now).Format(CompactTimeFormat)
}

func RFC2882Time(now time.Time) string {
	return Utcify(now).Format(RFC2882Format)
}

func DateStr(now time.Time) string {
	return Utcify(now).Format(DateFormat)
}

func ShortDate(now time.Time) string {
	return Utcify(now).Format(ShortDateFormat)
}

func ParseIsoTime(str string) (time.Time, error) {
	return time.Parse(IsoTimeFormat, str)
}

func ParseIsoNoSecondTime(str string) (time.Time, error) {
	return time.Parse(IsoNoSecondTimeFormat, str)
}

func ParseFullIsoTime(str string) (time.Time, error) {
	return time.Parse(FullIsoTimeFormat, str)
}

func ParseMysqlTime(str string) (time.Time, error) {
	return time.Parse(MysqlTimeFormat, str)
}

func ParseNormalTime(str string) (time.Time, error) {
	return time.Parse(NormalTimeFormat, str)
}

func ParseCompactTime(str string) (time.Time, error) {
	return time.Parse(CompactTimeFormat, str)
}

func ParseRFC2882Time(str string) (time.Time, error) {
	return time.Parse(RFC2882Format, str)
}

func ParseDate(str string) (time.Time, error) {
	return time.Parse(DateFormat, str)
}

func ParseShortDate(str string) (time.Time, error) {
	return time.Parse(ShortDateFormat, str)
}

func ParseTimeStr(str string) (time.Time, error) {
	if regutils.MatchFullISOTime(str) {
		return ParseFullIsoTime(str)
	} else if regutils.MatchISOTime(str) {
		return ParseIsoTime(str)
	} else if regutils.MatchISONoSecondTime(str) {
		return ParseIsoNoSecondTime(str)
	} else if regutils.MatchMySQLTime(str) {
		return ParseMysqlTime(str)
	} else if regutils.MatchNormalTime(str) {
		return ParseNormalTime(str)
	} else if regutils.MatchRFC2882Time(str) {
		return ParseRFC2882Time(str)
	} else if regutils.MatchCompactTime(str) {
		return ParseCompactTime(str)
	} else if regutils.MatchDate(str) {
		return ParseDate(str)
	} else if regutils.MatchDateCompact(str) {
		return ParseShortDate(str)
	} else {
		return time.Time{}, fmt.Errorf("unknown time format %s", str)
	}
}
