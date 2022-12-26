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
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/apis"
)

type RoleDetails struct {
	IdentityBaseResourceDetails
	apis.SharableResourceBaseInfo

	SRole

	// 具有该角色的用户数量
	UserCount int `json:"user_count"`
	// 具有该角色的用户组数量
	GroupCount int `json:"group_count"`
	// 有该角色的用户或组的项目的数量
	ProjectCount int `json:"project_count"`

	// 该角色匹配的权限的名称列表
	MatchPolicies []string `json:"match_policies"`

	// 不同级别的权限的名称列表
	Policies map[rbacscope.TRbacScope][]string `json:"policies"`
}
