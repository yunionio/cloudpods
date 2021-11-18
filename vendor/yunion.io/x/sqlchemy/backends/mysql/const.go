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
	"strconv"
	"strings"
)

var (
	// INT_WIDTH_DEFAULT records the default width of integer type
	INT_WIDTH_DEFAULT = map[string]int{
		"TINYINT":  4,
		"SMALLINT": 6,
		"INT":      11,
		"BIGINT":   20,
	}
	// UNSIGNED_INT_WIDTH_DEFAULT records the default width of unsigned integer type
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
