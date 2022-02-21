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

	"yunion.io/x/onecloud/pkg/apis"
)

type PolicyDetails struct {
	EnabledIdentityBaseResourceDetails
	apis.SharableResourceBaseInfo

	SPolicy
}

type PolicyBindRoleInput struct {
	// 角色ID
	RoleId string `json:"role_id"`
	// 项目ID
	ProjectId string `json:"project_id"`
	//	IP白名单
	Ips []string `json:"ips"`
	// 权限有效开始时间
	ValidSince time.Time `json:"valid_since"`
	// 权限有效结束时间
	ValidUntil time.Time `json:"valid_until"`
}
