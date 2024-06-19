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

package yunionconf

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/apis"
)

type ParameterListInput struct {
	apis.ResourceBaseListInput

	NamespaceId string `json:"namespace_id"`

	// 服务名称或ID
	ServiceId string `json:"service_id"`

	// Deprecated
	// swagger:ignore
	Service string `json:"service" yunion-deprecated-by:"service_id"`

	// 用户名称或ID
	UserId string `json:"user_id"`

	// Deprecated
	// swagger:ignore
	User string `json:"user" yunion-deprecated-by:"user_id"`

	// filter by name
	Name []string `json:"name"`
}

type ParameterCloneInput struct {
	DestNs   string `json:"dest_ns"`
	DestNsId string `json:"dest_ns_id"`
	DestName string `json:"dest_name"`
}

type ScopedPolicyCreateInput struct {
	apis.InfrasResourceBaseCreateInput

	// 策略类别
	Category string `json:"category"`

	// 策略定义内容
	Policies jsonutils.JSONObject `json:"policies"`
}

type ScopedPolicyUpdateInput struct {
	apis.InfrasResourceBaseUpdateInput

	// 策略定义内容
	Policies jsonutils.JSONObject `json:"policies"`
}

type ScopedPolicyListInput struct {
	apis.InfrasResourceBaseListInput

	// 策略类别
	Category []string `json:"category"`
}

type ScopedPolicyDetails struct {
	apis.InfrasResourceBaseDetails

	RefCount int `json:"ref_count"`

	SScopedPolicy
}

type ScopedPolicyBindingListInput struct {
	apis.ResourceBaseListInput

	Name []string `json:"name"`

	PolicyId  string `json:"policy_id"`
	DomainId  string `json:"domain_id"`
	ProjectId string `json:"project_id"`

	Category []string `json:"category"`

	Effective *bool `json:"effective"`

	Scope rbacscope.TRbacScope `json:"scope"`

	OrderByScopedpolicy string `json:"order_by_scopedpolicy"`
}

type ScopedPolicyBindingDetails struct {
	apis.ResourceBaseDetails

	Id string `json:"id"`

	PolicyName string `json:"policy_name"`

	Category string `json:"category"`

	Policies jsonutils.JSONObject `json:"policy"`

	ProjectDomain string `json:"project_domain"`

	Project string `json:"project"`

	SScopedPolicyBinding
}

type ScopedPolicyBindInput struct {
	// 绑定范围
	Scope rbacscope.TRbacScope `json:"scope"`
	// 绑定的目标ID（域或者项目ID)
	TargetIds []string `json:"target_ids"`
}
