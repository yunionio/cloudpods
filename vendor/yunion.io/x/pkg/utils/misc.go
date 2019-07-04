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

package utils

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type selectFunc func(obj interface{}) (string, error)

func ToDict(objs interface{}, ks selectFunc) (map[string]interface{}, error) {
	s := reflect.ValueOf(objs)
	if s.Kind() != reflect.Slice {
		return nil, fmt.Errorf("Not slice")
	}
	res := map[string]interface{}{}
	for i := 0; i < s.Len(); i++ {
		obj := s.Index(i).Interface()
		key, err := ks(obj)
		if err != nil {
			return nil, err
		}
		res[key] = obj
	}
	return res, nil
}

func GroupBy(items interface{}, ks selectFunc) (map[string][]interface{}, error) {
	s := reflect.ValueOf(items)
	if s.Kind() != reflect.Slice {
		return nil, fmt.Errorf("Not slice")
	}
	res := map[string][]interface{}{}
	for i := 0; i < s.Len(); i++ {
		obj := s.Index(i).Interface()
		key, err := ks(obj)
		if err != nil {
			return nil, err
		}
		values, ok := res[key]
		if !ok {
			values = []interface{}{}
		}
		values = append(values, obj)
		res[key] = values
	}
	return res, nil
}

func SelectDistinct(items []interface{}, ks selectFunc) ([]string, error) {
	keyMap := make(map[string]interface{})
	for _, item := range items {
		key, err := ks(item)
		if err != nil {
			return nil, err
		}
		keyMap[key] = item
	}

	keys := []string{}
	for key := range keyMap {
		keys = append(keys, key)
	}

	return keys, nil
}

func SubDict(dict map[string][]interface{}, keys ...string) (map[string][]interface{}, error) {
	if len(keys) == 0 {
		return dict, nil
	}
	res := make(map[string][]interface{})
	for _, key := range keys {
		if value, ok := dict[key]; ok {
			res[key] = value
		}
	}
	return res, nil
}

type StatItem2 interface {
	First() string
	Second() interface{}
}

type StatItem3 interface {
	First() string
	Second() string
	Third() interface{}
}

func ToStatDict2(items []StatItem2) (map[string]interface{}, error) {
	res := make(map[string]interface{})
	if len(items) == 0 {
		return res, nil
	}
	for _, item := range items {
		res[item.First()] = item.Second()
	}
	return res, nil
}

func ToStatDict3(items []StatItem3) (map[string]map[string]interface{}, error) {
	res := make(map[string]map[string]interface{})
	if len(items) == 0 {
		return res, nil
	}
	for _, item := range items {
		d1, ok := res[item.First()]
		if !ok {
			d1 = make(map[string]interface{})
			res[item.First()] = d1
		}
		d1[item.Second()] = item.Third()
	}
	return res, nil
}

func ConvertError(obj interface{}, toType string) error {
	return fmt.Errorf("Type cast error: %#v => %s", obj, toType)
}

func HasPrefix(s string, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

func HasSuffix(s string, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func IsMatch(s string, pattern string) bool {
	success, _ := regexp.MatchString(pattern, s)
	return success
}

func IsMatchIP4(s string) bool {
	return IsMatch(s, "^\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}$")
}

func IsMatchIP6(s string) bool {
	return IsMatch(s, "^\\s*((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])(.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])(.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])(.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])(.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])(.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])(.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])(.(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])){3}))|:)))(%.+)?i\\s*$")
}

func IsMatchCompactMacAddr(s string) bool {
	return IsMatch(s, "^[0-9a-fA-F]{12}$")
}

func IsMatchMacAddr(s string) bool {
	return IsMatch(s, "^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$")
}

func IsMatchSize(s string) bool {
	return IsMatch(s, "^\\d+[bBkKmMgG]?$")
}

func IsMatchUUID(s string) bool {
	return IsMatch(s, "^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$")
}

func IsMatchInteger(s string) bool {
	return IsMatch(s, "^[0-9]+$")
}

func IsMatchFloat(s string) bool {
	return IsMatch(s, "^\\d+(\\.\\d*)?$")
}

func Max(a int64, b int64) int64 {
	if a > b {
		return a
	}

	return b
}

func Min(a int64, b int64) int64 {
	if a < b {
		return a
	}

	return b
}

type Pred_t func(interface{}) bool

func Any(pred Pred_t, items ...interface{}) (bool, interface{}) {
	for _, it := range items {
		if pred(it) {
			return true, it
		}
	}

	return false, nil
}

func All(pred Pred_t, items ...interface{}) (bool, interface{}) {
	for _, it := range items {
		if !pred(it) {
			return false, it
		}
	}

	return true, nil
}

func IsLocalStorage(storageType string) bool {
	s := storageType
	return s == "local" || s == "baremetal" || s == "raw" ||
		s == "docker" || s == "volume" || s == "lvm"
}

func DistinctJoin(list []string, separator string) string {
	return strings.Join(Distinct(list), separator)
}

func Distinct(list []string) []string {

	ss := []string{}
	ssMap := make(map[string]int)

	for _, s := range list {
		if _, ok := ssMap[s]; !ok {
			ssMap[s] = 0
			ss = append(ss, s)
		}
	}

	return ss
}

func Ip2Int(ipString string) uint32 {
	ip := net.ParseIP(ipString)
	if ip == nil {
		return 0
	}
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

func IpRangeCount(ipStart, ipEnd string) int {
	return int(Ip2Int(ipEnd) - Ip2Int(ipStart) + 1)
}

var (
	PrivateIP1Start = Ip2Int("10.0.0.0")
	PrivateIP1End   = Ip2Int("10.255.255.255")
	PrivateIP2Start = Ip2Int("172.16.0.0")
	PrivateIP2End   = Ip2Int("172.31.255.255")
	PrivateIP3Start = Ip2Int("192.168.0.0")
	PrivateIP3End   = Ip2Int("192.168.255.255")

	HostLocalIPStart = Ip2Int("127.0.0.0")
	HostLocalIPEnd   = Ip2Int("127.255.255.255")

	LinkLocalIPStart = Ip2Int("169.254.0.0")
	LinkLocalIPEnd   = Ip2Int("169.254.255.255")
)

func IsPrivate(ip uint32) bool {
	return (PrivateIP1Start <= ip && ip <= PrivateIP1End) ||
		(PrivateIP2Start <= ip && ip <= PrivateIP3End) ||
		(PrivateIP3Start <= ip && ip <= PrivateIP3End)
}

func IsHostLocal(ip uint32) bool {
	return HostLocalIPStart <= ip && ip <= HostLocalIPEnd
}

func IsLinkLocal(ip uint32) bool {
	return LinkLocalIPStart <= ip && ip <= LinkLocalIPEnd
}

func IsExitAddress(ip string) bool {
	ipUint32 := Ip2Int(ip)
	if ipUint32 == 0 {
		return false
	}
	return !(IsPrivate(ipUint32) || IsHostLocal(ipUint32) || IsLinkLocal(ipUint32))
}

func Truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}

	return s[0:length] + "..."
}

// From html/template/content.go
// Copyright 2011 The Go Authors. All rights reserved.
// indirect returns the value, after dereferencing as many times
// as necessary to reach the base type (or nil).
func indirect(a interface{}) interface{} {
	if a == nil {
		return nil
	}
	if t := reflect.TypeOf(a); t.Kind() != reflect.Ptr {
		// Avoid creating a reflect.Value if it's not a pointer.
		return a
	}
	v := reflect.ValueOf(a)
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	return v.Interface()
}

// ToInt64E casts an interface to an int64 type.
func ToInt64E(i interface{}) (int64, error) {
	i = indirect(i)

	switch s := i.(type) {
	case int:
		return int64(s), nil
	case int64:
		return s, nil
	case int32:
		return int64(s), nil
	case int16:
		return int64(s), nil
	case int8:
		return int64(s), nil
	case uint:
		return int64(s), nil
	case uint64:
		return int64(s), nil
	case uint32:
		return int64(s), nil
	case uint16:
		return int64(s), nil
	case uint8:
		return int64(s), nil
	case float64:
		return int64(s), nil
	case float32:
		return int64(s), nil
	case string:
		v, err := strconv.ParseInt(s, 0, 0)
		if err == nil {
			return v, nil
		}
		return 0, fmt.Errorf("unable to cast %#v of type %T to int64", i, i)
	case bool:
		if s {
			return 1, nil
		}
		return 0, nil
	case nil:
		return 0, nil
	default:
		return 0, fmt.Errorf("unable to cast %#v of type %T to int64", i, i)
	}
}

// ToInt64 casts an interface to an int64 type.
func ToInt64(i interface{}) int64 {
	v, _ := ToInt64E(i)
	return v
}

// ToFloat64 casts an interface to a float64 type.
func ToFloat64(i interface{}) float64 {
	v, _ := ToFloat64E(i)
	return v
}

// ToFloat64E casts an interface to a float64 type.
func ToFloat64E(i interface{}) (float64, error) {
	i = indirect(i)

	switch s := i.(type) {
	case float64:
		return s, nil
	case float32:
		return float64(s), nil
	case int:
		return float64(s), nil
	case int64:
		return float64(s), nil
	case int32:
		return float64(s), nil
	case int16:
		return float64(s), nil
	case int8:
		return float64(s), nil
	case uint:
		return float64(s), nil
	case uint64:
		return float64(s), nil
	case uint32:
		return float64(s), nil
	case uint16:
		return float64(s), nil
	case uint8:
		return float64(s), nil
	case string:
		v, err := strconv.ParseFloat(s, 64)
		if err == nil {
			return v, nil
		}
		return 0, fmt.Errorf("unable to cast %#v of type %T to float64", i, i)
	case bool:
		if s {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("unable to cast %#v of type %T to float64", i, i)
	}
}

func ToDurationE(i interface{}) (d time.Duration, err error) {
	i = indirect(i)
	switch s := i.(type) {
	case time.Duration:
		return s, nil
	case int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8:
		d = time.Duration(ToInt64(s))
		return
	case float32, float64:
		d = time.Duration(ToFloat64(s))
		return
	case string:
		if strings.ContainsAny(s, "nsuµmh") {
			d, err = time.ParseDuration(s)
		} else {
			d, err = time.ParseDuration(s + "ns")
		}
		return
	default:
		err = fmt.Errorf("unable to cast %#v of type %T to Duration", i, i)
		return
	}
}

func ToDuration(i interface{}) time.Duration {
	v, _ := ToDurationE(i)
	return v
}

// GetSize parse size string to int
// defaultSize be used when sizeStr not end with defaultSize
// sizeStr: 1024, 1M, 1m, 1K etc.
// defaultSize: g, m, k, b etc.
// base: base multiple unit, 1024
func GetSize(sizeStr, defaultSize string, base int64) (size int64, err error) {
	if IsMatchInteger(sizeStr) {
		sizeStr += defaultSize
	}

	sizeNumStr := sizeStr[0 : len(sizeStr)-1]
	size, err = strconv.ParseInt(sizeNumStr, 10, 64)
	if err != nil {
		return
	}

	switch u := sizeStr[len(sizeStr)-1]; u {

	case 't', 'T':
		size = size * base * base * base * base

	case 'g', 'G':
		size = size * base * base * base

	case 'm', 'M':
		size = size * base * base

	case 'k', 'K':
		size = size * base

	case 'b', 'B':
		size = size

	default:
		err = fmt.Errorf("Incorrect unit %q", u)
	}

	return
}

func GetSizeBytes(sizeStr, defaultSize string) (int64, error) {
	return GetSize(sizeStr, defaultSize, 1024)
}

func GetBytes(sizeStr string) (int64, error) {
	if IsMatchInteger(sizeStr) {
		return 0, fmt.Errorf("Please append suffix unit like '[g, m, k, b]' to %q", sizeStr)
	}
	return GetSize(sizeStr, "", 1024)
}

func GetSizeGB(sizeStr, defaultSize string) (int64, error) {
	bytes, err := GetSizeBytes(sizeStr, defaultSize)
	if err != nil {
		return 0, err
	}
	return bytes / 1024 / 1024 / 1024, nil
}

func GetSizeMB(sizeStr, defaultSize string) (int64, error) {
	bytes, err := GetSizeBytes(sizeStr, defaultSize)
	if err != nil {
		return 0, err
	}
	return bytes / 1024 / 1024, nil
}

func GetSizeKB(sizeStr, defaultSize string) (int64, error) {
	bytes, err := GetSizeBytes(sizeStr, defaultSize)
	if err != nil {
		return 0, err
	}
	return bytes / 1024, nil
}

func TransSQLAchemyURL(pySQLSrc string) (dialect, ret string, err error) {
	if len(pySQLSrc) == 0 {
		err = fmt.Errorf("Empty input")
		return
	}

	dialect = "mysql"
	if !strings.Contains(pySQLSrc, `//`) {
		return dialect, pySQLSrc, nil
	}

	r := regexp.MustCompile(`[/@:]+`)
	strs := r.Split(pySQLSrc, -1)
	if len(strs) != 6 {
		err = fmt.Errorf("Incorrect mysql connection url: %s", pySQLSrc)
		return
	}
	user, passwd, host, port, dburl := strs[1], strs[2], strs[3], strs[4], strs[5]
	queryPos := strings.IndexByte(dburl, '?')
	if queryPos == 0 {
		err = fmt.Errorf("Missing database name")
		return
	}
	var query url.Values
	if queryPos > 0 {
		queryStr := dburl[queryPos+1:]
		if len(queryStr) > 0 {
			query, err = url.ParseQuery(queryStr)
			if err != nil {
				return
			}
		}
		dburl = dburl[:queryPos]
	} else {
		query = url.Values{}
	}
	query.Set("parseTime", "True")
	ret = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s", user, passwd, host, port, dburl, query.Encode())
	return
}

func ComposeURL(paths ...string) string {
	if len(paths) == 0 || len(paths[0]) == 0 {
		return ""
	}
	restURL := ComposeURL(paths[1:]...)
	if len(restURL) == 0 {
		return fmt.Sprintf("/%s", paths[0])
	}
	return fmt.Sprintf("/%s%s", paths[0], ComposeURL(paths[1:]...))
}
