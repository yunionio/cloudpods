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
	"reflect"
	"strings"
)

const (
	EMPTYSTR = " \t\n\r"
)

func skipEmpty(str []byte, offset int) int {
	i := offset
	for i < len(str) && strings.IndexByte(EMPTYSTR, str[i]) >= 0 {
		i++
	}
	return i
}

func Unquote(str string) string {
	ret, _ := findString([]byte(str), 0)
	return ret
}

func findString(str []byte, offset int) (string, int) {
	return _findWord(str, offset, "\n\r", isQuoteCharInternal)
}

func findWord(str []byte, offset int) (string, int) {
	return _findWord(str, offset, " :,\t\n}]", isQuoteCharInternal)
}

func isQuoteCharInternal(ch byte) (bool, string) {
	switch ch {
	case '"':
		return true, "\""
	case '\'':
		return true, "'"
	default:
		return false, ""
	}
}

func _findWord(str []byte, offset int, sepChars string, isQuoteChar func(ch byte) (bool, string)) (string, int) {
	var buffer bytes.Buffer
	i := skipEmpty(str, offset)
	if i >= len(str) {
		return "", i
	}
	quote, endstr := isQuoteChar(str[i])
	if quote {
		i++
	} else {
		// endstr = " :,\t\n\r}]"
		endstr = sepChars
	}
	for i < len(str) {
		if quote && str[i] == '\\' {
			if i+1 < len(str) {
				i++
				switch str[i] {
				case 'n':
					buffer.WriteByte('\n')
				case 'r':
					buffer.WriteByte('\r')
				case 't':
					buffer.WriteByte('\t')
				default:
					buffer.WriteByte(str[i])
				}
				i++
			} else {
				break
			}
		} else if strings.IndexByte(endstr, str[i]) >= 0 { // end
			if quote {
				i++
			}
			break
		} else {
			buffer.WriteByte(str[i])
			i++
		}
	}
	return buffer.String(), i
}

func FindWords(str []byte, offset int) []string {
	words, err := FindWords2(str, offset, " :,\t\n}]", isQuoteCharInternal)
	if err != nil {
		panic(err.Error())
	}
	return words
}

func SplitWords(str string) ([]string, error) {
	offset := 0
	words := make([]string, 0)
	for offset < len(str) {
		word, i := _findWord([]byte(str), offset, " \n\r\t", isQuoteCharInternal)
		words = append(words, word)
		offset = i
	}
	return words, nil
}

func FindWords2(str []byte, offset int, sepChars string, isQuoteChar func(ch byte) (bool, string)) ([]string, error) {
	words := make([]string, 0)
	for offset < len(str) {
		word, i := _findWord(str, offset, sepChars, isQuoteChar)
		words = append(words, word)
		i = skipEmpty(str, i)
		if i < len(str) {
			if strings.IndexByte(sepChars, str[i]) >= 0 {
				offset = i + 1
			} else {
				return nil, fmt.Errorf("Malformed multi value string: %s", string(str[offset:]))
			}
		} else {
			offset = i
		}
	}
	return words, nil
}

func TagMap(tag reflect.StructTag) map[string]string {
	ret := make(map[string]string)
	str := []byte(tag)
	i := 0
	for i < len(str) {
		var k, val string
		k, i = findWord(str, i)
		if len(k) == 0 {
			break
		}
		i = skipEmpty(str, i)
		if i >= len(str) || strings.IndexByte(EMPTYSTR, str[i]) >= 0 {
			val = ""
		} else if str[i] != ':' {
			panic(fmt.Sprintf("Invalid structTag: %s", tag))
		} else {
			i++
			val, i = findWord(str, i)
		}
		ret[k] = val
		i = skipEmpty(str, i)
	}
	return ret
}

func TagPop(m map[string]string, key string) (map[string]string, string, bool) {
	val, ok := m[key]
	if ok {
		delete(m, key)
	}
	return m, val, ok
}

func SplitCSV(csv string) []string {
	offset := 0
	words := make([]string, 0)
	str := []byte(csv)
	for offset < len(str) {
		var word string
		word, offset = _findWord(str, offset, ",\r\n", isQuoteCharInternal)
		words = append(words, word)
		if offset < len(str) {
			offset++
			if offset >= len(str) {
				words = append(words, "")
			}
		}
	}
	return words
}
