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

package apis

import (
	"testing"

	"yunion.io/x/pkg/util/rbacscope"
)

func TestSShareInfo_IsViolate(t *testing.T) {
	cases := []struct {
		name     string
		s1       SShareInfo
		s2       SShareInfo
		violate1 bool
		violate2 bool
	}{
		{
			name: "case1",
			s1: SShareInfo{
				IsPublic:    false,
				PublicScope: rbacscope.ScopeNone,
			},
			s2: SShareInfo{
				IsPublic:    true,
				PublicScope: rbacscope.ScopeSystem,
			},
			violate1: false,
			violate2: true,
		},
		{
			name: "case2",
			s1: SShareInfo{
				IsPublic:       true,
				PublicScope:    rbacscope.ScopeProject,
				SharedProjects: []string{"p1"},
			},
			s2: SShareInfo{
				IsPublic:       true,
				PublicScope:    rbacscope.ScopeProject,
				SharedProjects: []string{"p2"},
			},
			violate1: true,
			violate2: true,
		},
		{
			name: "case3",
			s1: SShareInfo{
				IsPublic:      true,
				PublicScope:   rbacscope.ScopeDomain,
				SharedDomains: []string{"p1"},
			},
			s2: SShareInfo{
				IsPublic:       true,
				PublicScope:    rbacscope.ScopeProject,
				SharedProjects: []string{"p2"},
			},
			violate1: true,
			violate2: false,
		},
		{
			name: "case4",
			s1: SShareInfo{
				IsPublic:      true,
				PublicScope:   rbacscope.ScopeDomain,
				SharedDomains: []string{"p1", "p2"},
			},
			s2: SShareInfo{
				IsPublic:      true,
				PublicScope:   rbacscope.ScopeDomain,
				SharedDomains: []string{"p1", "p2"},
			},
			violate1: false,
			violate2: false,
		},
	}
	for _, c := range cases {
		if c.s1.IsViolate(c.s2) != c.violate1 {
			t.Errorf("[%s] s1.violate(s2) want %v", c.name, c.violate1)
		} else if c.s2.IsViolate(c.s1) != c.violate2 {
			t.Errorf("[%s] s2.violate(s1) want: %v", c.name, c.violate2)
		}
	}
}

func TestSShareInfo_Intersect(t *testing.T) {
	cases := []struct {
		name string
		s1   SShareInfo
		s2   SShareInfo
		want SShareInfo
	}{
		{
			name: "case1",
			s1: SShareInfo{
				IsPublic:    false,
				PublicScope: rbacscope.ScopeNone,
			},
			s2: SShareInfo{
				IsPublic:    true,
				PublicScope: rbacscope.ScopeSystem,
			},
			want: SShareInfo{
				IsPublic:    false,
				PublicScope: rbacscope.ScopeNone,
			},
		},
		{
			name: "case2",
			s1: SShareInfo{
				IsPublic:       true,
				PublicScope:    rbacscope.ScopeProject,
				SharedProjects: []string{"p1"},
			},
			s2: SShareInfo{
				IsPublic:       true,
				PublicScope:    rbacscope.ScopeProject,
				SharedProjects: []string{"p2"},
			},
			want: SShareInfo{
				IsPublic:    false,
				PublicScope: rbacscope.ScopeNone,
			},
		},
		{
			name: "case3",
			s1: SShareInfo{
				IsPublic:      true,
				PublicScope:   rbacscope.ScopeDomain,
				SharedDomains: []string{"p1"},
			},
			s2: SShareInfo{
				IsPublic:       true,
				PublicScope:    rbacscope.ScopeProject,
				SharedProjects: []string{"p2"},
			},
			want: SShareInfo{
				IsPublic:       true,
				PublicScope:    rbacscope.ScopeProject,
				SharedProjects: []string{"p2"},
			},
		},
		{
			name: "case4",
			s1: SShareInfo{
				IsPublic:      true,
				PublicScope:   rbacscope.ScopeDomain,
				SharedDomains: []string{"p1", "p2"},
			},
			s2: SShareInfo{
				IsPublic:      true,
				PublicScope:   rbacscope.ScopeDomain,
				SharedDomains: []string{"p1", "p2"},
			},
			want: SShareInfo{
				IsPublic:      true,
				PublicScope:   rbacscope.ScopeDomain,
				SharedDomains: []string{"p1", "p2"},
			},
		},
		{
			name: "case5",
			s1: SShareInfo{
				IsPublic:      true,
				PublicScope:   rbacscope.ScopeDomain,
				SharedDomains: []string{"p1", "p2", "p4"},
			},
			s2: SShareInfo{
				IsPublic:      true,
				PublicScope:   rbacscope.ScopeDomain,
				SharedDomains: []string{"p1", "p2", "p3"},
			},
			want: SShareInfo{
				IsPublic:      true,
				PublicScope:   rbacscope.ScopeDomain,
				SharedDomains: []string{"p1", "p2"},
			},
		},
	}
	for _, c := range cases {
		inter := c.s1.Intersect(c.s2)
		if !inter.Equals(c.want) {
			t.Errorf("[%s] intersect got %#v != want %#v", c.name, inter, c.want)
		}
	}
}
