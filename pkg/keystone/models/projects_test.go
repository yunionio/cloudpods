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

package models

import (
	"strings"
	"testing"
)

func TestOrganizationTagsFromUserTags(t *testing.T) {
	keys := []string{"部门", "小组"}
	labels, orgTags, touched := organizationTagsFromUserTags(keys, map[string]string{
		"部门":  "销售",
		"小组":  "华北",
		"负责人": "alice",
	})
	if !touched {
		t.Fatalf("touched = false, want true")
	}
	if got, want := apiJoin(labels), "销售/华北"; got != want {
		t.Fatalf("labels = %s, want %s", got, want)
	}
	if !sameStringMap(orgTags, map[string]string{"部门": "销售", "小组": "华北"}) {
		t.Fatalf("orgTags = %#v", orgTags)
	}

	labels, orgTags, touched = organizationTagsFromUserTags(keys, map[string]string{
		"部门": "研发",
		"小组": "none",
	})
	if !touched || apiJoin(labels) != "研发" || len(orgTags) != 1 || orgTags["部门"] != "研发" {
		t.Fatalf("stop at cleared child: labels=%v orgTags=%v touched=%v", labels, orgTags, touched)
	}

	_, _, touched = organizationTagsFromUserTags(keys, map[string]string{"负责人": "bob"})
	if touched {
		t.Fatalf("unrelated tags should not touch organization binding")
	}

	_, _, touched = organizationTagsFromUserTags(keys, map[string]string{"小组": "后端"})
	if touched {
		t.Fatalf("missing top key should keep the current binding")
	}

	labels, orgTags, touched = organizationTagsFromUserTags(keys, map[string]string{"部门": ""})
	if !touched || len(labels) != 0 || len(orgTags) != 0 {
		t.Fatalf("empty top key should clear path: labels=%v orgTags=%v touched=%v", labels, orgTags, touched)
	}

	labels, orgTags, touched = organizationTagsFromUserTags(keys, map[string]string{"部门": "none"})
	if !touched || len(labels) != 0 || len(orgTags) != 0 {
		t.Fatalf("none top key should clear path: labels=%v orgTags=%v touched=%v", labels, orgTags, touched)
	}
}

func TestRewriteInactiveOrganizationUserTags(t *testing.T) {
	keys := []string{"部门", "小组"}
	out := rewriteInactiveOrganizationUserTags(keys, map[string]string{
		"部门":  "none",
		"小组":  "null",
		"foo": "bar",
	})
	if out["部门"] != "" || out["小组"] != "" || out["foo"] != "bar" {
		t.Fatalf("inactive org keys should become empty: %#v", out)
	}

	out = rewriteInactiveOrganizationUserTags(keys, map[string]string{
		"user:部门": "none",
		"foo":     "bar",
	})
	if out["user:部门"] != "" || out["foo"] != "bar" {
		t.Fatalf("prefixed inactive org key should become empty: %#v", out)
	}

	in := map[string]string{"部门": "销售", "foo": "bar"}
	out = rewriteInactiveOrganizationUserTags(keys, in)
	if out["部门"] != "销售" || out["foo"] != "bar" {
		t.Fatalf("active tags should stay unchanged: %#v", out)
	}
}

func apiJoin(labels []string) string {
	return strings.Join(labels, "/")
}

func TestNormalizeProjectName(t *testing.T) {
	cases := []struct {
		In   string
		Want string
	}{
		{"分公司1", "fengongsi1"},
		{"集团/分公司/项目A", "jituan-fengongsi-xiangmua"},
	}
	for _, c := range cases {
		got := NormalizeProjectName(c.In)
		if got != c.Want {
			t.Errorf("NormalizeProjectName %s got %s want %s", c.In, got, c.Want)
		}
	}
}
