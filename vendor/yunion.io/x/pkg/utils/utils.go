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
	"bytes"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"unicode"
)

func isUpperChar(ch byte) bool {
	return ch >= 'A' && ch <= 'Z'
}

func isLowerChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
}

func CamelSplit(str string, sep string) string {
	tokens := CamelSplitTokens(str)
	return strings.Join(tokens, sep)
}

func CamelSplitTokens(str string) []string {
	var (
		tokens     = make([]string, 0, 4)
		buf        = make([]byte, 0, 16)
		upperCount int
	)
	for i := range str {
		var (
			c     = str[i]
			inc   = true
			split bool
		)
		if isUpperChar(c) {
			upperCount += 1
			if upperCount == 1 {
				split = true
			} else if upperCount > 2 && i+1 < len(str) && isLowerChar(str[i+1]) {
				split = true
			}
			c += 'a' - 'A'
		} else if isLowerChar(c) {
			upperCount = 0
		} else {
			upperCount = 0
			split = true
			inc = false
		}
		if split && len(buf) > 0 {
			tokens = append(tokens, string(buf))
			buf = buf[:0]
		}
		if inc {
			buf = append(buf, c)
		}
	}
	if len(buf) > 0 {
		tokens = append(tokens, string(buf))
	}
	return tokens
}

func Capitalize(str string) string {
	if len(str) >= 1 {
		return strings.ToUpper(str[:1]) + strings.ToLower(str[1:])
	} else {
		return str
	}
}

func Kebab2Camel(kebab string, sep string) string {
	var buf bytes.Buffer
	segs := strings.Split(kebab, sep)
	for _, s := range segs {
		buf.WriteString(Capitalize(s))
	}
	return buf.String()
}

var TRUE_STRS = []string{"1", "true", "on", "yes"}

func ToBool(str string) bool {
	val := strings.ToLower(strings.TrimSpace(str))
	for _, v := range TRUE_STRS {
		if v == val {
			return true
		}
	}
	return false
}

func DecodeMeta(str string) string {
	s, e := url.QueryUnescape(str)
	if e == nil && s != str {
		return DecodeMeta(s)
	} else {
		return str
	}
}

func IsInStringArray(val string, array []string) bool {
	return IsInArray(val, array)
}

func InStringArray(val string, array []string) (ok bool, i int) {
	return InArray2(val, array)
}

func InArray2[T int | uint | int8 | uint8 | int16 | uint16 | int32 | uint32 | int64 | uint64 | float32 | float64 | string](val T, array []T) (ok bool, i int) {
	for i = range array {
		if ok = array[i] == val; ok {
			return
		}
	}
	return
}

func IsInArray[T int | uint | int8 | uint8 | int16 | uint16 | int32 | uint32 | int64 | uint64 | float32 | float64 | string](val T, array []T) bool {
	for _, ele := range array {
		if ele == val {
			return true
		}
	}
	return false
}

func InArray(v interface{}, in interface{}) (ok bool, i int) {
	val := reflect.Indirect(reflect.ValueOf(in))
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		for ; i < val.Len(); i++ {
			if ok = v == val.Index(i).Interface(); ok {
				return
			}
		}
	}
	return
}

func TruncateString(v interface{}, maxLen int) string {
	str := fmt.Sprintf("%s", v)
	if len(str) > maxLen {
		str = str[:maxLen] + ".."
	}
	return str
}

func IsAscii(str string) bool {
	for _, c := range str {
		if c > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func FloatRound(num float64, precision int) float64 {
	for i := 0; i < precision; i++ {
		num *= 10
	}
	temp := float64(int64(num))
	for i := 0; i < precision; i++ {
		temp /= 10.0
	}
	num = temp
	return num
}

func getStringInQuote(s string, start, length int, quote byte, backIndex int) (string, int) {
	var i = start + 1
	if quote != '\'' && quote != '"' {
		for i < length {
			if s[i] == ' ' || s[i] == '\'' || s[i] == '"' {
				return s[backIndex:i], i
			}
			i++
		}
		return s[backIndex:], i
	} else {
		for i < length {
			if s[i] == quote {
				return s[backIndex+1 : i], i + 1
			}
			i++
		}
		return s[backIndex:], i
	}
}

func ArgsStringToArray(s string) []string {
	var args, i, j = make([]string, 0), 0, 0
	var length = len(s)
	for i < length {
		switch s[i] {
		case ' ':
			if i > j {
				args = append(args, s[j:i])
				j = i
			}
			i++
			j = i
		case '"':
			var s1, s2 string
			var oldStr = s[j:i]
			s1, i = getStringInQuote(s, i, length, '"', i)
			s1 = oldStr + s1
			for i < length && s[i] != ' ' {
				s2, i = getStringInQuote(s, i, length, s[i], i)
				s1 += s2
				fmt.Println(s2)
			}
			args = append(args, s1)
			i++
			j = i
		case '\'':
			var s1, s2 string
			var oldStr = s[j:i]
			s1, i = getStringInQuote(s, i, length, '\'', i)
			s1 = oldStr + s1
			for i < length && s[i] != ' ' {
				s2, i = getStringInQuote(s, i, length, s[i], i)
				s1 += s2
			}
			args = append(args, s1)
			i++
			j = i
		default:
			i++
		}
	}
	if j < length {
		args = append(args, s[j:])
	}
	return args
}
