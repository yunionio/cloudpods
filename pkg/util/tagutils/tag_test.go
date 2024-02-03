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

import (
	"testing"

	"yunion.io/x/jsonutils"
)

func TestSTagCompare(t *testing.T) {
	cases := []struct {
		t1  STag
		t2  STag
		cmp int
	}{
		{
			t1: STag{
				Key:   "a",
				Value: "1",
			},
			t2: STag{
				Key:   "b",
				Value: "2",
			},
			cmp: -1,
		},
		{
			t1: STag{
				Key:   "a",
				Value: NoValue,
			},
			t2: STag{
				Key:   "a",
				Value: "1",
			},
			cmp: 1,
		},
		{
			t1: STag{
				Key:   "a",
				Value: AnyValue,
			},
			t2: STag{
				Key:   "a",
				Value: "1",
			},
			cmp: -1,
		},
		{
			t1: STag{
				Key:   "a",
				Value: AnyValue,
			},
			t2: STag{
				Key:   "a",
				Value: NoValue,
			},
			cmp: -1,
		},
		{
			t1: STag{
				Key:   "a",
				Value: NoValue,
			},
			t2: STag{
				Key:   "a",
				Value: AnyValue,
			},
			cmp: 1,
		},
	}
	for _, c := range cases {
		got := Compare(c.t1, c.t2)
		if got != c.cmp {
			t.Errorf("Compare %s %s want %d got %d", jsonutils.Marshal(c.t1), jsonutils.Marshal(c.t2), c.cmp, got)
		}
	}
}

func TestKeyPrefix(t *testing.T) {
	cases := []struct {
		tag  STag
		want string
	}{
		{
			tag: STag{
				Key:   "user:abc",
				Value: "1",
			},
			want: "user",
		},
		{
			tag: STag{
				Key:   "org:abc",
				Value: "1",
			},
			want: "org",
		},
		{
			tag: STag{
				Key:   "cls:abc",
				Value: "1",
			},
			want: "cls",
		},
	}
	for _, c := range cases {
		got := c.tag.KeyPrefix()
		if got != c.want {
			t.Errorf("tag %s keyprefix %s got %s", c.tag.String(), c.want, got)
		}
	}
}
