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
	"testing"

	"yunion.io/x/pkg/util/rbacscope"
)

func TestSamluserResourceScope(t *testing.T) {
	if got := SamluserManager.ResourceScope(); got != rbacscope.ScopeUser {
		t.Fatalf("ResourceScope = %s, want %s", got, rbacscope.ScopeUser)
	}
}

func TestEffectiveSamluserListScope(t *testing.T) {
	cases := []struct {
		name        string
		reqScope    string
		isAdmin     bool
		domainAllow bool
		wantScope   string
		wantSet     bool
	}{
		{name: "domain reader keeps domain default", reqScope: "", domainAllow: true, wantScope: string(rbacscope.ScopeDomain), wantSet: true},
		{name: "system reader keeps domain default", reqScope: "", domainAllow: true, wantScope: string(rbacscope.ScopeDomain), wantSet: true},
		{name: "ordinary user keeps user scope", reqScope: "", domainAllow: false, wantSet: false},
		{name: "explicit user scope stays", reqScope: "user", domainAllow: false, wantSet: false},
		{name: "explicit domain scope stays for rbac to reject", reqScope: "domain", domainAllow: false, wantSet: false},
		{name: "admin=true is not rewritten", reqScope: "", isAdmin: true, domainAllow: true, wantSet: false},
		{name: "none scope uses domain default for domain reader", reqScope: "none", domainAllow: true, wantScope: string(rbacscope.ScopeDomain), wantSet: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := effectiveSamluserListScope(c.reqScope, c.isAdmin, c.domainAllow)
			if ok != c.wantSet || got != c.wantScope {
				t.Fatalf("effectiveSamluserListScope(%q, %v, %v) = (%q, %v), want (%q, %v)", c.reqScope, c.isAdmin, c.domainAllow, got, ok, c.wantScope, c.wantSet)
			}
		})
	}
}

func TestSamluserListOwnerConstraint(t *testing.T) {
	owner, restrict := samluserListOwnerConstraint("user-a", false)
	if !restrict || owner != "user-a" {
		t.Fatalf("ordinary user constraint = (%q, %v), want (user-a, true)", owner, restrict)
	}
	owner, restrict = samluserListOwnerConstraint("admin", true)
	if restrict || owner != "" {
		t.Fatalf("domain reader constraint = (%q, %v), want (\"\", false)", owner, restrict)
	}
}
