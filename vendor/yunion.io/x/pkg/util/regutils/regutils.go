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

package regutils

import (
	"net"
	"regexp"
	"strings"
)

var FUNCTION_REG *regexp.Regexp
var UUID_REG *regexp.Regexp
var UUID_EXACT_REG *regexp.Regexp
var INTEGER_REG *regexp.Regexp
var FLOAT_REG *regexp.Regexp
var MACADDR_REG *regexp.Regexp
var COMPACT_MACADDR_REG *regexp.Regexp
var NSPTR_REG *regexp.Regexp
var NAME_REG *regexp.Regexp
var DOMAINNAME_REG *regexp.Regexp
var DOMAINSRV_REG *regexp.Regexp
var SIZE_REG *regexp.Regexp
var MONTH_REG *regexp.Regexp
var DATE_REG *regexp.Regexp
var DATE_COMPACT_REG *regexp.Regexp
var DATE_EXCEL_REG *regexp.Regexp
var ISO_TIME_REG *regexp.Regexp
var ISO_NO_SECOND_TIME_REG *regexp.Regexp
var FULLISO_TIME_REG *regexp.Regexp
var ISO_TIME_REG2 *regexp.Regexp
var ISO_NO_SECOND_TIME_REG2 *regexp.Regexp
var FULLISO_TIME_REG2 *regexp.Regexp
var FULLISO_TIME_REG3 *regexp.Regexp
var ZSTACK_TIME_REG *regexp.Regexp
var COMPACT_TIME_REG *regexp.Regexp
var MYSQL_TIME_REG *regexp.Regexp
var CLICKHOUSE_TIME_REG *regexp.Regexp
var NORMAL_TIME_REG *regexp.Regexp
var FULLNORMAL_TIME_REG *regexp.Regexp
var RFC2882_TIME_REG *regexp.Regexp
var CEPH_TIME_REG *regexp.Regexp
var EMAIL_REG *regexp.Regexp
var CHINA_MOBILE_REG *regexp.Regexp
var FS_FORMAT_REG *regexp.Regexp
var US_CURRENCY_REG *regexp.Regexp
var EU_CURRENCY_REG *regexp.Regexp

func init() {
	FUNCTION_REG = regexp.MustCompile(`^\w+\(.*\)$`)
	UUID_REG = regexp.MustCompile(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`)
	UUID_EXACT_REG = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
	INTEGER_REG = regexp.MustCompile(`^[0-9]+$`)
	FLOAT_REG = regexp.MustCompile(`^\d+(\.\d*)?$`)
	MACADDR_REG = regexp.MustCompile(`^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$`)
	COMPACT_MACADDR_REG = regexp.MustCompile(`^[0-9a-fA-F]{12}$`)
	NSPTR_REG = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\.in-addr\.arpa$`)
	NAME_REG = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9._@-]*$`)
	DOMAINNAME_REG = regexp.MustCompile(`^([a-zA-Z0-9_]{1}[a-zA-Z0-9_-]{0,62}){1}(\.[a-zA-Z0-9_]{1}[a-zA-Z0-9_-]{0,62})*[\._]?$`)
	SIZE_REG = regexp.MustCompile(`^\d+[bBkKmMgG]?$`)
	MONTH_REG = regexp.MustCompile(`^\d{4}-\d{2}$`)
	DATE_REG = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	DATE_COMPACT_REG = regexp.MustCompile(`^\d{8}$`)
	DATE_EXCEL_REG = regexp.MustCompile(`^\d{2}-\d{2}-\d{2}$`)
	ISO_TIME_REG = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(Z|[+-]\d{2}:\d{2})$`)
	ISO_NO_SECOND_TIME_REG = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(Z|[+-]\d{2}:\d{2})$`)
	FULLISO_TIME_REG = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3,9}(Z|[+-]\d{2}:\d{2})$`)
	ISO_TIME_REG2 = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(Z|[+-]\d{2}:\d{2})$`)
	ISO_NO_SECOND_TIME_REG2 = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}(Z|[+-]\d{2}:\d{2})$`)
	FULLISO_TIME_REG2 = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3,9}(Z|[+-]\d{2}:\d{2})$`)
	FULLISO_TIME_REG3 = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3,9}$`)
	COMPACT_TIME_REG = regexp.MustCompile(`^\d{14}$`)
	ZSTACK_TIME_REG = regexp.MustCompile(`^\w+ \d{1,2}, \d{4} \d{1,2}:\d{1,2}:\d{1,2} (AM|PM)$`) //ZStack time format "Apr 1, 2019 3:23:17 PM"
	MYSQL_TIME_REG = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`)
	CLICKHOUSE_TIME_REG = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} [+-]\d{4} [A-Z]{3}$`)
	NORMAL_TIME_REG = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}$`)
	FULLNORMAL_TIME_REG = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}$`)
	RFC2882_TIME_REG = regexp.MustCompile(`[A-Z][a-z]{2}, [0-9]{1,2} [A-Z][a-z]{2} [0-9]{4} [0-9]{2}:[0-9]{2}:[0-9]{2} [A-Z]{3}`)
	// Tue May  7 15:46:33 2024
	CEPH_TIME_REG = regexp.MustCompile(`[A-Z][a-z]{2} [A-Z][a-z]{2} [ 123][0-9] [0-9]{2}:[0-9]{2}:[0-9]{2} [0-9]{4}`)
	EMAIL_REG = regexp.MustCompile(`^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,4}$`)
	CHINA_MOBILE_REG = regexp.MustCompile(`^1[0-9-]{10}$`)
	FS_FORMAT_REG = regexp.MustCompile(`^(ext|fat|hfs|xfs|swap|ntfs|reiserfs|ufs|btrfs)`)
	US_CURRENCY_REG = regexp.MustCompile(`^[+-]?(\d{0,3}|((\d{1,3},)+\d{3}))(\.\d*)?$`)
	EU_CURRENCY_REG = regexp.MustCompile(`^[+-]?(\d{0,3}|((\d{1,3}\.)+\d{3}))(,\d*)?$`)
}

func MatchFunction(str string) bool {
	return FUNCTION_REG.MatchString(str)
}

func MatchUUID(str string) bool {
	return UUID_REG.MatchString(str)
}

func MatchUUIDExact(str string) bool {
	return UUID_EXACT_REG.MatchString(str)
}

func MatchInteger(str string) bool {
	return INTEGER_REG.MatchString(str)
}

func MatchFloat(str string) bool {
	return FLOAT_REG.MatchString(str)
}

func MatchMacAddr(str string) bool {
	return MACADDR_REG.MatchString(str)
}

func MatchCompactMacAddr(str string) bool {
	return COMPACT_MACADDR_REG.MatchString(str)
}

func MatchIP4Addr(str string) bool {
	ip := net.ParseIP(str)
	return ip != nil && !strings.Contains(str, ":")
}

func MatchCIDR(str string) bool {
	ip, _, err := net.ParseCIDR(str)
	if err != nil {
		return false
	}
	return ip != nil && !strings.Contains(str, ":")
}

func MatchCIDR6(str string) bool {
	ip, _, err := net.ParseCIDR(str)
	if err != nil {
		return false
	}
	return ip != nil && !strings.Contains(str, ".")
}

func MatchIP6Addr(str string) bool {
	ip := net.ParseIP(str)
	return ip != nil && strings.Contains(str, ":")
}

func MatchIPAddr(str string) bool {
	ip := net.ParseIP(str)
	return ip != nil
}

func MatchPtr(str string) bool {
	return NSPTR_REG.MatchString(str)
}

func MatchName(str string) bool {
	return NAME_REG.MatchString(str)
}

func MatchDomainName(str string) bool {
	if str == "" || len(strings.Replace(str, ".", "", -1)) > 255 {
		return false
	}
	return !MatchIPAddr(str) && DOMAINNAME_REG.MatchString(str)
}

func MatchDomainSRV(str string) bool {
	if !MatchDomainName(str) {
		return false
	}

	// Ref: https://tools.ietf.org/html/rfc2782
	//
	//	_Service._Proto.Name
	parts := strings.SplitN(str, ".", 3)
	if len(parts) != 3 {
		return false
	}
	for i := 0; i < 2; i++ {
		if len(parts[i]) < 2 || parts[i][0] != '_' {
			return false
		}
	}
	if len(parts[2]) == 0 {
		return false
	}
	return true
}

func MatchSize(str string) bool {
	return SIZE_REG.MatchString(str)
}

func MatchMonth(str string) bool {
	return MONTH_REG.MatchString(str)
}

func MatchDate(str string) bool {
	return DATE_REG.MatchString(str)
}

func MatchDateCompact(str string) bool {
	return DATE_COMPACT_REG.MatchString(str)
}

func MatchDateExcel(str string) bool {
	return DATE_EXCEL_REG.MatchString(str)
}

func MatchZStackTime(str string) bool {
	return ZSTACK_TIME_REG.MatchString(str)
}

func MatchISOTime(str string) bool {
	return ISO_TIME_REG.MatchString(str)
}

func MatchISONoSecondTime(str string) bool {
	return ISO_NO_SECOND_TIME_REG.MatchString(str)
}

func MatchFullISOTime(str string) bool {
	return FULLISO_TIME_REG.MatchString(str)
}

func MatchISOTime2(str string) bool {
	return ISO_TIME_REG2.MatchString(str)
}

func MatchISONoSecondTime2(str string) bool {
	return ISO_NO_SECOND_TIME_REG2.MatchString(str)
}

func MatchFullISOTime2(str string) bool {
	return FULLISO_TIME_REG2.MatchString(str)
}

func MatchFullISOTime3(str string) bool {
	return FULLISO_TIME_REG3.MatchString(str)
}

func MatchCompactTime(str string) bool {
	return COMPACT_TIME_REG.MatchString(str)
}

func MatchMySQLTime(str string) bool {
	return MYSQL_TIME_REG.MatchString(str)
}

func MatchClickhouseTime(str string) bool {
	return CLICKHOUSE_TIME_REG.MatchString(str)
}

func MatchNormalTime(str string) bool {
	return NORMAL_TIME_REG.MatchString(str)
}

func MatchFullNormalTime(str string) bool {
	return FULLNORMAL_TIME_REG.MatchString(str)
}

func MatchRFC2882Time(str string) bool {
	return RFC2882_TIME_REG.MatchString(str)
}

func MatchCephTime(str string) bool {
	return CEPH_TIME_REG.MatchString(str)
}

func MatchEmail(str string) bool {
	return EMAIL_REG.MatchString(str)
}

func MatchMobile(str string) bool {
	return CHINA_MOBILE_REG.MatchString(str)
}

func MatchFS(str string) bool {
	return FS_FORMAT_REG.MatchString(str)
}

func MatchUSCurrency(str string) bool {
	return US_CURRENCY_REG.MatchString(str)
}

func MatchEUCurrency(str string) bool {
	return EU_CURRENCY_REG.MatchString(str)
}
