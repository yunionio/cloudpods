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

package identity

import "yunion.io/x/onecloud/pkg/util/rbacutils"

type SIdentityObject struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type SDomainObject struct {
	SIdentityObject
	Domain SIdentityObject `json:"domain"`
}

type SFetchDomainObject struct {
	SIdentityObject
	Domain   string `json:"domain"`
	DomainId string `json:"domain_id"`
}

type SRoleAssignment struct {
	Scope struct {
		Domain  SIdentityObject `json:"domain"`
		Project SDomainObject   `json:"project"`
	} `json:"scope"`
	User  SDomainObject `json:"user"`
	Group SDomainObject `json:"group"`
	Role  SDomainObject `json:"role"`

	Policies struct {
		Project []string `json:"project"`
		Domain  []string `json:"domain"`
		System  []string `json:"system"`
	} `json:"policies"`
}

// rbacutils.IRbacIdentity interfaces

func (ra *SRoleAssignment) GetProjectId() string {
	return ra.Scope.Project.Id
}

func (ra *SRoleAssignment) GetRoleIds() []string {
	return []string{ra.Role.Id}
}

func (ra *SRoleAssignment) GetLoginIp() string {
	return ""
}

func (ra *SRoleAssignment) GetTokenString() string {
	return rbacutils.FAKE_TOKEN
}

type RAInputObject struct {
	Id string `json:"id"`
}

type RoleAssignmentsInput struct {
	User  RAInputObject `json:"user"`
	Group RAInputObject `json:"group"`
	Role  RAInputObject `json:"role"`

	Scope struct {
		Project RAInputObject `json:"project"`
		Domain  RAInputObject `json:"domain"`
	} `json:"scope"`

	Users    []string `json:"users"`
	Groups   []string `json:"groups"`
	Roles    []string `json:"roles"`
	Projects []string `json:"projects"`
	Domains  []string `json:"domains"`

	ProjectDomainId string   `json:"project_domain_id"`
	ProjectDomains  []string `json:"project_domains"`

	IncludeNames    *bool `json:"include_names"`
	Effective       *bool `json:"effective"`
	IncludeSubtree  *bool `json:"include_subtree"`
	IncludeSystem   *bool `json:"include_system"`
	IncludePolicies *bool `json:"include_policies"`

	Limit  *int `json:"limit"`
	Offset *int `json:"offset"`
}

type RoleAssignmentsOutput struct {
	RoleAssignments []SRoleAssignment `json:"role_assignments,allowempty"`

	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}
