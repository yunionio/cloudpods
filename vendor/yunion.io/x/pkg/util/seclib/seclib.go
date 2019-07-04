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

package seclib

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
)

const (
	DIGITS  = "23456789"
	LETTERS = "abcdefghjkmnpqrstuvwxyz"
	PUNC    = ""
)

var CHARS = fmt.Sprintf("%s%s%s%s", DIGITS, LETTERS, strings.ToUpper(LETTERS), PUNC)

func RandomPassword(width int) string {
	if width < 6 {
		width = 6
	}
	for {
		var buf bytes.Buffer
		digitsCnt := 0
		letterCnt := 0
		upperCnt := 0
		for i := 0; i < width; i += 1 {
			index := rand.Intn(len(CHARS))
			ch := CHARS[index]
			if strings.IndexByte(DIGITS, ch) >= 0 {
				digitsCnt += 1
			} else if strings.IndexByte(LETTERS, ch) >= 0 {
				letterCnt += 1
			} else if strings.IndexByte(LETTERS, ch+32) >= 0 {
				upperCnt += 1
			}
			buf.WriteByte(ch)
		}
		if digitsCnt > 1 && letterCnt > 1 && upperCnt > 1 {
			return buf.String()
		}
	}
	return ""
}
