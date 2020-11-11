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
	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type RolePolicyListInput struct {
	apis.ResourceBaseListInput

	RoleIds []string `json:"role_ids"`

	ProjectId string `json:"project_id"`

	PolicyId string `json:"policy_id"`

	Auth *bool `json:"auth"`
}

type RolePolicyDetails struct {
	apis.ResourceBaseDetails

	Id string `json:"id"`

	Name string `json:"name"`

	Role string `json:"role"`

	Project string `json:"project"`

	Policy string `json:"policy"`

	Scope rbacutils.TRbacScope `json:"scope"`

	Description string `json:"description"`

	SRolePolicy
}

const (
	ROLE_SET_POLICY_ACTION_REPLACE = "replace"
	ROLE_SET_POLICY_ACTION_UPDATE  = "update"
	ROLE_SET_POLICY_ACTION_DEFAULT = ROLE_SET_POLICY_ACTION_REPLACE
)

type RolePerformSetPoliciesInput struct {
	// 操作：replace|update, 默认为replace
	Action string `json:"action"`
	// 权限列表
	Policies []RolePerformAddPolicyInput `json:"policies"`
}

type RolePerformAddPolicyInput struct {
	PolicyId  string   `json:"policy_id"`
	ProjectId string   `json:"project_id"`
	Ips       []string `json:"ips"`
}

type RolePerformRemovePolicyInput struct {
	PolicyId  string `json:"policy_id"`
	ProjectId string `json:"project_id"`
}
