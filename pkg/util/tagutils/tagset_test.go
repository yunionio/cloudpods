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

func TestTTagset_Add(t *testing.T) {
	cases := []struct {
		in   TTagSet
		want TTagSet
	}{
		{
			in: TTagSet{
				STag{
					Key:   "b",
					Value: "1",
				},
				STag{
					Key:   "a",
					Value: "2",
				},
			},
			want: TTagSet{
				STag{
					Key:   "a",
					Value: "2",
				},
				STag{
					Key:   "b",
					Value: "1",
				},
			},
		},
		{
			in: TTagSet{
				STag{
					Key:   "a",
					Value: NoValue,
				},
				STag{
					Key:   "a",
					Value: AnyValue,
				},
			},
			want: TTagSet{},
		},
		{
			in: TTagSet{
				STag{
					Key:   "b",
					Value: "1",
				},
				STag{
					Key:   "a",
					Value: "2",
				},
				STag{
					Key:   "a",
					Value: AnyValue,
				},
			},
			want: TTagSet{
				STag{
					Key:   "a",
					Value: AnyValue,
				},
				STag{
					Key:   "b",
					Value: "1",
				},
			},
		},
		{
			in: TTagSet{
				STag{
					Key:   "b",
					Value: "1",
				},
				STag{
					Key:   "a",
					Value: "2",
				},
				STag{
					Key:   "a",
					Value: NoValue,
				},
			},
			want: TTagSet{
				STag{
					Key:   "a",
					Value: "2",
				},
				STag{
					Key:   "a",
					Value: NoValue,
				},
				STag{
					Key:   "b",
					Value: "1",
				},
			},
		},
		{
			in: TTagSet{
				STag{
					Key:   "b",
					Value: "1",
				},
				STag{
					Key:   "a",
					Value: NoValue,
				},
				STag{
					Key:   "a",
					Value: "2",
				},
			},
			want: TTagSet{
				STag{
					Key:   "a",
					Value: "2",
				},
				STag{
					Key:   "a",
					Value: NoValue,
				},
				STag{
					Key:   "b",
					Value: "1",
				},
			},
		},
		{
			in: TTagSet{
				STag{
					Key:   "b",
					Value: "1",
				},
				STag{
					Key:   "a",
					Value: "2",
				},
				STag{
					Key:   "a",
					Value: NoValue,
				},
				STag{
					Key:   "a",
					Value: AnyValue,
				},
			},
			want: TTagSet{
				STag{
					Key:   "b",
					Value: "1",
				},
			},
		},
	}
	for _, c := range cases {
		got := TTagSet{}
		got = got.Append(c.in...)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("want %s got %s", jsonutils.Marshal(c.want), jsonutils.Marshal(got))
		}
	}
}

func TestTTagSet_Contains(t *testing.T) {
	cases := []struct {
		t1    TTagSet
		t2    TTagSet
		t1ct2 bool
		t2ct1 bool
	}{
		{
			t1:    TTagSet{},
			t2:    TTagSet{},
			t1ct2: true,
			t2ct1: true,
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
			t1ct2: true,
			t2ct1: false,
		},
		{
			t1: TTagSet{},
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
			t1ct2: true,
			t2ct1: false,
		},
		{
			t1: TTagSet{
				STag{
					Key:   "project",
					Value: "a",
				},
				STag{
					Key:   "project",
					Value: "b",
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
			t1ct2: false,
			t2ct1: false,
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
			t1ct2: false,
			t2ct1: false,
		},
		{
			t1: TTagSet{
				STag{
					Key:   "project",
					Value: AnyValue,
				},
				STag{
					Key:   "zone",
					Value: "a",
				},
			},
			t2: TTagSet{
				STag{
					Key:   "project",
					Value: "b",
				},
				STag{
					Key:   "zone",
					Value: "a",
				},
				STag{
					Key:   "env",
					Value: "c",
				},
			},
			t1ct2: true,
			t2ct1: false,
		},
		{
			t1: TTagSet{
				STag{
					Key:   "project",
					Value: AnyValue,
				},
			},
			t2: TTagSet{
				STag{
					Key:   "project",
					Value: "b",
				},
			},
			t1ct2: true,
			t2ct1: false,
		},
	}
	for i, c := range cases {
		got12 := c.t1.Contains(c.t2)
		if got12 != c.t1ct2 {
			t.Errorf("[%d] t1 contains t2 want: %v got: %v", i, c.t1ct2, got12)
		}
		got21 := c.t2.Contains(c.t1)
		if got21 != c.t2ct1 {
			t.Errorf("[%d] t2 contains t1 want: %v got: %v", i, c.t2ct1, got21)
		}
	}
}

func TestTagset2MapString(t *testing.T) {
	cases := []struct {
		ts   TTagSet
		want map[string]string
	}{
		{
			ts:   TTagSet{},
			want: map[string]string{},
		},
		{
			ts: TTagSet{
				STag{
					Key:   "project",
					Value: "a",
				},
			},
			want: map[string]string{
				"project": "a",
			},
		},
		{
			ts: TTagSet{
				STag{
					Key:   "project",
					Value: "b",
				},
				STag{
					Key:   "project",
					Value: "a",
				},
			},
			want: map[string]string{
				"project": "a",
			},
		},
		{
			ts: TTagSet{
				STag{
					Key:   "project",
					Value: "b",
				},
				STag{
					Key:   "project",
					Value: AnyValue,
				},
			},
			want: map[string]string{
				"project": "",
			},
		},
		{
			ts: TTagSet{
				STag{
					Key:   "project",
					Value: NoValue,
				},
				STag{
					Key:   "project",
					Value: AnyValue,
				},
			},
			want: map[string]string{},
		},
		{
			ts: TTagSet{
				STag{
					Key:   "project",
					Value: NoValue,
				},
				STag{
					Key:   "project",
					Value: "a",
				},
			},
			want: map[string]string{
				"project": "a",
			},
		},
		{
			ts: TTagSet{
				STag{
					Key:   "project",
					Value: NoValue,
				},
				STag{
					Key:   "project",
					Value: "a",
				},
				STag{
					Key:   "env",
					Value: "c",
				},
			},
			want: map[string]string{
				"project": "a",
				"env":     "c",
			},
		},
	}
	for i, c := range cases {
		got := Tagset2MapString(c.ts)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("[%d] want %s got %s", i, jsonutils.Marshal(c.want), jsonutils.Marshal(got))
		}
	}
}

func TestTagSetAppend(t *testing.T) {
	cases := []struct {
		set1 TTagSet
		set2 TTagSet
		want TTagSet
	}{
		{
			set1: TTagSet{
				{
					Key:   "a",
					Value: "1",
				},
			},
			set2: TTagSet{
				{
					Key:   "b",
					Value: "2",
				},
			},
			want: TTagSet{
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
		{
			set1: TTagSet{
				{
					Key:   "a",
					Value: AnyValue,
				},
			},
			set2: TTagSet{
				{
					Key:   "a",
					Value: "1",
				},
				{
					Key:   "b",
					Value: "2",
				},
			},
			want: TTagSet{
				{
					Key:   "a",
					Value: AnyValue,
				},
				{
					Key:   "b",
					Value: "2",
				},
			},
		},
		{
			set1: TTagSet{
				{
					Key:   "a",
					Value: AnyValue,
				},
			},
			set2: TTagSet{
				{
					Key:   "a",
					Value: NoValue,
				},
				{
					Key:   "b",
					Value: "2",
				},
			},
			want: TTagSet{
				{
					Key:   "b",
					Value: "2",
				},
			},
		},
		{
			set1: TTagSet{
				{
					Key:   "a",
					Value: AnyValue,
				},
			},
			set2: TTagSet{
				{
					Key:   "a",
					Value: NoValue,
				},
			},
			want: TTagSet{},
		},
		{
			set1: TTagSet{
				{
					Key:   "a",
					Value: AnyValue,
				},
			},
			set2: TTagSet{},
			want: TTagSet{
				{
					Key:   "a",
					Value: AnyValue,
				},
			},
		},
		{
			set1: TTagSet{
				{
					Key:   "a",
					Value: AnyValue,
				},
			},
			set2: TTagSet{
				{
					Key:   "a",
					Value: AnyValue,
				},
			},
			want: TTagSet{
				{
					Key:   "a",
					Value: AnyValue,
				},
			},
		},
	}
	for _, c := range cases {
		got := c.set1.Append(c.set2...)
		if jsonutils.Marshal(got).String() != jsonutils.Marshal(c.want).String() {
			t.Errorf("got: %s want: %s", jsonutils.Marshal(got).String(), jsonutils.Marshal(c.want).String())
		}
	}
}

func TestTagSetKeyPrefix(t *testing.T) {
	cases := []struct {
		tags TTagSet
		want string
	}{
		{
			tags: TTagSet{
				STag{
					Key:   "user:abc",
					Value: "1",
				},
				STag{
					Key:   "user:efd",
					Value: "1",
				},
			},
			want: "user",
		},
		{
			tags: TTagSet{
				STag{
					Key:   "org:abc",
					Value: "1",
				},
			},
			want: "org",
		},
		{
			tags: TTagSet{
				STag{
					Key:   "cls:abc",
					Value: "1",
				},
			},
			want: "cls",
		},
		{
			tags: TTagSet{
				STag{
					Key:   "cls:abc",
					Value: "1",
				},
				STag{
					Key:   "user:abc",
					Value: "1",
				},
			},
			want: "",
		},
	}
	for _, c := range cases {
		got := c.tags.KeyPrefix()
		if got != c.want {
			t.Errorf("tag %s keyprefix %s got %s", c.tags.String(), c.want, got)
		}
	}
}

func TestTagSet2Paths(t *testing.T) {
	cases := []struct {
		tagset TTagSet
		keys   []string
		paths  [][]string
	}{
		{
			tagset: TTagSet{
				{
					Key:   "部门",
					Value: "技术",
				},
				{
					Key:   "环境",
					Value: "测试",
				},
			},
			keys: []string{
				"部门", "环境",
			},
			paths: [][]string{
				{
					"技术", "测试",
				},
			},
		},
		{
			tagset: TTagSet{
				{
					Key:   "部门",
					Value: "技术",
				},
				{
					Key:   "环境",
					Value: "测试",
				},
				{
					Key:   "环境",
					Value: "研发",
				},
			},
			keys: []string{
				"部门", "环境",
			},
			paths: [][]string{
				{
					"技术", "测试",
				},
				{
					"技术", "研发",
				},
			},
		},
		{
			tagset: TTagSet{
				{
					Key:   "部门",
					Value: "技术",
				},
				{
					Key:   "环境",
					Value: "测试",
				},
				{
					Key:   "业务",
					Value: "TDCC",
				},
			},
			keys: []string{
				"业务", "部门", "环境",
			},
			paths: [][]string{
				{
					"TDCC", "技术", "测试",
				},
			},
		},
		{
			tagset: TTagSet{
				{
					Key:   "部门",
					Value: "技术",
				},
				{
					Key:   "环境",
					Value: "测试",
				},
				{
					Key:   "业务",
					Value: "TDCC",
				},
				{
					Key:   "业务",
					Value: "TKE",
				},
			},
			keys: []string{
				"业务", "部门", "环境",
			},
			paths: [][]string{
				{
					"TDCC", "技术", "测试",
				},
				{
					"TKE", "技术", "测试",
				},
			},
		},
		{
			tagset: TTagSet{
				{
					Key:   "部门",
					Value: "技术",
				},
				{
					Key:   "环境",
					Value: "测试",
				},
				{
					Key:   "环境",
					Value: "生产",
				},
				{
					Key:   "业务",
					Value: "TDCC",
				},
				{
					Key:   "业务",
					Value: "TKE",
				},
			},
			keys: []string{
				"业务", "部门", "环境",
			},
			paths: [][]string{
				{
					"TDCC", "技术", "测试",
				},
				{
					"TKE", "技术", "测试",
				},
				{
					"TDCC", "技术", "生产",
				},
				{
					"TKE", "技术", "生产",
				},
			},
		},
		{
			tagset: TTagSet{
				{
					Key:   "org:系统",
					Value: "G-BOOK",
				},
			},
			keys: []string{
				"org:系统", "org:业务模块", "org:環境",
			},
			paths: [][]string{
				{
					"G-BOOK",
				},
			},
		},
	}
	for _, c := range cases {
		paths := TagSet2Paths(c.tagset, c.keys)
		if jsonutils.Marshal(paths).String() != jsonutils.Marshal(c.paths).String() {
			t.Errorf("tagset: %s keys: %s want %s got %s", jsonutils.Marshal(c.tagset), jsonutils.Marshal(c.keys), jsonutils.Marshal(c.paths), jsonutils.Marshal(paths))
		}
	}
}
