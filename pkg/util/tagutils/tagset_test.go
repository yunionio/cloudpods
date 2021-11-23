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

package tagutils

import "testing"

func TestTTagSet_Contains(t *testing.T) {
	cases := []struct {
		t1       TTagSet
		t2       TTagSet
		contains bool
	}{
		{
			t1:       TTagSet{},
			t2:       TTagSet{},
			contains: true,
		},
		{
			t1: TTagSet{
				STag{
					Key:   "project",
					Value: "a",
				},
			},
			t2: TTagSet{
				STag{
					Key:   "project",
					Value: "a",
				},
				STag{
					Key:   "env",
					Value: "c",
				},
			},
			contains: true,
		},
		{
			t1: TTagSet{
				STag{
					Key:   "project",
					Value: "a",
				},
			},
			t2: TTagSet{
				STag{
					Key:   "project",
					Value: "b",
				},
			},
			contains: false,
		},
	}
	for i, c := range cases {
		got := c.t1.Contains(c.t2)
		if got != c.contains {
			t.Errorf("[%d] want: %v got: %v", i, c.contains, got)
		}
	}
}
