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

package appsrv

import (
	"testing"
)

func TestSplitPath(t *testing.T) {
	cases := []struct {
		in  string
		out int
	}{
		{in: "/v2.0/tokens/123", out: 3},
		{in: "/v2.0//tokens//123", out: 3},
		{in: "/", out: 0},
		{in: "/v2.0//123//", out: 2},
	}
	for _, p := range cases {
		ret := SplitPath(p.in)
		if len(ret) != p.out {
			t.Error("Split error for ", p.in, " out ", ret)
		}
	}
}
