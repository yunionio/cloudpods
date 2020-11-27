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

type DomainizedResourceInput struct {
	// 指定项目归属域名称或ID
	// required: false
	ProjectDomainId string `json:"project_domain_id" help:"name or id of the belonging domain"`

	// swagger:ignore
	// Deprecated
	Domain string `json:"domain" yunion-deprecated-by:"project_domain_id"`
	// swagger:ignore
	// Deprecated
	// Project domain Id filter, alias for project_domain
	ProjectDomain string `json:"project_domain" yunion-deprecated-by:"project_domain_id"`
	// swagger:ignore
	// Deprecated
	// Domain Id filter, alias for project_domain
	DomainId string `json:"domain_id" yunion-deprecated-by:"project_domain_id"`
}

type ProjectizedResourceInput struct {
	// 指定项目的名称或ID
	// required: false
	ProjectId string `json:"project_id"`
	// swagger:ignore
	// Deprecated
	// Filter by project_id, alias for project
	Project string `json:"project" yunion-deprecated-by:"project_id"`
	// swagger:ignore
	// Deprecated
	// Filter by tenant ID or Name, alias for project
	Tenant string `json:"tenant" yunion-deprecated-by:"project_id"`
	// swagger:ignore
	// Deprecated
	// Filter by tenant_id, alias for project
	TenantId string `json:"tenant_id" yunion-deprecated-by:"project_id"`
}

type DomainizedResourceCreateInput struct {
	DomainizedResourceInput
}

type ProjectizedResourceCreateInput struct {
	DomainizedResourceInput
	ProjectizedResourceInput
}

type SharableResourceBaseCreateInput struct {
	// 是否共享
	// required: false
	IsPublic *bool `json:"is_public" token:"public" negative:"private" help:"Turn on/off public/private"`

	// 共享范围
	// required: false
	PublicScope string `json:"public_scope" help:"set public_scope, either project, domain or system" choices:"project|domain|system"`
}

type SharableVirtualResourceCreateInput struct {
	VirtualResourceCreateInput

	SharableResourceBaseCreateInput
}

type AdminSharableVirtualResourceBaseCreateInput struct {
	SharableVirtualResourceCreateInput

	// 记录
	Records string `json:"records"`
}

type StatusDomainLevelUserResourceCreateInput struct {
	StatusDomainLevelResourceCreateInput

	// 本地用户Id，若为空则使用当前用户Id作为此参数值
	OwnerId string `json:"owner_id"`
}

type UserResourceCreateInput struct {
	StandaloneResourceCreateInput

	// 本地用户Id，若为空则使用当前用户Id作为此参数值
	OwnerId string `json:"owner_id"`
}

type VirtualResourceCreateInput struct {
	StatusStandaloneResourceCreateInput
	ProjectizedResourceCreateInput

	// description: indicate the resource is a system resource, which is not visible to user
	// required: false
	IsSystem *bool `json:"is_system"`
}

type EnabledBaseResourceCreateInput struct {
	// 该资源是否被管理员*人为*启用或者禁用
	// required: false
	Enabled *bool `json:"enabled" help:"turn on enabled flag"`

	// 该资源是否被管理员*人为*禁用, 和enabled互斥
	// required: false
	Disabled *bool `json:"disabled" help:"turn off enabled flag"`
}

func (input *EnabledBaseResourceCreateInput) AfterUnmarshal() {
	if input.Disabled != nil && input.Enabled == nil {
		enabled := !(*input.Disabled)
		input.Enabled = &enabled
	}
}

type StatusBaseResourceCreateInput struct {
	// 用来存储资源的状态
	// required: false
	Status string `json:"status" help:"set initial status"`
}

type EnabledStatusDomainLevelResourceCreateInput struct {
	StatusDomainLevelResourceCreateInput
	EnabledBaseResourceCreateInput
}

type StatusDomainLevelResourceCreateInput struct {
	DomainLevelResourceCreateInput
	StatusBaseResourceCreateInput
}

type DomainLevelResourceCreateInput struct {
	StandaloneResourceCreateInput
	DomainizedResourceCreateInput
}

type EnabledStatusStandaloneResourceCreateInput struct {
	StatusStandaloneResourceCreateInput
	EnabledBaseResourceCreateInput
}

type StatusStandaloneResourceCreateInput struct {
	StandaloneResourceCreateInput
	StatusBaseResourceCreateInput
}

type StandaloneAnonResourceCreateInput struct {
	ResourceBaseCreateInput

	// 资源描述
	// required: false
	// example: test create network
	Description string `json:"description" token:"desc" help:"description"`

	// 资源是否为模拟资源
	// description: the resource is an emulated resource
	// required: false
	IsEmulated *bool `json:"is_emulated" token:"emulated" negative:"no_emulated" help:"set is_emulated flag"`

	// 标签列表,最多支持20个
	// example: { "user:rd": "op" }
	Metadata map[string]string `json:"__meta__" token:"tag" help:"tags in the form of key=value"`
}

type StandaloneResourceCreateInput struct {
	StandaloneAnonResourceCreateInput

	// 资源名称，如果generate_name为空，则为必填项
	// description: resource name, required if generated_name is not given
	// unique: true
	// required: true
	// example: test-network
	Name string `json:"name" help:"name of newly created resource" positional:"true" required:"true"`

	// 生成资源名称的模板，如果name为空，则为必填项
	// description: generated resource name, given a pattern to generate name, required if name is not given
	// unique: false
	// required: false
	// example: test###
	GenerateName string `json:"generate_name" help:"pattern for generating name if no name is given"`
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

type PerformStatusInput struct {
	// 更改的目标状态值
	// required:true
	Status string `json:"status"`

	// 更改状态的原因描述
	// required:false
	Reason string `json:"reason"`
}

type GetDetailsStatusOutput struct {
	// 状态
	Status string `json:"status"`
}

type PerformPublicDomainInput struct {
	// 共享项目资源的共享范围，可能的值为：project, domain和system
	// pattern: project|domain|system
	Scope string `json:"scope"`

	// 如果共享范围为域，则在此列表中指定共享的目标域
	SharedDomainIds []string `json:"shared_domain_ids"`
	// Deprecated
	// swagger:ignore
	SharedDomains []string `json:"shared_domains" yunion-deprecated-by:"shared_domain_ids"`
}

type PerformPublicProjectInput struct {
	PerformPublicDomainInput

	// 如果共享范围为项目，则在此列表中指定共享的目标项目
	SharedProjectIds []string `json:"shared_project_ids"`
	// Deprecated
	// swagger:ignore
	SharedProjects []string `json:"shared_projects" yunion-deprecated-by:"shared_project_ids"`
}

type PerformPrivateInput struct {
}

type PerformChangeProjectOwnerInput struct {
	ProjectizedResourceInput
}

type PerformFreezeInput struct {
}

type PerformUnfreezeInput struct {
}

type PerformChangeDomainOwnerInput struct {
	DomainizedResourceInput
}

type PerformEnableInput struct {
}

type PerformDisableInput struct {
}

type StorageForceDetachHostInput struct {
	// Host id or name
	HostId string `json:"host_id"`
	// Deprecated
	// swagger:ignore
	Host string `json:"host" yunion-deprecated-by:"host_id"`
}

type InfrasResourceBaseCreateInput struct {
	DomainLevelResourceCreateInput

	SharableResourceBaseCreateInput
}

type StatusInfrasResourceBaseCreateInput struct {
	InfrasResourceBaseCreateInput
	StatusBaseResourceCreateInput
}

type EnabledStatusInfrasResourceBaseCreateInput struct {
	StatusInfrasResourceBaseCreateInput
	EnabledBaseResourceCreateInput
}

type ScopedResourceCreateInput struct {
	ProjectizedResourceCreateInput
	Scope string `json:"scope"`
}

type OpsLogCreateInput struct {
	ModelBaseCreateInput

	ObjType string `json:"obj_type"`
	ObjId   string `json:"obj_id"`
	ObjName string `json:"obj_name"`
	Action  string `json:"action"`
	Notes   string `json:"notes"`

	ProjectId string `json:"tenant_id"`
	Project   string `json:"tenant"`

	ProjectDomainId string `json:"project_domain_id"`
	ProjectDomain   string `json:"project_domain"`

	UserId   string `json:"user_id"`
	User     string `json:"user"`
	DomainId string `json:"domain_id"`
	Domain   string `json:"domain"`
	Roles    string `json:"roles"`

	OwnerDomainId  string `json:"owner_domain_id"`
	OwnerProjectId string `json:"owner_tenant_id"`
}

// 设置资源的标签（元数据）输入
type PerformMetadataInput map[string]string

// 设置资源的用户标签（元数据）输入
type PerformUserMetadataInput map[string]string

// 全量替换资源的用户标签（元数据）输入
type PerformSetUserMetadataInput map[string]string

// 获取资源的元数据输入
type GetMetadataInput struct {
	// 指定需要获取的所有标签的KEY列表，如果列表为空，则获取全部标签
	// 标签分为
	//
	// | 类型     | 说明                                        |
	// |----------|---------------------------------------------|
	// | 系统标签 | 平台定义的标签                              |
	// | 用户标签 | key以user:为前缀，用户自定义标签            |
	// | 外部标签 | key以ext:为前缀，为从其他平台同步过来的标签 |
	//
	Field []string `json:"field"`

	// 按标签前缀过滤
	Prefix string `json:"prefix"`
}

// 获取资源标签（元数据）输出
type GetMetadataOutput map[string]string
