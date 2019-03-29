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
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
)

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func EscapeString(str string, pairs [][]string) string {
	if len(pairs) == 0 {
		pairs = [][]string{
			{"\\", `\\`},
			{"\n", `\n`},
			{"\r", `\r`},
			{"\t", `\t`},
			{`"`, `\"`},
			{"'", `\'`},
			{"$", `\$`},
		}
	}
	for _, pair := range pairs {
		k, v := pair[0], pair[1]
		str = strings.Replace(str, k, v, -1)
	}
	return str
}

func EscapeEchoString(str string) (string, error) {
	pairs := [][]string{
		{"\\", `\\`},
		{"\n", `\n`},
		{"\r", `\r`},
		{"\t", `\t`},
		{`"`, `\"`},
	}
	innerPairs := [][]string{
		{"\\", `\\`},
		{"\n", `\n`},
		{"\r", `\r`},
		{"\t", `\t`},
		{`"`, `\"`},
		{"$", `\$`},
	}
	segs, err := SplitByQuotation(str)
	if err != nil {
		return "", err
	}
	ret := ""
	for idx := 0; idx < len(segs); idx++ {
		s := EscapeString(segs[idx], innerPairs)
		if idx%2 == 1 {
			s = EscapeString(segs[idx], pairs)
			s = `\"` + EscapeString(s, innerPairs) + `\"`
		}
		ret += s
	}
	return ret, nil
}

func findQuotationPos(line string, offset int) int {
	if offset > len(line) {
		return -1
	}
	subStr := line[offset:]
	pos := strings.Index(subStr, `"`)
	if pos < 0 {
		return pos
	}
	pos += offset
	if pos > 0 && line[pos-1] == '\\' {
		return findQuotationPos(line, pos+1)
	}
	return pos
}

func SplitByQuotation(line string) ([]string, error) {
	segs := []string{}
	offset := 0
	for offset < len(line) {
		pos := findQuotationPos(line, offset)
		if pos < 0 {
			segs = append(segs, line[offset:])
			offset = len(line)
		} else {
			if pos == offset {
				offset += 1
			} else {
				segs = append(segs, line[offset:pos])
				offset = pos + 1
			}
			pos = findQuotationPos(line, offset)
			if pos < 0 {
				return nil, fmt.Errorf("Unpaired quotations: %s", line[offset:])
			} else {
				segs = append(segs, line[offset:pos])
				offset = pos + 1
			}
		}
	}
	return segs, nil
}
