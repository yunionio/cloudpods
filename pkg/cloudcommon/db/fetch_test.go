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

package db

import (
	"testing"

	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
)

func TestResolveQueryScopeFallback(t *testing.T) {
	cases := []struct {
		name       string
		reqScope   string
		hasAdmin   bool
		isAdmin    bool
		allowScope rbacscope.TRbacScope
		resScope   rbacscope.TRbacScope
	}{
		{"admin_true_project_caller", "", true, true, rbacscope.ScopeProject, rbacscope.ScopeProject},
		{"admin_true_project_caller_domain_resource", "", true, true, rbacscope.ScopeProject, rbacscope.ScopeDomain},
		{"admin_true_user_caller", "", true, true, rbacscope.ScopeUser, rbacscope.ScopeProject},
		{"admin_false_project_caller", "", true, false, rbacscope.ScopeProject, rbacscope.ScopeProject},
		{"admin_false_system_caller", "", true, false, rbacscope.ScopeSystem, rbacscope.ScopeProject},
		{"scope_user_on_project_resource", "user", false, false, rbacscope.ScopeProject, rbacscope.ScopeProject},
		{"scope_user_on_domain_resource", "user", false, false, rbacscope.ScopeDomain, rbacscope.ScopeDomain},
		{"scope_max_no_permission", "max", false, false, rbacscope.ScopeNone, rbacscope.ScopeProject},
		{"scope_max_no_permission_domain_resource", "max", false, false, rbacscope.ScopeNone, rbacscope.ScopeDomain},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := resolveQueryScope(c.reqScope, c.hasAdmin, c.isAdmin, c.allowScope, c.resScope, policy.PolicyActionList)
			if got == "" || got == rbacscope.ScopeNone {
				t.Fatalf("queryScope = %q, want a real scope (never unset/none)", got)
			}
			if got == rbacscope.ScopeUser && c.resScope != rbacscope.ScopeUser {
				t.Fatalf("queryScope = user on a %s-scoped resource, want the resource scope", c.resScope)
			}
			if got != c.resScope {
				t.Fatalf("queryScope = %q, want fallback to resource scope %q", got, c.resScope)
			}
		})
	}
}

// the supported explicit views and fallbacks keep working
func TestResolveQueryScopeSemantics(t *testing.T) {
	cases := []struct {
		name       string
		reqScope   string
		hasAdmin   bool
		isAdmin    bool
		allowScope rbacscope.TRbacScope
		resScope   rbacscope.TRbacScope
		action     string
		want       rbacscope.TRbacScope
	}{
		// explicit views
		{"explicit_system", "system", false, false, rbacscope.ScopeSystem, rbacscope.ScopeProject, policy.PolicyActionList, rbacscope.ScopeSystem},
		{"explicit_domain", "domain", false, false, rbacscope.ScopeDomain, rbacscope.ScopeProject, policy.PolicyActionList, rbacscope.ScopeDomain},
		{"explicit_project_on_domain_resource", "project", false, false, rbacscope.ScopeProject, rbacscope.ScopeDomain, policy.PolicyActionList, rbacscope.ScopeProject},
		{"explicit_user_on_user_resource", "user", false, false, rbacscope.ScopeUser, rbacscope.ScopeUser, policy.PolicyActionList, rbacscope.ScopeUser},
		{"unknown_scope_string_defaults_to_project", "garbage", false, false, rbacscope.ScopeProject, rbacscope.ScopeProject, policy.PolicyActionList, rbacscope.ScopeProject},
		// max = widest allowed scope
		{"max_for_system_caller", "max", false, false, rbacscope.ScopeSystem, rbacscope.ScopeProject, policy.PolicyActionList, rbacscope.ScopeSystem},
		{"max_for_project_caller", "max", false, false, rbacscope.ScopeProject, rbacscope.ScopeProject, policy.PolicyActionList, rbacscope.ScopeProject},
		// admin view for callers above project scope
		{"admin_true_system_caller", "", true, true, rbacscope.ScopeSystem, rbacscope.ScopeProject, policy.PolicyActionList, rbacscope.ScopeSystem},
		{"admin_true_domain_caller", "", true, true, rbacscope.ScopeDomain, rbacscope.ScopeProject, policy.PolicyActionList, rbacscope.ScopeDomain},
		// default views
		{"no_params_project_resource", "", false, false, rbacscope.ScopeProject, rbacscope.ScopeProject, policy.PolicyActionList, rbacscope.ScopeProject},
		{"no_params_domain_resource", "", false, false, rbacscope.ScopeDomain, rbacscope.ScopeDomain, policy.PolicyActionList, rbacscope.ScopeDomain},
		{"no_params_user_resource", "", false, false, rbacscope.ScopeUser, rbacscope.ScopeUser, policy.PolicyActionList, rbacscope.ScopeUser},
		{"get_action_uses_allow_scope", "", false, false, rbacscope.ScopeProject, rbacscope.ScopeProject, policy.PolicyActionGet, rbacscope.ScopeProject},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := resolveQueryScope(c.reqScope, c.hasAdmin, c.isAdmin, c.allowScope, c.resScope, c.action)
			if got != c.want {
				t.Fatalf("queryScope = %q, want %q", got, c.want)
			}
		})
	}
}
