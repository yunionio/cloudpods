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

package sqlchemy

import (
	"strconv"
	"strings"
)

const (
	SQL_OP_AND      = "AND"
	SQL_OP_OR       = "OR"
	SQL_OP_NOT      = "NOT"
	SQL_OP_LIKE     = "LIKE"
	SQL_OP_IN       = "IN"
	SQL_OP_NOTIN    = "NOT IN"
	SQL_OP_EQUAL    = "="
	SQL_OP_LT       = "<"
	SQL_OP_LE       = "<="
	SQL_OP_GT       = ">"
	SQL_OP_GE       = ">="
	SQL_OP_BETWEEN  = "BETWEEN"
	SQL_OP_NOTEQUAL = "<>"
)

const (
	TAG_IGNORE           = "ignore"
	TAG_NAME             = "name"
	TAG_WIDTH            = "width"
	TAG_TEXT_LENGTH      = "length"
	TAG_CHARSET          = "charset"
	TAG_PRECISION        = "precision"
	TAG_DEFAULT          = "default"
	TAG_UNIQUE           = "unique"
	TAG_INDEX            = "index"
	TAG_PRIMARY          = "primary"
	TAG_NULLABLE         = "nullable"
	TAG_AUTOINCREMENT    = "auto_increment"
	TAG_AUTOVERSION      = "auto_version"
	TAG_UPDATE_TIMESTAMP = "updated_at"
	TAG_CREATE_TIMESTAMP = "created_at"
	TAG_ALLOW_ZERO       = "allow_zero"
)

var (
	INT_WIDTH_DEFAULT = map[string]int{
		"TINYINT":  4,
		"SMALLINT": 6,
		"INT":      11,
		"BIGINT":   20,
	}
	UNSIGNED_INT_WIDTH_DEFAULT = map[string]int{
		"TINYINT":  3,
		"SMALLINT": 5,
		"INT":      10,
		"BIGINT":   20,
	}
)

func intWidthString(typeStr string) string {
	return strconv.FormatInt(int64(INT_WIDTH_DEFAULT[strings.ToUpper(typeStr)]), 10)
}

func uintWidthString(typeStr string) string {
	return strconv.FormatInt(int64(UNSIGNED_INT_WIDTH_DEFAULT[strings.ToUpper(typeStr)]), 10)
}
