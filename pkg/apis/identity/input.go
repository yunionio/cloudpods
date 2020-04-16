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

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
)

type IdentityBaseResourceCreateInput struct {
	apis.StandaloneResourceCreateInput
}

type EnabledIdentityBaseResourceCreateInput struct {
	IdentityBaseResourceCreateInput

	Enabled *bool `json:"enabled"`
}

type IdentityBaseResourceListInput struct {
	apis.StandaloneResourceListInput
	apis.DomainizedResourceListInput
}

type EnabledIdentityBaseResourceListInput struct {
	IdentityBaseResourceListInput

	// filter by enabled status
	Enabled *bool `json:"enabled"`
}

type ProjectFilterListInput struct {
	// 项目归属域
	ProjectDomain string `json:"project_domain"`
	// swagger:ignore
	// Deprecated
	ProjectDomainId string `json:"project_domain_id" "yunion:deprecated-by":"project-domain"`

	// 以项目（ID或Name）过滤列表结果
	Project string `json:"project"`
	// swagger:ignore
	// Deprecated
	// filter by project_id
	ProjectId string `json:"project_id" "yunion:deprecated-by":"project"`
	// swagger:ignore
	// Deprecated
	// filter by tenant
	Tenant string `json:"tenant" "yunion:deprecated-by":"project"`
	// swagger:ignore
	// Deprecated
	// filter by tenant_id
	TenantId string `json:"tenant_id" "yunion:deprecated-by":"project"`
}

type UserFilterListInput struct {
	// 用户归属域
	UserDomain string `json:"user_domain"`
	// swagger:ignore
	// Deprecated
	UserDomainId string `json:"user_domain_id"`

	// filter by user
	User string `json:"user"`
	// swagger:ignore
	// Deprecated
	// filter by user_id
	UserId string `json:"user_id" "yunion:deprecated-by":"user"`
}

type GroupFilterListInput struct {
	// 组归属域
	GroupDomain string `json:"group_domain"`
	// swagger:ignore
	// Deprecated
	GroupDomainId string `json:"group_domain_id"`

	// filter by group
	Group string `json:"group"`
	// swagger:ignore
	// Deprecated
	// filter by group_id
	GroupId string `json:"group_id" "yunion:deprecated-by":"group"`
}

type RoleFilterListInput struct {
	// 角色归属域
	RoleDomain string `json:"role_domain"`
	// swagger:ignore
	// Deprecated
	RoleDomainId string `json:"role_domain_id"`

	// filter by role
	Role string `json:"role"`
	// swagger:ignore
	// Deprecated
	// filter by role_id
	RoleId string `json:"role_id" "yunion:deprecated-by":"role"`
}

type ServiceFilterListInput struct {
	// 服务类型过滤
	ServiceType string `json:"service_type"`

	// 服务名称或ID过滤
	Service string `json:"service"`
	// swagger:ignore
	// Deprecated
	// filter by service_id
	ServiceId string `json:"service_id" "yunion:deprecated-by":"service"`

	// 以服务名称排序
	OrderByService string `json:"order_by_service"`
}

type RoleListInput struct {
	IdentityBaseResourceListInput
	apis.SharableResourceBaseListInput

	ProjectFilterListInput
	UserFilterListInput
	GroupFilterListInput
}

type GroupListInput struct {
	IdentityBaseResourceListInput

	UserFilterListInput
	ProjectFilterListInput

	// 名称过滤
	Displayname string `json:"displayname"`
}

type ProjectListInput struct {
	IdentityBaseResourceListInput

	UserFilterListInput
	GroupFilterListInput
}

type DomainListInput struct {
	apis.StandaloneResourceListInput

	Enabled *bool `json:"enabled"`
}

type UserListInput struct {
	EnabledIdentityBaseResourceListInput

	GroupFilterListInput
	ProjectFilterListInput
	RoleFilterListInput

	// email
	Email string `json:"email"`
	// mobile
	Mobile string `json:"mobile"`
	// displayname
	Displayname string `json:"displayname"`

	// 是否允许web控制台登录
	AllowWebConsole *bool `json:"allow_web_console"`

	// 是否开启MFA认证
	EnableMfa *bool `json:"enable_mfa"`
}

type EndpointListInput struct {
	apis.StandaloneResourceListInput

	ServiceFilterListInput
	RegionFilterListInput

	// 以Endpoint接口类型过滤，可能值为: internal, internalURL, public, publicURL, admin, adminURL, console
	Interface string `json:"interface"`

	// 是否启用
	Enabled *bool `json:"enabled"`
}

type SJoinProjectsInput struct {
	Projects []string
	Roles    []string
}

func (input SJoinProjectsInput) Validate() error {
	if len(input.Projects) == 0 {
		return errors.Error("empty projects")
	}
	if len(input.Roles) == 0 {
		return errors.Error("empty roles")
	}
	return nil
}

type SProjectRole struct {
	Project string
	Role    string
}
type SLeaveProjectsInput struct {
	ProjectRoles []SProjectRole
}

func (input SLeaveProjectsInput) Validate() error {
	if len(input.ProjectRoles) == 0 {
		return errors.Error("empty project_roles")
	}
	for i := range input.ProjectRoles {
		if len(input.ProjectRoles[i].Project) == 0 {
			return errors.Error("no project in project_roles")
		}
		if len(input.ProjectRoles[i].Role) == 0 {
			return errors.Error("no role in project_roles")
		}
	}
	return nil
}

type SProjectAddUserGroupInput struct {
	Users  []string
	Groups []string
	Roles  []string
}

func (input SProjectAddUserGroupInput) Validate() error {
	if len(input.Users) == 0 && len(input.Groups) == 0 {
		return errors.Error("empty user and group")
	}
	if len(input.Roles) == 0 {
		return errors.Error("invalid roles")
	}
	return nil
}

type SUserRole struct {
	User string
	Role string
}
type SGroupRole struct {
	Group string
	Role  string
}

type SProjectRemoveUserGroupInput struct {
	UserRoles  []SUserRole
	GroupRoles []SGroupRole
}

func (input SProjectRemoveUserGroupInput) Validate() error {
	if len(input.UserRoles) == 0 && len(input.GroupRoles) == 0 {
		return errors.Error("empty input")
	}
	for i := range input.UserRoles {
		if len(input.UserRoles[i].User) == 0 {
			return errors.Error("empty user")
		}
		if len(input.UserRoles[i].Role) == 0 {
			return errors.Error("empty role")
		}
	}
	for i := range input.GroupRoles {
		if len(input.GroupRoles[i].Group) == 0 {
			return errors.Error("empty group")
		}
		if len(input.GroupRoles[i].Role) == 0 {
			return errors.Error("empty role")
		}
	}
	return nil
}

type IdentityProviderListInput struct {
	apis.EnabledStatusStandaloneResourceListInput

	// 以驱动类型过滤
	Driver []string `json:"driver"`

	// 以模板过滤
	Template []string `json:"template"`

	// 以同步状态过滤
	SyncStatus []string `json:"sync_status"`
}

type CredentialListInput struct {
	apis.StandaloneResourceListInput

	UserFilterListInput
	ProjectFilterListInput

	Type []string `json:"type"`

	Enabled *bool `json:"enabled"`
}

type PolicyListInput struct {
	EnabledIdentityBaseResourceListInput
	apis.SharableResourceBaseListInput

	// 以类型查询
	Type []string `json:"type"`
}

type RegionFilterListInput struct {
	// 以区域名称或ID过滤
	Region string `json:"region"`
	// swagger:ignore
	// Deprecated
	RegionId string `json:"region_id" "yunion:deprecated-by":"region"`
}

type RegionListInput struct {
	apis.StandaloneResourceListInput
}

type ServiceListInput struct {
	apis.StandaloneResourceListInput

	// 以Service Type过滤
	Type []string `json:"type"`

	// 是否启用/禁用
	Enabled *bool `json:"enabled"`
}

type IdentityBaseUpdateInput struct {
	apis.StandaloneResourceBaseUpdateInput
}

type EnabledIdentityBaseUpdateInput struct {
	IdentityBaseUpdateInput

	// 是否启用
	Enabled *bool `json:"enabled"`
}

type GroupUpdateInput struct {
	IdentityBaseUpdateInput

	// display name
	Displayname string `json:"displayname"`
}

type IdentityProviderUpdateInput struct {
	apis.EnabledStatusStandaloneResourceBaseUpdateInput

	TargetDomainId string `json:"target_domain_id"`

	AutoCreateProject *bool `json:"auto_create_project"`

	SyncIntervalSeconds *int `json:"sync_interval_seconds"`
}

type PolicyUpdateInput struct {
	EnabledIdentityBaseUpdateInput

	Type string `json:"type"`

	Blob jsonutils.JSONObject `json:"blob"`
}

type ProjectUpdateInput struct {
	IdentityBaseUpdateInput

	// 显示名称
	Displayname string `json:"displayname"`
}

type RoleUpdateInput struct {
	IdentityBaseUpdateInput
}

type UserUpdateInput struct {
	EnabledIdentityBaseUpdateInput

	Email string `json:"email"`

	Mobile string `json:"mobile"`

	Displayname string `json:"displayname"`

	IsSystemAccount *bool `json:"is_system_account"`

	AllowWebConsole *bool `json:"allow_web_console"`

	EnableMfa *bool `json:"enable_mfa"`

	Password string `json:"password"`
}

type UserCreateInput struct {
	EnabledIdentityBaseResourceCreateInput

	Email string `json:"email"`

	Mobile string `json:"mobule"`

	Displayname string `json:"displayname"`

	IsSystemAccount *bool `json:"is_system_account"`

	AllowWebConsole *bool `json:"allow_web_console"`

	EnableMfa *bool `json:"enable_mfa"`

	Password string `json:"password"`

	SkipPasswordComplexityCheck *bool `json:"skip_password_complexity_check"`
}

type ProjectCreateInput struct {
	IdentityBaseResourceCreateInput

	// 显示名称
	Displayname string `json:"displayname"`
}

type GroupCreateInput struct {
	IdentityBaseResourceCreateInput

	// display name
	Displayname string `json:"displayname"`
}

type PolicyCreateInput struct {
	EnabledIdentityBaseResourceCreateInput
	apis.SharableResourceBaseCreateInput

	Type string `json:"type"`

	Blob jsonutils.JSONObject `json:"blob"`
}

type RoleCreateInput struct {
	IdentityBaseResourceCreateInput

	apis.SharableResourceBaseCreateInput
}
