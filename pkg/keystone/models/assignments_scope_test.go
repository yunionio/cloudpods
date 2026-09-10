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

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func testRoleAssignmentCred() *mcclient.SSimpleToken {
	return &mcclient.SSimpleToken{
		UserId:          "user-1",
		ProjectId:       "proj-1",
		ProjectDomainId: "domain-1",
	}
}

func TestCheckRoleAssignmentListInputSystem(t *testing.T) {
	input := api.RoleAssignmentsInput{}
	input.User.Id = "other-user"
	restrict, err := checkRoleAssignmentListInput(rbacscope.ScopeSystem, testRoleAssignmentCred(), &input)
	if err != nil {
		t.Fatalf("system scope: %v", err)
	}
	if restrict != "" {
		t.Fatalf("system restrictDomainId = %q", restrict)
	}
	if input.User.Id != "other-user" {
		t.Fatalf("system must not rewrite user id, got %q", input.User.Id)
	}
}

func TestCheckRoleAssignmentListInputUser(t *testing.T) {
	cred := testRoleAssignmentCred()
	input := api.RoleAssignmentsInput{}
	restrict, err := checkRoleAssignmentListInput(rbacscope.ScopeUser, cred, &input)
	if err != nil {
		t.Fatalf("user scope: %v", err)
	}
	if restrict != "" {
		t.Fatalf("user restrictDomainId = %q", restrict)
	}
	if input.User.Id != cred.UserId {
		t.Fatalf("user id = %q, want %q", input.User.Id, cred.UserId)
	}
	if input.IncludePolicies != nil {
		t.Fatal("user scope must not request include_policies")
	}

	input.User.Id = "other-user"
	if _, err := checkRoleAssignmentListInput(rbacscope.ScopeUser, cred, &input); err == nil {
		t.Fatal("expected error when listing another user")
	}

	input = api.RoleAssignmentsInput{}
	input.Group.Id = "group-1"
	if _, err := checkRoleAssignmentListInput(rbacscope.ScopeUser, cred, &input); err == nil {
		t.Fatal("expected error when listing by group")
	}
}

func TestCheckRoleAssignmentListInputDomain(t *testing.T) {
	cred := testRoleAssignmentCred()
	input := api.RoleAssignmentsInput{}
	restrict, err := checkRoleAssignmentListInput(rbacscope.ScopeDomain, cred, &input)
	if err != nil {
		t.Fatalf("domain scope: %v", err)
	}
	if restrict != cred.ProjectDomainId {
		t.Fatalf("restrictDomainId = %q, want %q", restrict, cred.ProjectDomainId)
	}

	input.ProjectDomainId = "other-domain"
	if _, err := checkRoleAssignmentListInput(rbacscope.ScopeDomain, cred, &input); err == nil {
		t.Fatal("expected error when listing another domain")
	}
}

func TestCheckRoleAssignmentListInputProject(t *testing.T) {
	cred := testRoleAssignmentCred()
	input := api.RoleAssignmentsInput{}
	restrict, err := checkRoleAssignmentListInput(rbacscope.ScopeProject, cred, &input)
	if err != nil {
		t.Fatalf("project scope: %v", err)
	}
	if restrict != "" {
		t.Fatalf("project restrictDomainId = %q", restrict)
	}
	if input.Scope.Project.Id != cred.ProjectId {
		t.Fatalf("project id = %q, want %q", input.Scope.Project.Id, cred.ProjectId)
	}

	input.Scope.Project.Id = "other-proj"
	if _, err := checkRoleAssignmentListInput(rbacscope.ScopeProject, cred, &input); err == nil {
		t.Fatal("expected error when listing another project")
	}
}

func TestCheckRoleAssignmentListInputNone(t *testing.T) {
	input := api.RoleAssignmentsInput{}
	if _, err := checkRoleAssignmentListInput(rbacscope.ScopeNone, testRoleAssignmentCred(), &input); err == nil {
		t.Fatal("expected error when scope is none")
	}
}
