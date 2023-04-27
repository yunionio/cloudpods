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

package clickhouse

import (
	"regexp"
	"sort"
	"strings"

	"yunion.io/x/log"

	"yunion.io/x/sqlchemy"
)

// name type default_type default_expression comment codec_expression ttl_expression

type sSqlColumnInfo struct {
	Name              string `json:"name"`
	Type              string `json:"type"`
	DefaultType       string `json:"default_type"`
	DefaultExpression string `json:"default_expression"`
	Comment           string `json:"comment"`
	CodecExpression   string `json:"codec_expression"`
	TtlExpression     string `json:"ttl_expression"`
}

func (info *sSqlColumnInfo) isNullable() bool {
	if strings.HasPrefix(info.Type, "Nullable(") {
		return true
	} else {
		return false
	}
}

func (info *sSqlColumnInfo) getType() string {
	if strings.HasPrefix(info.Type, "Nullable(") {
		return info.Type[len("Nullable(") : len(info.Type)-1]
	} else {
		return info.Type
	}
}

func (info *sSqlColumnInfo) getDefault() string {
	if info.DefaultType == "DEFAULT" {
		if strings.HasPrefix(info.DefaultExpression, "CAST(") {
			defaultVals := strings.Split(info.DefaultExpression[len("CAST("):len(info.DefaultExpression)-1], ",")
			defaultVal := defaultVals[0]
			typeStr := info.getType()
			if typeStr == "String" || strings.HasPrefix(typeStr, "FixString") {
				defaultVal = defaultVal[1 : len(defaultVal)-1]
			}
			return defaultVal
		} else {
			return info.DefaultExpression
		}
	}
	return ""
}

func (info *sSqlColumnInfo) getTagmap() map[string]string {
	tagmap := make(map[string]string)
	if info.isNullable() {
		tagmap[sqlchemy.TAG_NULLABLE] = "true"
	} else {
		tagmap[sqlchemy.TAG_NULLABLE] = "false"
	}
	defVal := info.getDefault()
	if len(defVal) > 0 {
		if info.getType() == "String" && defVal[0] == '\'' {
			defVal = defVal[1 : len(defVal)-1]
		}
		tagmap[sqlchemy.TAG_DEFAULT] = defVal
	}
	return tagmap
}

func (info *sSqlColumnInfo) toColumnSpec() sqlchemy.IColumnSpec {
	sqlType := info.getType()
	switch sqlType {
	case "String":
		c := NewTextColumn(info.Name, sqlType, info.getTagmap(), false)
		return &c
	case "Int8", "Int16", "Int32", "Int64", "UInt8", "UInt16", "UInt32", "UInt64":
		c := NewIntegerColumn(info.Name, sqlType, info.getTagmap(), false)
		return &c
	case "Float32", "Float64":
		c := NewFloatColumn(info.Name, sqlType, info.getTagmap(), false)
		return &c
	case "DateTime", "DateTime('UTC')":
		c := NewDateTimeColumn(info.Name, info.getTagmap(), false)
		return &c
	default:
		if strings.HasPrefix(sqlType, "Decimal") {
			c := NewDecimalColumn(info.Name, info.getTagmap(), false)
			return &c
		} else if strings.HasPrefix(sqlType, "FixString") {
			c := NewTextColumn(info.Name, "FixString", info.getTagmap(), false)
			return &c
		}
		log.Errorf("unsupported type %s", info.Type)
	}
	return nil
}

const (
	primaryKeyPrefix  = "PRIMARY KEY "
	orderByPrefix     = "ORDER BY "
	partitionByPrefix = "PARTITION BY "
	setttingsPrefix   = "SETTINGS"
	ttlPrefix         = "TTL "

	paramPattern      = `(\w+|\([\w,\s]+\))`
	primaryKeyPattern = primaryKeyPrefix + paramPattern
	orderByPattern    = orderByPrefix + paramPattern
)

var (
	primaryKeyRegexp = regexp.MustCompile(primaryKeyPattern)
	orderByRegexp    = regexp.MustCompile(orderByPattern)
)

func parseKeys(keyStr string) []string {
	keyStr = strings.TrimSpace(keyStr)
	if keyStr[0] == '(' {
		keyStr = keyStr[1 : len(keyStr)-1]
	}
	ret := make([]string, 0)
	for _, key := range strings.Split(keyStr, ",") {
		key = strings.TrimSpace(key)
		ret = append(ret, key)
	}
	sort.Strings(ret)
	return ret
}

func findSegment(sqlStr string, prefix string) string {
	partIdx := strings.Index(sqlStr, prefix)
	if partIdx > 0 {
		partIdx += len(prefix)
		nextIdx := -1
		for _, pattern := range []string{partitionByPrefix, primaryKeyPrefix, orderByPrefix, setttingsPrefix, ttlPrefix} {
			idx := strings.Index(sqlStr[partIdx:], pattern)
			if idx > 0 && (nextIdx < 0 || nextIdx > idx) {
				nextIdx = idx
			}
		}
		if nextIdx < 0 {
			return strings.TrimSpace(sqlStr[partIdx:])
		} else {
			return strings.TrimSpace(sqlStr[partIdx:][:nextIdx])
		}
	}
	return ""
}

func trimPartition(partStr string) string {
	for {
		partStr = strings.TrimSpace(partStr)
		if partStr[0] == '(' {
			partStr = partStr[1 : len(partStr)-1]
		} else {
			break
		}
	}
	partStr = strings.ReplaceAll(partStr, " ", "")
	return partStr
}

func parsePartitions(partStr string) []string {
	partStr = trimPartition(partStr)
	parts := strings.Split(partStr, ",")
	sort.Strings(parts)
	return parts
}

func parseCreateTable(sqlStr string) (primaries []string, orderbys []string, partitions []string, ttl string) {
	matches := primaryKeyRegexp.FindAllStringSubmatch(sqlStr, -1)
	if len(matches) > 0 {
		primaries = parseKeys(matches[0][1])
	}
	matches = orderByRegexp.FindAllStringSubmatch(sqlStr, -1)
	if len(matches) > 0 {
		orderbys = parseKeys(matches[0][1])
	}
	partitionStr := findSegment(sqlStr, partitionByPrefix)
	partitions = parsePartitions(partitionStr)
	ttl = findSegment(sqlStr, ttlPrefix)
	return
}
