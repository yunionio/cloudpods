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

import "yunion.io/x/onecloud/pkg/apis"

type IdentityProviderDetails struct {
	apis.EnabledStatusStandaloneResourceDetails

	// 认证源账号信息同步周期
	SyncIntervalSeconds int `json:"sync_interval_seconds"`

	// 认证源的目标域名称
	TargetDomain string `json:"target_domain"`

	// 该认证源关联的所有域的角色数量
	RoleCount int `json:"role_count,allowempty"`

	// 该认证源关联的所有域的用户数量
	UserCount int `json:"user_count,allowempty"`

	// 该认证源关联的所有域的权限策略数量
	PolicyCount int `json:"policy_count,allowempty"`

	// 该认证源关联的所有域的数量
	DomainCount int `json:"domain_count,allowempty"`

	// 该认证源关联的所有域的项目数量
	ProjectCount int `json:"project_count,allowempty"`

	// 该认证源关联的所有域的组数量
	GroupCount int `json:"group_count,allowempty"`

	SIdentityProvider
}

type IdpResourceInfo struct {
	// 认证源ID
	IdpId string `json:"idp_id"`

	// 认证源名称
	Idp string `json:"idp"`

	// 该资源在认证源的原始ID
	IdpEntityId string `json:"idp_entity_id"`

	// 认证源类型, 例如sql, cas, ldap等
	IdpDriver string `json:"idp_driver"`
}

type IdentityProviderCreateInput struct {
	apis.EnabledStatusStandaloneResourceCreateInput

	// 后端驱动名称
	Driver string `json:"driver"`

	// 模板名称
	Template string `json:"template"`

	// 默认导入用户和组的域
	TargetDomain string `json:"target_domain"`
	// swagger:ignore
	// Deprecated
	TargetDomainId string `json:"target_domain_id" "yunion:deprecated-by":"target_domain"`

	// 新建域的时候是否自动新建第一个项目
	AutoCreateProject *bool `json:"auto_create_project"`

	// 自动同步间隔，单位：秒
	SyncIntervalSeconds *int `json:"sync_interval_seconds"`

	// 配置信息
	Config TConfigs `json:"config"`
}
