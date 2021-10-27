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

package mysql

import (
	"fmt"
	"math/bits"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/sqlchemy"
)

type sSqlColumnInfo struct {
	Field      string
	Type       string
	Collation  string
	Null       string
	Key        string
	Default    string
	Extra      string
	Privileges string
	Comment    string
}

func decodeSqlTypeString(typeStr string) []string {
	typeReg := regexp.MustCompile(`(\w+)\((\d+)(,\s*(\d+))?\)`)
	matches := typeReg.FindStringSubmatch(typeStr)
	if len(matches) >= 3 {
		return matches[1:]
	}
	parts := strings.Split(typeStr, " ")
	return []string{parts[0]}
}

func (info *sSqlColumnInfo) toColumnSpec() sqlchemy.IColumnSpec {
	tagmap := make(map[string]string)

	matches := decodeSqlTypeString(info.Type)
	typeStr := strings.ToUpper(matches[0])
	width := 0
	if len(matches) > 1 {
		width, _ = strconv.Atoi(matches[1])
	}
	if width > 0 {
		tagmap[sqlchemy.TAG_WIDTH] = fmt.Sprintf("%d", width)
	}
	if info.Null == "YES" {
		tagmap[sqlchemy.TAG_NULLABLE] = "true"
	} else {
		tagmap[sqlchemy.TAG_NULLABLE] = "false"
	}
	if info.Key == "PRI" {
		tagmap[sqlchemy.TAG_PRIMARY] = "true"
	} else {
		tagmap[sqlchemy.TAG_PRIMARY] = "false"
	}
	charset := ""
	if info.Collation == "ascii_general_ci" {
		charset = "ascii"
	} else if info.Collation == "utf8_general_ci" || info.Collation == "utf8mb4_unicode_ci" {
		charset = "utf8"
	} else {
		charset = "ascii"
	}
	if len(charset) > 0 {
		tagmap[sqlchemy.TAG_CHARSET] = charset
	}
	if info.Default != "NULL" {
		tagmap[sqlchemy.TAG_DEFAULT] = info.Default
	}
	if strings.HasSuffix(typeStr, "CHAR") {
		c := NewTextColumn(info.Field, typeStr, tagmap, false)
		return &c
	} else if strings.HasSuffix(typeStr, "TEXT") {
		tagmap[sqlchemy.TAG_TEXT_LENGTH] = typeStr[:len(typeStr)-4]
		c := NewTextColumn(info.Field, typeStr, tagmap, false)
		return &c
	} else if strings.HasSuffix(typeStr, "INT") {
		if info.Extra == "auto_increment" {
			tagmap[sqlchemy.TAG_AUTOINCREMENT] = "true"
		}
		unsigned := false
		if strings.HasSuffix(info.Type, " unsigned") {
			unsigned = true
		}
		if _, ok := tagmap[sqlchemy.TAG_WIDTH]; !ok {
			if unsigned {
				tagmap[sqlchemy.TAG_WIDTH] = uintWidthString(typeStr)
			} else {
				tagmap[sqlchemy.TAG_WIDTH] = intWidthString(typeStr)
			}
		}
		c := NewIntegerColumn(info.Field, typeStr, unsigned, tagmap, false)
		return &c
	} else if typeStr == "FLOAT" || typeStr == "DOUBLE" {
		c := NewFloatColumn(info.Field, typeStr, tagmap, false)
		return &c
	} else if typeStr == "DECIMAL" {
		if len(matches) > 3 {
			precision, _ := strconv.Atoi(matches[3])
			if precision > 0 {
				tagmap[sqlchemy.TAG_PRECISION] = fmt.Sprintf("%d", precision)
			}
		}
		c := NewDecimalColumn(info.Field, tagmap, false)
		return &c
	} else if typeStr == "DATETIME" {
		c := NewDateTimeColumn(info.Field, tagmap, false)
		return &c
	} else if typeStr == "DATE" || typeStr == "TIMESTAMP" {
		c := NewTimeTypeColumn(info.Field, typeStr, tagmap, false)
		return &c
	} else if strings.HasPrefix(typeStr, "ENUM(") {
		// enum type, force convert to text
		// discourage use of enum, use text instead
		enums := utils.FindWords([]byte(typeStr[5:len(typeStr)-1]), 0)

		width := 0
		for i := range enums {
			if width < len(enums[i]) {
				width = len(enums[i])
			}
		}
		tagmap[sqlchemy.TAG_WIDTH] = fmt.Sprintf("%d", 1<<uint(bits.Len(uint(width))))
		c := NewTextColumn(info.Field, "VARCHAR", tagmap, false)
		return &c
	} else {
		log.Errorf("unsupported type %s", typeStr)
		return nil
	}
}
