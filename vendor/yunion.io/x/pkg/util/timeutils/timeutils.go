// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package timeutils

import (
	"fmt"
	"time"
	"strings"

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

func Localify(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().Local()
	}
	return now.Local()
}

const (
	IsoTimeFormat         = "2006-01-02T15:04:05Z07:00"
	IsoNoSecondTimeFormat = "2006-01-02T15:04Z07:00"
	FullIsoTimeFormat     = "2006-01-02T15:04:05.000000Z07:00"
	FullIsoNanoTimeFormat = "2006-01-02T15:04:05.000000000Z07:00"
	MysqlTimeFormat       = "2006-01-02 15:04:05"
	NormalTimeFormat      = "2006-01-02T15:04:05"
	FullNormalTimeFormat  = "2006-01-02T15:04:05.000000"
	CompactTimeFormat     = "20060102150405"
	DateFormat            = "2006-01-02"
	ShortDateFormat       = "20060102"
	ZStackTimeFormat      = "Jan 2, 2006 15:04:05 PM"

	RFC2882Format = time.RFC1123
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

func FullIsoNanoTime(now time.Time) string {
	return Utcify(now).Format(FullIsoNanoTimeFormat)
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

func ZStackTime(now time.Time) string {
	return Localify(now).Format(ZStackTimeFormat)
}

func ParseIsoTime(str string) (time.Time, error) {
	return time.Parse(IsoTimeFormat, str)
}

func ParseIsoNoSecondTime(str string) (time.Time, error) {
	return time.Parse(IsoNoSecondTimeFormat, str)
}

func toFullIsoNanoTimeFormat(str string) string {
	// 2019-09-17T20:50:17.66667134+08:00
	subsecStr := str[20:]
	pos := strings.IndexByte(subsecStr, 'Z')
	if pos < 0 {
		pos = strings.IndexByte(subsecStr, '+')
	}
	leftOver := subsecStr[pos:]
	subsecStr = subsecStr[:pos]
	for len(subsecStr) < 9 {
		subsecStr += "0"
	}
	return str[:20] + subsecStr + leftOver
}

func ParseFullIsoTime(str string) (time.Time, error) {
	return time.Parse(FullIsoNanoTimeFormat, toFullIsoNanoTimeFormat(str))
}

func ParseMysqlTime(str string) (time.Time, error) {
	return time.Parse(MysqlTimeFormat, str)
}

func ParseNormalTime(str string) (time.Time, error) {
	return time.Parse(NormalTimeFormat, str)
}

func ParseFullNormalTime(str string) (time.Time, error) {
	return time.Parse(FullNormalTimeFormat, str)
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

func ParseZStackDate(str string) (time.Time, error) {
	return time.ParseInLocation(ZStackTimeFormat, str, time.Local)
}

func ParseTimeStr(str string) (time.Time, error) {
	str = strings.TrimSpace(str)
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
	} else if regutils.MatchFullNormalTime(str) {
		return ParseFullNormalTime(str)
	} else if regutils.MatchRFC2882Time(str) {
		return ParseRFC2882Time(str)
	} else if regutils.MatchCompactTime(str) {
		return ParseCompactTime(str)
	} else if regutils.MatchDate(str) {
		return ParseDate(str)
	} else if regutils.MatchDateCompact(str) {
		return ParseShortDate(str)
	} else if regutils.MatchZStackTime(str) {
		return ParseZStackDate(str)
	} else {
		return time.Time{}, fmt.Errorf("unknown time format %s", str)
	}
}
