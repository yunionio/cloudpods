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

package cloudprovider

import "testing"

func TestParseRange(t *testing.T) {
	cases := []struct {
		in    string
		start int64
		end   int64
	}{
		{
			in:    "bytes=0-200",
			start: 0,
			end:   200,
		},
		{
			in:    "200-3232300",
			start: 200,
			end:   3232300,
		},
		{
			in:    "200-",
			start: 200,
			end:   0,
		},
		{
			in:    "-232323",
			start: 0,
			end:   232323,
		},
	}
	for _, c := range cases {
		got := ParseRange(c.in)
		if got.Start != c.start || got.End != c.end {
			t.Fatalf("got.start(%d) != want.start(%d) or got.end(%d) != want.end(%d)", got.Start, c.start, got.End, c.end)
		}
	}
}
