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

func TestTTagSetList_Contains(t *testing.T) {
	cases := []struct {
		tsl      TTagSetList
		ts       TTagSet
		contains bool
	}{
		{
			tsl: TTagSetList{
				nil,
			},
			ts: TTagSet{
				STag{
					Key:   "project",
					Value: "a",
				},
			},
			contains: true,
		},
	}
	for i, c := range cases {
		got := c.tsl.Contains(c.ts)
		if got != c.contains {
			t.Errorf("[%d] want %v got %v", i, c.contains, got)
		}
	}
}

func TestTTagSetList_ContainsAll(t *testing.T) {
	cases := []struct {
		t1       TTagSetList
		t2       TTagSetList
		contains bool
	}{
		{
			t1: TTagSetList{
				TTagSet{},
			},
			t2: TTagSetList{
				TTagSet{},
			},
			contains: true,
		},
		{
			t1: TTagSetList{
				nil,
			},
			t2: TTagSetList{
				nil,
			},
			contains: true,
		},
		{
			t1: TTagSetList{
				nil,
			},
			t2: TTagSetList{
				TTagSet{},
			},
			contains: true,
		},
		{
			t1: TTagSetList{
				nil,
			},
			t2: TTagSetList{
				TTagSet{
					STag{
						Key:   "project",
						Value: "a",
					},
				},
			},
			contains: true,
		},
		{
			t1: TTagSetList{
				TTagSet{},
			},
			t2: TTagSetList{
				TTagSet{
					STag{
						Key:   "project",
						Value: "a",
					},
				},
				TTagSet{
					STag{
						Key:   "project",
						Value: "b",
					},
				},
			},
			contains: true,
		},
		{
			t1: TTagSetList{
				TTagSet{
					STag{
						Key:   "project",
						Value: "a",
					},
				},
			},
			t2: TTagSetList{
				TTagSet{
					STag{
						Key:   "project",
						Value: "b",
					},
				},
			},
			contains: false,
		},
	}
	for i, c := range cases {
		got := c.t1.ContainsAll(c.t2)
		if got != c.contains {
			t.Errorf("[%d] want: %v got: %v", i, c.contains, got)
		}
	}
}

func TestTTagSetList_Append(t *testing.T) {
	cases := []struct {
		list TTagSetList
		tag  TTagSet
		want TTagSetList
	}{
		{
			list: TTagSetList{
				TTagSet{},
			},
			tag: TTagSet{
				STag{
					Key:   "project",
					Value: "a",
				},
			},
			want: TTagSetList{
				TTagSet{},
			},
		},
		{
			list: TTagSetList{
				TTagSet{
					STag{
						Key:   "project",
						Value: "a",
					},
				},
				TTagSet{
					STag{
						Key:   "project",
						Value: "b",
					},
				},
			},
			tag: TTagSet{
				STag{
					Key:   "project",
					Value: "c",
				},
			},
			want: TTagSetList{
				TTagSet{
					STag{
						Key:   "project",
						Value: "a",
					},
				},
				TTagSet{
					STag{
						Key:   "project",
						Value: "b",
					},
				},
				TTagSet{
					STag{
						Key:   "project",
						Value: "c",
					},
				},
			},
		},
		{
			list: TTagSetList{
				TTagSet{
					STag{
						Key:   "project",
						Value: "a",
					},
				},
				TTagSet{
					STag{
						Key:   "project",
						Value: "b",
					},
				},
			},
			tag: TTagSet{},
			want: TTagSetList{
				TTagSet{},
			},
		},
	}
	for i, c := range cases {
		got := c.list.Append(c.tag)
		if got.String() != c.want.String() {
			t.Errorf("[%d] want %s got %s", i, c.want.String(), got.String())
		}
	}
}
