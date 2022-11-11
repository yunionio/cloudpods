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
	"math/rand"
	"strconv"
	"strings"
	"time"
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

func GetCharTypeCount(str string) int {
	digitIdx := 0
	lowerIdx := 1
	upperIdx := 2
	otherIdx := 3
	complexity := make([]int, 4)
	for _, b := range []byte(str) {
		if b >= '0' && b <= '9' {
			complexity[digitIdx] += 1
		} else if b >= 'a' && b <= 'z' {
			complexity[lowerIdx] += 1
		} else if b >= 'A' && b <= 'Z' {
			complexity[upperIdx] += 1
		} else {
			complexity[otherIdx] += 1
		}
	}
	ret := 0
	for i := range complexity {
		if complexity[i] > 0 {
			ret += 1
		}
	}
	return ret
}

// Qcloud: 1-128个英文字母、数字和+=,.@_-
// Aws: 请使用字母数字和‘+=,.@-_’字符。 最长 64 个字符
// Common: 1-64个字符, 数字字母或 +=,.@-_
func GenerateRoleName(roleName string) string {
	ret := ""
	for _, s := range roleName {
		if (s >= '0' && s <= '9') || (s >= 'a' && s <= 'z') || (s >= 'A' && s <= 'Z') || strings.Contains("+=,.@-_", string(s)) {
			ret += string(s)
		}
	}

	if len(ret) == 0 {
		return func(length int) string {
			bytes := []byte("23456789abcdefghijkmnpqrstuvwxyz")
			result := []byte{}
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			for i := 0; i < length; i++ {
				result = append(result, bytes[r.Intn(len(bytes))])
			}
			return "role-" + string(result)
		}(12)
	}

	if len(ret) > 64 {
		return ret[:64]
	}
	return ret
}

func FilterEmpty(input []string) []string {
	ret := make([]string, 0)
	for i := range input {
		if len(input[i]) > 0 {
			ret = append(ret, input[i])
		}
	}
	return ret
}

func PrettyFloat(f float64, precision int) string {
	fstr := strconv.FormatFloat(f, 'f', -1, 64)
	dotPos := strings.Index(fstr, ".")
	if dotPos < 0 {
		return fstr
	}
	neg := false
	if fstr[0] == '-' {
		neg = true
		fstr = fstr[1:]
		dotPos -= 1
	}
	numPart, _ := strconv.ParseInt(fstr[:dotPos], 10, 64)
	prePartStr := fstr[dotPos+1:]
	if numPart == 0 {
		nonZeroPos := 0
		for prePartStr[nonZeroPos] == '0' {
			nonZeroPos += 1
		}
		precision += nonZeroPos
	}
	if precision < len(prePartStr) {
		prePart, _ := strconv.ParseInt(prePartStr[:precision], 10, 64)
		if prePartStr[precision] > '4' {
			// rounding up
			prePart += 1
		}
		prePartStr = fmt.Sprintf(fmt.Sprintf("%%.%dd", precision), prePart)
		if len(prePartStr) > precision {
			prePartStr = prePartStr[1:]
			numPart += 1
		}
	}
	for len(prePartStr) > 0 && prePartStr[len(prePartStr)-1] == '0' {
		prePartStr = prePartStr[0 : len(prePartStr)-1]
	}
	ret := strconv.FormatInt(numPart, 10)
	if len(prePartStr) > 0 {
		ret += "." + prePartStr
	}
	if neg {
		ret = "-" + ret
	}
	return ret
}
