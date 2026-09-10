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
	"context"
	"testing"
	"time"

	"yunion.io/x/onecloud/pkg/mcclient"
)

func TestNormalizeRolePolicyBindingRequiresRoleAndPolicy(t *testing.T) {
	ctx := context.Background()
	userCred := &mcclient.SSimpleToken{}
	_, _, _, err := normalizeRolePolicyBinding(ctx, userCred, "", "project-1", "policy-1")
	if err == nil {
		t.Fatal("empty role_id: expected error")
	}
	_, _, _, err = normalizeRolePolicyBinding(ctx, userCred, "   ", "project-1", "policy-1")
	if err == nil {
		t.Fatal("whitespace role_id: expected error")
	}
	_, _, _, err = normalizeRolePolicyBinding(ctx, userCred, "role-1", "project-1", "")
	if err == nil {
		t.Fatal("empty policy_id: expected error")
	}
	_, _, _, err = normalizeRolePolicyBinding(ctx, userCred, "role-1", "project-1", "  ")
	if err == nil {
		t.Fatal("whitespace policy_id: expected error")
	}
}

func TestGetMatchPolicyIds2NoRoles(t *testing.T) {
	ids, err := RolePolicyManager.getMatchPolicyIds2(false, nil, "project-1", "", time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("got %v, want empty", ids)
	}
}
