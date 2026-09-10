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

	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

func TestIdpJoinAllowsSystemRole(t *testing.T) {
	cases := []struct {
		projectFromAttr bool
		roleFromAttr    bool
		want            bool
	}{
		{false, false, true},
		{true, false, false},
		{false, true, false},
		{true, true, false},
	}
	for _, c := range cases {
		got := idpJoinAllowsSystemRole(c.projectFromAttr, c.roleFromAttr)
		if got != c.want {
			t.Fatalf("projectFromAttr=%v roleFromAttr=%v got %v want %v", c.projectFromAttr, c.roleFromAttr, got, c.want)
		}
	}
}

func TestValidateIdpJoinPolicies(t *testing.T) {
	sys := rbacutils.TPolicyGroup{rbacscope.ScopeSystem: {}}
	if err := validateIdpJoinPolicies(sys, false); err == nil {
		t.Fatal("expected error for system-scope role from attributes")
	}
	if err := validateIdpJoinPolicies(sys, true); err != nil {
		t.Fatalf("configured default may assign system-scope role: %v", err)
	}
	proj := rbacutils.TPolicyGroup{rbacscope.ScopeProject: {}}
	if err := validateIdpJoinPolicies(proj, false); err != nil {
		t.Fatalf("project-scope role should be allowed: %v", err)
	}
	domain := rbacutils.TPolicyGroup{rbacscope.ScopeDomain: {}}
	if err := validateIdpJoinPolicies(domain, false); err != nil {
		t.Fatalf("domain-scope role should be allowed: %v", err)
	}
}
