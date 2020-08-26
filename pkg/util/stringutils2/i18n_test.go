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

func TestIsUtf8(t *testing.T) {
	cases := []struct {
		In   string
		Want bool
	}{
		{"中文", true},
		{"this is eng", false},
	}
	for _, c := range cases {
		got := IsUtf8(c.In)
		if got != c.Want {
			t.Errorf("IsUtf8 %s got %v want %v", c.In, got, c.Want)
		}
	}
}

func TestIsPrintableAscii(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{
			in:   "passw0rd",
			want: true,
		},
		{
			in:   string([]byte{128, 45, 48}),
			want: false,
		},
		{
			in:   "中文",
			want: false,
		},
	}
	for _, c := range cases {
		got := IsPrintableAsciiString(c.in)
		if got != c.want {
			t.Errorf("%s IsPringtableAsciiString got %v want %v", c.in, got, c.want)
		}
	}
}
