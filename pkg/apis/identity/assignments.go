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
	// UUID
	Id string `json:"id"`
	// 名称
	Name string `json:"name"`
}

type SDomainObject struct {
	SIdentityObject

	// 归属域信息
	Domain SIdentityObject `json:"domain"`
}

type SDomainObjectWithMetadata struct {
	SDomainObject

	// 标签信息
	Metadata map[string]string `json:"metadata"`
}

type SFetchDomainObject struct {
	SIdentityObject
	// 归属域名称
	Domain string `json:"domain"`
	// 归属域ID
	DomainId string `json:"domain_id"`
}

type SFetchDomainObjectWithMetadata struct {
	SFetchDomainObject

	// 项目标签
	Metadata map[string]string `json:"metadata"`
}

type SRoleAssignment struct {
	// 归属范围
	Scope struct {
		// 归属域信息
		Domain SIdentityObject `json:"domain"`
		// 归属项目信息，归属范围为项目时有值
		Project SDomainObjectWithMetadata `json:"project"`
	} `json:"scope"`

	// 用户信息
	User SDomainObject `json:"user"`
	// 用户组信息
	Group SDomainObject `json:"group"`
	// 用户加入项目的角色信息
	Role SDomainObject `json:"role"`

	// 用户角色关联的权限信息
	Policies struct {
		// 关联的项目权限名称列表
		Project []string `json:"project"`
		// 关联的域权限名称列表
		Domain []string `json:"domain"`
		// 关联的系统权限名称列表
		System []string `json:"system"`
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
