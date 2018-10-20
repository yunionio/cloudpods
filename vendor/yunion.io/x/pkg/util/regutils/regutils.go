package regutils

import (
	"regexp"
)

var FUNCTION_REG *regexp.Regexp
var UUID_REG *regexp.Regexp
var UUID_EXACT_REG *regexp.Regexp
var INTEGER_REG *regexp.Regexp
var FLOAT_REG *regexp.Regexp
var MACADDR_REG *regexp.Regexp
var COMPACT_MACADDR_REG *regexp.Regexp
var IPADDR_REG_PATTERN *regexp.Regexp
var IP6ADDR_REG *regexp.Regexp
var CIDR_REG_PATTERN *regexp.Regexp
var NSPTR_REG *regexp.Regexp
var NAME_REG *regexp.Regexp
var DOMAINNAME_REG *regexp.Regexp
var DOMAINSRV_REG *regexp.Regexp
var SIZE_REG *regexp.Regexp
var MONTH_REG *regexp.Regexp
var DATE_REG *regexp.Regexp
var DATE_COMPACT_REG *regexp.Regexp
var ISO_TIME_REG *regexp.Regexp
var ISO_NO_SECOND_TIME_REG *regexp.Regexp
var FULLISO_TIME_REG *regexp.Regexp
var COMPACT_TIME_REG *regexp.Regexp
var MYSQL_TIME_REG *regexp.Regexp
var NORMAL_TIME_REG *regexp.Regexp
var RFC2882_TIME_REG *regexp.Regexp
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
	IPADDR_REG_PATTERN = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)
	CIDR_REG_PATTERN = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}(\/\d{1,2})?$`)
	IP6ADDR_REG = regexp.MustCompile(`^\s*((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])(.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])(.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])(.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])(.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])(.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])(.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])(.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])){3}))|:)))(%.+)?\s*$`)
	NSPTR_REG = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\.in-addr\.arpa$`)
	NAME_REG = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9._@-]*$`)
	DOMAINNAME_REG = regexp.MustCompile(`^[a-zA-Z0-9-.]+$`)
	DOMAINSRV_REG = regexp.MustCompile(`^[a-zA-Z0-9-._]+$`)
	SIZE_REG = regexp.MustCompile(`^\d+[bBkKmMgG]?$`)
	MONTH_REG = regexp.MustCompile(`^\d{4}-\d{2}$`)
	DATE_REG = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	DATE_COMPACT_REG = regexp.MustCompile(`^\d{8}$`)
	ISO_TIME_REG = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`)
	ISO_NO_SECOND_TIME_REG = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}Z$`)
	FULLISO_TIME_REG = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$`)
	COMPACT_TIME_REG = regexp.MustCompile(`^\d{14}$`)
	MYSQL_TIME_REG = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`)
	NORMAL_TIME_REG = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}$`)
	RFC2882_TIME_REG = regexp.MustCompile(`[A-Z][a-z]{2}, [0-9]{1,2} [A-Z][a-z]{2} [0-9]{4} [0-9]{2}:[0-9]{2}:[0-9]{2} [A-Z]{3}`)
	EMAIL_REG = regexp.MustCompile(`^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,4}$`)
	CHINA_MOBILE_REG = regexp.MustCompile(`^1[0-9-]{10}$`)
	FS_FORMAT_REG = regexp.MustCompile(`^(ext|fat|hfs|xfs|swap|ntfs|reiserfs|ufs|btrfs)`)
	US_CURRENCY_REG = regexp.MustCompile(`^(\d{0,3}|((\d{1,3},)+\d{3}))(\.\d*)?$`)
	EU_CURRENCY_REG = regexp.MustCompile(`^(\d{0,3}|((\d{1,3}\.)+\d{3}))(,\d*)?$`)
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
	return IPADDR_REG_PATTERN.MatchString(str)
}

func MatchCIDR(str string) bool {
	return CIDR_REG_PATTERN.MatchString(str)
}

func MatchIP6Addr(str string) bool {
	return IP6ADDR_REG.MatchString(str)
}

func MatchIPAddr(str string) bool {
	return MatchIP4Addr(str) || MatchIP6Addr(str)
}

func MatchPtr(str string) bool {
	return NSPTR_REG.MatchString(str)
}

func MatchName(str string) bool {
	return NAME_REG.MatchString(str)
}

func MatchDomainName(str string) bool {
	return DOMAINNAME_REG.MatchString(str)
}

func MatchDomainSRV(str string) bool {
	return DOMAINSRV_REG.MatchString(str)
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

func MatchISOTime(str string) bool {
	return ISO_TIME_REG.MatchString(str)
}

func MatchISONoSecondTime(str string) bool {
	return ISO_NO_SECOND_TIME_REG.MatchString(str)
}

func MatchFullISOTime(str string) bool {
	return FULLISO_TIME_REG.MatchString(str)
}

func MatchCompactTime(str string) bool {
	return COMPACT_TIME_REG.MatchString(str)
}

func MatchMySQLTime(str string) bool {
	return MYSQL_TIME_REG.MatchString(str)
}

func MatchNormalTime(str string) bool {
	return NORMAL_TIME_REG.MatchString(str)
}

func MatchRFC2882Time(str string) bool {
	return RFC2882_TIME_REG.MatchString(str)
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
