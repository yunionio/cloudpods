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

type DomainizedResourceListInput struct {
	// swagger:ignore
	// Is an admin call? equivalent to scope=system
	// Deprecated
	Admin *bool `json:"admin"`

	// Specify query scope, either project, domain or system
	// 指定查询的权限范围，可能值为project, domain or system
	Scope string `json:"scope"`

	// Project domain filter, either by Id or Name
	// 指定查询的项目，ID or Name
	ProjectDomain string `json:"project_domain"`
	// swagger:ignore
	// Deprecated
	// Project domain Id filter, alias for project_domain
	ProjectDomainId string `json:"project_domain_id"`
	// swagger:ignore
	// Deprecated
	// Domain Id filter, alias for project_domain
	DomainId string `json:"domain_id"`
}

func (input DomainizedResourceListInput) DomainStr() string {
	if len(input.ProjectDomain) > 0 {
		return input.ProjectDomain
	}
	if len(input.ProjectDomainId) > 0 {
		return input.ProjectDomainId
	}
	if len(input.DomainId) > 0 {
		return input.DomainId
	}
	return ""
}

type DomainizedResourceCreateInput struct {
	// description: the owner domain name or id
	// required: false
	Domain string `json:"project_domain"`

	// description: the owner domain name or id, alias field of domain
	// required: false
	DomainId string `json:"domain_id"`
}

type ProjectizedResourceListInput struct {
	DomainizedResourceListInput

	// Filter by project, either ID or Name
	Project string `json:"project"`
	// swagger:ignore
	// Deprecated
	// Filter by project_id, alias for project
	ProjectId string `json:"project_id"`
	// swagger:ignore
	// Deprecated
	// Filter by tenant ID or Name, alias for project
	Tenant string `json:"tenant"`
	// swagger:ignore
	// Deprecated
	// Filter by tenant_id, alias for project
	TenantId string `json:"tenant_id"`
}

func (input ProjectizedResourceListInput) ProjectStr() string {
	if len(input.Project) > 0 {
		return input.Project
	}
	if len(input.ProjectId) > 0 {
		return input.ProjectId
	}
	if len(input.Tenant) > 0 {
		return input.Tenant
	}
	if len(input.TenantId) > 0 {
		return input.TenantId
	}
	return ""
}

type ProjectizedResourceCreateInput struct {
	DomainizedResourceCreateInput

	// description: the owner project name or id
	// required: false
	Project string `json:"project"`

	// description: the owner project name or id, alias field of project
	// required: false
	ProjectId string `json:"project_id"`
}

type UserResourceListInput struct {
	// User ID or Name
	User string `json:"user"`
	// swagger:ignore
	// Deprecated
	// Filter by userId
	UserId string `json:"user_id"`
}

func (input UserResourceListInput) UserStr() string {
	if len(input.User) > 0 {
		return input.User
	}
	if len(input.UserId) > 0 {
		return input.UserId
	}
	return ""
}

type SharableVirtualResourceCreateInput struct {
	VirtualResourceCreateInput

	// description: indicate the resource is a public resource
	// required: false
	IsPublic *bool `json:"is_public"`

	// description: indicate the shared scope for a public resource, which can be domain or system or none
	// required: false
	PublicScope string `json:"public_scope"`
}

type VirtualResourceCreateInput struct {
	StatusStandaloneResourceCreateInput
	ProjectizedResourceCreateInput

	// description: indicate the resource is a system resource, which is not visible to user
	// required: false
	IsSystem *bool `json:"is_system"`
}

type EnabledStatusStandaloneResourceCreateInput struct {
	StatusStandaloneResourceCreateInput

	// description: indicate the resource is enabled/disabled by administrator
	// required: false
	Enabled *bool `json:"enabled"`
}

type StatusStandaloneResourceCreateInput struct {
	StandaloneResourceCreateInput

	// description: the status of the resource
	// required: false
	Status string `json:"status"`
}

type StandaloneResourceCreateInput struct {
	ResourceBaseCreateInput

	// description: resource name, required if generated_name is not given
	// unique: true
	// required: true
	// example: test-network
	Name string `json:"name"`

	// description: generated resource name, given a pattern to generate name, required if name is not given
	// unique: false
	// required: false
	// example: test###
	GenerateName string `json:"generate_name"`

	// description: resource description
	// required: false
	// example: test create network
	Description string `json:"description"`

	// description: the resource is an emulated resource
	// required: false
	IsEmulated *bool `json:"is_emulated"`

	// 标签列表,最多支持20个
	// example: { "user:rd": "op" }
	Metadata map[string]string `json:"__meta__"`
}

type JoinResourceBaseCreateInput struct {
	ResourceBaseCreateInput
}

type ResourceBaseCreateInput struct {
	ModelBaseCreateInput
}

type ModelBaseCreateInput struct {
	Meta
}
