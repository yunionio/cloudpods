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

package stringutils2

import (
	"fmt"
	"strconv"
	"strings"
)

// name##
// name##9#
func ParseNamePattern2(name string) (string, string, int, int) {
	const RepChar = '#'
	var match string
	var pattern string
	var patternLen int
	var offset int

	start := strings.IndexByte(name, RepChar)
	if start >= 0 {
		end := start + 1
		for end < len(name) && name[end] == RepChar {
			end += 1
		}
		patternLen = end - start
		nend := strings.IndexByte(name[end:], RepChar)
		if nend > 0 {
			if oi, err := strconv.ParseInt(name[end:end+nend], 10, 64); err == nil {
				offset = int(oi)
			}
			end = end + nend + 1
		}
		match = fmt.Sprintf("%s%%%s", name[:start], name[end:])
		pattern = fmt.Sprintf("%s%%0%dd%s", name[:start], patternLen, name[end:])
	} else {
		match = fmt.Sprintf("%s-%%", name)
		pattern = fmt.Sprintf("%s-%%d", name)
	}
	return match, pattern, patternLen, offset
}
