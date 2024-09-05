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
	"testing"
)

func TestParseNamePattern2(t *testing.T) {
	cases := []struct {
		input      string
		match      string
		pattern    string
		patternLen int
		offset     int
		charType   byte
	}{
		{
			input:      "testimg###",
			match:      "testimg%",
			pattern:    "testimg%03d",
			patternLen: 3,
			offset:     0,
			charType:   RepChar,
		},
		{
			input:      "testimg###66#",
			match:      "testimg%",
			pattern:    "testimg%03d",
			patternLen: 3,
			offset:     66,
			charType:   RepChar,
		},
		{
			input:      "testimg###ab#",
			match:      "testimg%",
			pattern:    "testimg%03d",
			patternLen: 3,
			offset:     0,
			charType:   RepChar,
		},
		{
			input:      "testimg",
			match:      "testimg-%",
			pattern:    "testimg-%d",
			patternLen: 0,
			offset:     0,
		},
		{
			input:      "testimg???",
			match:      "testimg%",
			pattern:    "testimg%s",
			patternLen: 3,
			offset:     0,
			charType:   RandomChar,
		},
	}
	for _, c := range cases {
		m, p, pl, o, ch := ParseNamePattern2(c.input)
		if m != c.match || p != c.pattern || pl != c.patternLen || o != c.offset || ch != c.charType {
			t.Errorf("match got %s want %s, pattern got %s want %s, patternLen got %d want %d, offset got %d want %d, charType got %c want %c", m, c.match, p, c.pattern, pl, c.patternLen, o, c.offset, ch, c.charType)
		}
	}
}
