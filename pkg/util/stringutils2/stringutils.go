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
	"unicode"

	"yunion.io/x/pkg/util/osprofile"
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

func GenerateHostName(name string, osType string) string {
	if len(name) < 2 {
		name = fmt.Sprintf("hostname-for-%s-%s", name, osType)
	}
	// ()英文句号（.）和短横线（-）不能作为首尾字符，更不能连续使用。
	// 点号（.）和短横线（-）不能作为 HostName 的首尾字符，不能连续使用。
	var init = func(s string) string {
		for {
			if strings.Contains(s, "..") || strings.Contains(s, "--") {
				s = strings.ReplaceAll(s, "..", ".")
				s = strings.ReplaceAll(s, "--", "-")
				continue
			}
			if strings.HasPrefix(s, ".") || strings.HasPrefix(s, "-") {
				s = strings.TrimPrefix(s, ".")
				s = strings.TrimPrefix(s, "-")
				continue
			}
			if strings.HasSuffix(s, ".") || strings.HasSuffix(s, "-") {
				s = strings.TrimSuffix(s, ".")
				s = strings.TrimSuffix(s, "-")
				continue
			}
			break
		}
		return s
	}
	name = init(name)
	// (阿里云)Windows实例：字符长度为2~15，不支持英文句号（.），不能全是数字。允许大小写英文字母、数字和短横线（-）。
	// (腾讯云)Windows 实例：名字符长度为[2, 15]，允许字母（不限制大小写）、数字和短横线（-）组成，不支持点号（.），不能全是数字
	var forWindows = func(s string) string {
		s = strings.ReplaceAll(s, ".", "")
		ret := ""
		for _, c := range s {
			if unicode.IsLetter(c) || unicode.IsNumber(c) || c == '-' {
				ret += string(c)
			}
		}
		_, err := strconv.Atoi(ret)
		if err == nil {
			ret = "host-" + ret
		}
		if len(ret) > 15 {
			ret = init(ret[:15])
		}
		return ret
	}
	// (阿里云)其他类型实例（Linux等）：字符长度为2~64，支持多个英文句号（.），英文句号之间为一段，每段允许大小写英文字母、数字和短横线（-）。
	// (腾讯云)其他类型（Linux 等）实例：字符长度为[2, 60]，允许支持多个点号，点之间为一段，每段允许字母（不限制大小写）、数字和短横线（-）组成。
	var forOther = func(s string) string {
		if len(s) > 60 {
			return init(s[:60])
		}
		return s
	}
	if strings.ToLower(osType) == strings.ToLower(osprofile.OS_TYPE_WINDOWS) {
		return forWindows(name)
	}
	return forOther(name)
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
