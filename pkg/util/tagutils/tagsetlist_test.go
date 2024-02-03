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
	"reflect"
	"testing"

	"yunion.io/x/jsonutils"
)

func TestTTagSetList_Contains(t *testing.T) {
	cases := []struct {
		tsl      TTagSetList
		ts       TTagSet
		contains bool
	}{
		{
			tsl:      TTagSetList{},
			ts:       TTagSet{},
			contains: true,
		},
		{
			tsl: TTagSetList{},
			ts: TTagSet{
				STag{
					Key:   "project",
					Value: "a",
				},
			},
			contains: true,
		},
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

func TestTTagSetList_Flattern(t *testing.T) {
	cases := []struct {
		tsl  TTagSetList
		want map[string]TTagSet
	}{
		{
			tsl: TTagSetList{
				TTagSet{
					STag{
						Key:   "project",
						Value: "a",
					},
					STag{
						Key:   "env",
						Value: "product",
					},
				},
				TTagSet{
					STag{
						Key:   "project",
						Value: "b",
					},
				},
			},
			want: map[string]TTagSet{
				"": {
					STag{
						Key:   "project",
						Value: "a",
					},
					STag{
						Key:   "env",
						Value: "product",
					},
				},
			},
		},
	}
	for _, c := range cases {
		got := c.tsl.Flattern()
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("want %s got %s", jsonutils.Marshal(c.want), jsonutils.Marshal(got))
		}
	}
}

func TestIntersect(t *testing.T) {
	cases := []struct {
		tsl  TTagSetList
		ts   TTagSet
		want TTagSetList
	}{
		{
			tsl: TTagSetList{
				TTagSet{
					{
						Key:   "a",
						Value: "1",
					},
				},
			},
			ts: TTagSet{
				{
					Key:   "b",
					Value: "2",
				},
			},
			want: TTagSetList{
				TTagSet{
					{
						Key:   "a",
						Value: "1",
					},
					{
						Key:   "b",
						Value: "2",
					},
				},
			},
		},
	}
	for _, c := range cases {
		got := c.tsl.Intersect(c.ts)
		if jsonutils.Marshal(got).String() != jsonutils.Marshal(c.want).String() {
			t.Errorf("got %s want %s", jsonutils.Marshal(got).String(), jsonutils.Marshal(c.want).String())
		}
	}
}

func TestIntersects(t *testing.T) {
	cases := []struct {
		tsl  TTagSetList
		ts2  TTagSetList
		want TTagSetList
	}{
		{
			tsl: TTagSetList{
				TTagSet{
					{
						Key:   "a",
						Value: "1",
					},
				},
				TTagSet{
					{
						Key:   "b",
						Value: "1",
					},
				},
			},
			ts2: TTagSetList{
				TTagSet{
					{
						Key:   "c",
						Value: "2",
					},
				},
			},
			want: TTagSetList{
				TTagSet{
					{
						Key:   "a",
						Value: "1",
					},
					{
						Key:   "c",
						Value: "2",
					},
				},
				TTagSet{
					{
						Key:   "b",
						Value: "1",
					},
					{
						Key:   "c",
						Value: "2",
					},
				},
			},
		},
		{
			tsl: TTagSetList{
				TTagSet{
					{
						Key:   "a",
						Value: "1",
					},
				},
				TTagSet{
					{
						Key:   "b",
						Value: "1",
					},
				},
			},
			ts2: TTagSetList{},
			want: TTagSetList{
				TTagSet{
					{
						Key:   "a",
						Value: "1",
					},
				},
				TTagSet{
					{
						Key:   "b",
						Value: "1",
					},
				},
			},
		},
	}
	for _, c := range cases {
		got := c.tsl.IntersectList(c.ts2)
		if jsonutils.Marshal(got).String() != jsonutils.Marshal(c.want).String() {
			t.Errorf("got %s want %s", jsonutils.Marshal(got).String(), jsonutils.Marshal(c.want).String())
		}
	}
}

func TestFlattern(t *testing.T) {
	cases := []struct {
		tsl  TTagSetList
		want map[string]TTagSet
	}{
		{
			tsl: TTagSetList{
				TTagSet{
					STag{
						Key:   "user:部门",
						Value: "技术",
					},
					STag{
						Key:   "user:环境",
						Value: "UAT",
					},
				},
				TTagSet{
					STag{
						Key:   "org:部门",
						Value: "技术",
					},
					STag{
						Key:   "org:环境",
						Value: "UAT",
					},
				},
			},
			want: map[string]TTagSet{
				"user": {
					STag{
						Key:   "user:部门",
						Value: "技术",
					},
					STag{
						Key:   "user:环境",
						Value: "UAT",
					},
				},
				"org": {
					STag{
						Key:   "org:部门",
						Value: "技术",
					},
					STag{
						Key:   "org:环境",
						Value: "UAT",
					},
				},
			},
		},
		{
			tsl: TTagSetList{
				TTagSet{
					STag{
						Key:   "user:国家",
						Value: "中国",
					},
					STag{
						Key:   "user:城市",
						Value: "天津",
					},
					STag{
						Key:   "user:部门",
						Value: "技术",
					},
					STag{
						Key:   "user:环境",
						Value: "Product",
					},
				},
				TTagSet{
					STag{
						Key:   "user:部门",
						Value: "技术",
					},
					STag{
						Key:   "user:环境",
						Value: "UAT",
					},
				},
				TTagSet{
					STag{
						Key:   "user:城市",
						Value: "北京",
					},
					STag{
						Key:   "user:部门",
						Value: "研发",
					},
					STag{
						Key:   "user:环境",
						Value: "UAT",
					},
				},
				TTagSet{
					STag{
						Key:   "org:部门",
						Value: "技术",
					},
					STag{
						Key:   "org:环境",
						Value: "UAT",
					},
				},
			},
			want: map[string]TTagSet{
				"user": {
					STag{
						Key:   "user:国家",
						Value: "中国",
					},
					STag{
						Key:   "user:城市",
						Value: "天津",
					},
					STag{
						Key:   "user:部门",
						Value: "技术",
					},
					STag{
						Key:   "user:环境",
						Value: "Product",
					},
				},
				"org": {
					STag{
						Key:   "org:部门",
						Value: "技术",
					},
					STag{
						Key:   "org:环境",
						Value: "UAT",
					},
				},
			},
		},
	}
	for _, c := range cases {
		got := c.tsl.Flattern()
		if jsonutils.Marshal(got).String() != jsonutils.Marshal(c.want).String() {
			t.Errorf("tsl %s flattern got %s want %s", jsonutils.Marshal(c.tsl), jsonutils.Marshal(got), jsonutils.Marshal(c.want))
		}
	}
}
