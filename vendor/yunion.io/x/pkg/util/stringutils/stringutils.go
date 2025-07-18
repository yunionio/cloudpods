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

package stringutils

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/golang-plus/uuid"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/timeutils"
)

func ParseNamePattern(name string) (string, string, int) {
	const RepChar = '#'
	var match string
	var pattern string
	var patternLen int

	start := strings.IndexByte(name, RepChar)
	if start >= 0 {
		end := start + 1
		for end < len(name) && name[end] == RepChar {
			end += 1
		}
		match = fmt.Sprintf("%s%%%s", name[:start], name[end:])
		pattern = fmt.Sprintf("%s%%0%dd%s", name[:start], end-start, name[end:])
		patternLen = end - start
	} else {
		match = fmt.Sprintf("%s-%%", name)
		pattern = fmt.Sprintf("%s-%%d", name)
	}
	return match, pattern, patternLen
}

func UUID4() string {
	uid, _ := uuid.NewV4()
	return uid.String()
}

func Interface2String(val interface{}) string {
	if val == nil {
		return ""
	}
	switch vval := val.(type) {
	case string:
		return vval
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", vval)
	case float32, float64:
		return fmt.Sprintf("%f", vval)
	case bool:
		return fmt.Sprintf("%v", vval)
	case error:
		return vval.Error()
	case time.Time:
		return timeutils.FullIsoTime(vval)
	case fmt.Stringer:
		return vval.String()
	default:
		json := jsonutils.Marshal(val)
		return json.String()
	}
}

func SplitKeyValue(line string) (string, string) {
	return SplitKeyValueBySep(line, ":")
}

func SplitKeyValueBySep(line string, sep string) (string, string) {
	pos := strings.Index(line, sep)
	if pos > 0 {
		key := strings.TrimSpace(line[:pos])
		val := strings.TrimSpace(line[pos+1:])
		return key, val
	}
	return "", ""
}

func ContainsWord(str, w string) bool {
	reg := regexp.MustCompile(fmt.Sprintf("\\b%s\\b", w))
	return reg.MatchString(str)
}

func byte2hex(b byte) byte {
	if b >= 0 && b <= 9 {
		return b + 0x30
	}
	if b >= 10 && b <= 15 {
		return b - 10 + 0x61
	}
	return '?'
}

func Bytes2Str(b []byte) string {
	buf := strings.Builder{}
	for i := range b {
		buf.WriteByte(byte2hex((b[i] & 0xf0) >> 4))
		buf.WriteByte(byte2hex(b[i] & 0x0f))
	}
	return buf.String()
}
