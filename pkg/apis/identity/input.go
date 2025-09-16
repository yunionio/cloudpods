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
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/util/tagutils"
)

type IdentityBaseResourceCreateInput struct {
	apis.StandaloneResourceCreateInput
	apis.DomainizedResourceCreateInput
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

	apis.EnabledResourceBaseListInput
}

type ProjectFilterListInput struct {
	// 项目归属域
	ProjectDomainId string `json:"project_domain_id"`
	// swagger:ignore
	// Deprecated
	ProjectDomain string `json:"project_domain" yunion-deprecated-by:"project_domain_id"`

	// 以项目（ID或Name）过滤列表结果
	ProjectId string `json:"project_id"`
	// swagger:ignore
	// Deprecated
	// filter by project_id
	Project string `json:"project" yunion-deprecated-by:"project_id"`
	// swagger:ignore
	// Deprecated
	// filter by tenant
	Tenant string `json:"tenant" yunion-deprecated-by:"project_id"`
	// swagger:ignore
	// Deprecated
	// filter by tenant_id
	TenantId string `json:"tenant_id" yunion-deprecated-by:"project_id"`
}

type UserFilterListInput struct {
	// 用户归属域
	UserDomainId string `json:"user_domain_id"`
	// swagger:ignore
	// Deprecated
	UserDomain string `json:"user_domain" yunion-deprecated-by:"user_domain_id"`

	// filter by user
	UserId string `json:"user_id"`
	// swagger:ignore
	// Deprecated
	// filter by user_id
	User string `json:"user" yunion-deprecated-by:"user_id"`
}

type GroupFilterListInput struct {
	// 组归属域
	GroupDomainId string `json:"group_domain_id"`
	// swagger:ignore
	// Deprecated
	GroupDomain string `json:"group_domain" yunion-deprecated-by:"group_domain_id"`

	// filter by group
	GroupId string `json:"group_id"`
	// swagger:ignore
	// Deprecated
	// filter by group_id
	Group string `json:"group" yunion-deprecated-by:"group_id"`
}

type RoleFilterListInput struct {
	// 角色归属域
	RoleDomainId string `json:"role_domain_id"`
	// swagger:ignore
	// Deprecated
	RoleDomain string `json:"role_domain" yunion-deprecated-by:"role_domain_id"`

	// filter by role
	RoleId string `json:"role_id"`
	// swagger:ignore
	// Deprecated
	// filter by role_id
	Role string `json:"role" yunion-deprecated-by:"role_id"`
}

type ServiceFilterListInput struct {
	// 服务类型过滤
	ServiceType string `json:"service_type"`

	// 服务名称或ID过滤
	ServiceId string `json:"service_id"`
	// swagger:ignore
	// Deprecated
	// filter by service_id
	Service string `json:"service" yunion-deprecated-by:"service_id"`

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

	// 按IDP过滤
	IdpId string `json:"idp_id"`
}

type ProjectListInput struct {
	IdentityBaseResourceListInput

	UserFilterListInput
	GroupFilterListInput

	// filter projects by Identity Provider
	IdpId string `json:"idp_id"`

	// 过滤出指定用户或者组可以加入的项目
	Jointable *bool `json:"jointable"`

	// project tags filter imposed by policy
	PolicyProjectTags tagutils.TTagSetList `json:"policy_project_tags"`

	// 通过项目管理员id过滤
	AdminId []string `json:"admin_id"`
}

type DomainListInput struct {
	apis.StandaloneResourceListInput

	Enabled *bool `json:"enabled"`

	// 按IDP过滤
	IdpId string `json:"idp_id"`

	// 按IDP_ENTITY_ID过滤
	IdpEntityId string `json:"idp_entity_id"`

	// domain tags filter imposed by policy
	PolicyDomainTags tagutils.TTagSetList `json:"policy_domain_tags"`
}

type UserListInput struct {
	EnabledIdentityBaseResourceListInput

	GroupFilterListInput
	ProjectFilterListInput
	RoleFilterListInput

	// 角色生效所在的域
	RoleAssignmentDomainId string `json:"role_assignment_domain_id"`
	// 角色生效所在的项目
	RoleAssignmentProjectId string `json:"role_assignment_project_id"`

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

	// 关联IDP
	IdpId string `json:"idp_id"`

	// 按IDP_ENTITY_ID过滤
	IdpEntityId string `json:"idp_entity_id"`
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
	Projects []string `json:"projects"`
	Roles    []string `json:"roles"`
	// 启用用户, 仅用户禁用时生效
	Enabled bool
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
	Project string `json:"project"`
	Role    string `json:"role"`
}

type SLeaveProjectsInput struct {
	ProjectRoles []SProjectRole `json:"project_roles"`
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
	Users          []string
	Groups         []string
	Roles          []string
	EnableAllUsers bool
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

	apis.DomainizedResourceListInput

	// 以驱动类型过滤
	Driver []string `json:"driver"`

	// 以模板过滤
	Template []string `json:"template"`

	// 以同步状态过滤
	SyncStatus []string `json:"sync_status"`

	// 过滤支持SSO的认证源，如果值为all，则列出所有的全局认证源，否则可出sso为域ID的域认证源
	// example: all
	SsoDomain string `json:"sso_domain"`

	AutoCreateProject *bool `json:"auto_create_project"`
	AutoCreateUser    *bool `json:"auto_create_user"`
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

	// 是否显示系统权限
	IsSystem *bool `json:"is_system"`

	// filter policies by role id
	RoleId string `json:"role_id"`
	// swagger: ignore
	// Deprecated
	Role string `json:"role" yunion-deprecated-by:"role_id"`
}

type RegionFilterListInput struct {
	// 以区域名称或ID过滤
	RegionId string `json:"region_id"`
	// swagger:ignore
	// Deprecated
	Region string `json:"region" yunion-deprecated-by:"region_id"`
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

	// TargetDomainId string `json:"target_domain_id"`

	// 当认证后用户加入项目不存在时是否自动创建项目
	AutoCreateProject *bool `json:"auto_create_project"`
	// 当认证后用户不存在时是否自动创建用户
	AutoCreateUser *bool `json:"auto_create_user"`

	SyncIntervalSeconds *int `json:"sync_interval_seconds"`

	// 图标URL
	IconUri string `json:"icon_uri"`
}

type PolicyTagInput struct {
	// 匹配的资源标签
	ObjectTags tagutils.TTagSet `json:"object_tags,allowempty"`
	// 匹配的项目标签
	ProjectTags tagutils.TTagSet `json:"project_tags,allowempty"`
	// 匹配的域标签
	DomainTags tagutils.TTagSet `json:"domain_tags,allowempty"`

	// 组织架构节点ID
	OrgNodeId []string `json:"org_node_id,allowempty"`
}

type PolicyUpdateInput struct {
	EnabledIdentityBaseUpdateInput

	// Deprecated
	// swagger:ignore
	Type string `json:"type"`

	// Policy内容
	Blob jsonutils.JSONObject `json:"blob"`

	// 生效范围，project|domain|system
	Scope rbacscope.TRbacScope `json:"scope"`

	// 是否为系统权限
	IsSystem *bool `json:"is_system"`

	PolicyTagInput

	// Policy tag更新策略，可能的值为：add|remove|remove，默认为add
	TagUpdatePolicy string `json:"tag_update_policy"`
}

const (
	TAG_UPDATE_POLICY_ADD     = "add"
	TAG_UPDATE_POLICY_REMOVE  = "remove"
	TAG_UPDATE_POLICY_REPLACE = "replace"
)

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

	SkipPasswordComplexityCheck *bool `json:"skip_password_complexity_check"`

	Lang string `json:"lang"`
}

type UserCreateInput struct {
	EnabledIdentityBaseResourceCreateInput

	Email string `json:"email"`

	Mobile string `json:"mobile"`

	Displayname string `json:"displayname"`

	IsSystemAccount *bool `json:"is_system_account"`

	AllowWebConsole *bool `json:"allow_web_console"`

	EnableMfa *bool `json:"enable_mfa"`

	Password string `json:"password"`

	SkipPasswordComplexityCheck *bool `json:"skip_password_complexity_check"`

	IdpId string `json:"idp_id"`

	IdpEntityId string `json:"idp_entity_id"`

	Lang string `json:"lang"`
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

	// Deprecated
	// swagger:ignore
	Type string `json:"type"`

	// policy
	Blob jsonutils.JSONObject `json:"blob"`

	// 生效范围，project|domain|system
	Scope rbacscope.TRbacScope `json:"scope"`

	// 是否为系统权限
	IsSystem *bool `json:"is_system"`

	PolicyTagInput
}

type RoleCreateInput struct {
	IdentityBaseResourceCreateInput

	apis.SharableResourceBaseCreateInput
}

type PerformGroupAddUsersInput struct {
	// 待添加用户列表（ID或名称）
	UserIds []string `json:"user_ids"`
	// Deprecated
	// swagger:ignore
	User []string `json:"user" yunion-deprecated-by:"user_ids"`
}

type PerformGroupRemoveUsersInput struct {
	// 待删除用户列表（ID或名称）
	UserIds []string `json:"user_ids"`
	// Deprecated
	// swagger:ignore
	User []string `json:"user" yunion-deprecated-by:"user_ids"`
}

type UserLinkIdpInput struct {
	IdpId       string `json:"idp_id"`
	IdpEntityId string `json:"idp_entity_id"`
}

type UserUnlinkIdpInput UserLinkIdpInput

type SProjectSetAdminInput struct {
	UserId string
}
