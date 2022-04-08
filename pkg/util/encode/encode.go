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

package encode

import (
	"strings"
)

const lowerhex = "0123456789abcdef"

func ishex(c rune) bool {
	switch {
	case rune('0') <= c && c <= rune('9'):
		return true
	case rune('a') <= c && c <= rune('f'):
		return true
	}
	return false
}

func unhex(c rune) byte {
	switch {
	case rune('0') <= c && c <= rune('9'):
		return byte(c) - '0'
	case rune('a') <= c && c <= rune('f'):
		return byte(c) - 'a' + 10
	}
	return 0
}

func shouldEncode(c rune) bool {
	if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c > 256 {
		return false
	}
	return true
}

func EncodeGoogleLabel(label string) string {
	var t strings.Builder
	for _, c := range label {
		if shouldEncode(c) {
			t.WriteByte('_')
			t.WriteByte(lowerhex[c>>4])
			t.WriteByte(lowerhex[c&15])
			continue
		}
		t.WriteRune(c)
	}
	return t.String()
}

func DecodeGoogleLable(label string) string {
	s := []rune{}
	for _, c := range label {
		s = append(s, c)
	}
	var t strings.Builder
	for j := 0; j < len(s); {
		c := s[j]
		if c == rune('_') && j+2 <= len(s) && ishex(s[j+1]) && ishex(s[j+2]) {
			t.WriteByte(unhex(s[j+1])<<4 | unhex(s[j+2]))
			j += 3
		} else {
			t.WriteRune(s[j])
			j += 1
		}
	}
	return t.String()
}
