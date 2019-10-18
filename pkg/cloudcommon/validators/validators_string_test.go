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

package validators

import (
	"testing"
)

func TestStringLenValidator(t *testing.T) {
	cases := []*C{
		{
			Name:      "missing non-optional",
			In:        `{}`,
			Out:       `{}`,
			Optional:  false,
			Err:       ERR_MISSING_KEY,
			ValueWant: "",
		},
		{
			Name:      "missing optional",
			In:        `{}`,
			Out:       `{}`,
			Optional:  true,
			ValueWant: "",
		},
		{
			Name:      "missing with default",
			In:        `{}`,
			Out:       `{s: "12345"}`,
			Default:   "12345",
			ValueWant: "12345",
		},
		{
			Name:      "stringified",
			In:        `{"s": 100}`,
			Out:       `{s: "100"}`,
			ValueWant: "100",
		},
		{
			Name:      "stringified too long",
			In:        `{"s": 9876543210}`,
			Out:       `{"s": 9876543210}`,
			Err:       ERR_INVALID_LENGTH,
			ValueWant: "",
		},
		{
			Name:      "stringified too short",
			In:        `{"s": 0}`,
			Out:       `{"s": 0}`,
			Err:       ERR_INVALID_LENGTH,
			ValueWant: "",
		},
		{
			Name:      "good length",
			In:        `{"s": "abcde"}`,
			Out:       `{"s": "abcde"}`,
			ValueWant: "abcde",
		},
		{
			Name:      "bad length (too short)",
			In:        `{"s": "0"}`,
			Out:       `{"s": "0"}`,
			Err:       ERR_INVALID_LENGTH,
			ValueWant: "",
		},
		{
			Name:      "bad length (too long)",
			In:        `{"s": "9876543210"}`,
			Out:       `{"s": "9876543210"}`,
			Err:       ERR_INVALID_LENGTH,
			ValueWant: "",
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			v := NewStringLenRangeValidator("s", 2, 5)
			if c.Default != nil {
				s := c.Default.(string)
				v.Default(s)
			}
			if c.Optional {
				v.Optional(true)
			}
			testS(t, v, c)
		})
	}
}
