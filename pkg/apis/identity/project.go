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
	"time"
)

type ExternalResourceInfo struct {
	// 外部资源统计信息（资源类别：数量）
	ExtResource map[string]int `json:"ext_resource"`
	// 外部资源统计信息上次更新时间
	ExtResourcesLastUpdate time.Time `json:"ext_resources_last_update"`
	// 外部资源统计信息下次更新时间
	ExtResourcesNextUpdate time.Time `json:"ext_resources_next_update"`
}

type ProjectDetails struct {
	IdentityBaseResourceDetails

	SProject

	// 项目管理员名称
	Admin string `json:"admin"`
	// 项目管理员域ID
	AdminDomainId string `json:"admin_domain_id"`
	// 项目管理员域名称
	AdminDomain string `json:"admin_domain"`

	// 加入项目的用户组数量
	GroupCount int `json:"group_count"`
	// 加入项目的用户数量
	UserCount int `json:"user_count"`

	// 归属该项目的外部资源统计信息
	ExternalResourceInfo

	Organization *SProjectOrganization
}

type ProjectCleanInput struct {
}
