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

package cloudid

import (
	"yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	CLOUD_POLICY_TYPE_SYSTEM = string(cloudid.PolicyTypeSystem)
	CLOUD_POLICY_TYPE_CUSTOM = string(cloudid.PolicyTypeCustom)
)

type CloudpolicyListInput struct {
	apis.StatusInfrasResourceBaseListInput

	CloudaccountResourceListInput
	CloudproviderResourceListInput

	// 根据平台过滤
	Provider []string `json:"provider"`

	// 根据子账号过滤权限
	ClouduserId string `json:"clouduser_id"`

	// 根据权限组过滤权限
	CloudgroupId string `json:"cloudgroup_id"`

	// 根据订阅过滤权限
	ManagerId string `json:"manager_id"`

	// 权限类型
	//
	//
	// | 类型    |  说明                |
	// |---------| ------------            |
	// | system  |  过滤系统权限        |
	// | custom  |  过滤自定义权限      |
	PolicyType string `json:"policy_type"`
}

type PolicyUsage struct {
	CloudgroupCount int
	ClouduserCount  int
}

type CloudpolicyDetails struct {
	apis.StatusInfrasResourceBaseDetails

	CloudaccountResourceDetails
	CloudproviderResourceDetails

	SCloudpolicy

	PolicyUsage
}

type CloudpolicyCreateInput struct {
	apis.StatusInfrasResourceBaseCreateInput

	// default: custom
	PolicyType string `json:"policy_type"`

	// 策略详情
	Document *jsonutils.JSONDict `json:"document"`
}

type CloudpolicyResourceListInput struct {
	// 根据公有云权限过滤资源
	Cloudpolicy string `json:"cloudpolicy"`

	// swagger:ignore
	CloudpolicyId string `json:"cloudpolicy_id" yunion-deprecated-by:"cloudpolicy"`
}

type CloudpolicyResourceDetails struct {
	// 公有云权限名称
	Cloudpolicy string `json:"cloudpolicy"`
}

type CloudpolicyUpdateInput struct {
	apis.StatusInfrasResourceBaseUpdateInput

	Document *jsonutils.JSONDict `json:"document"`

	// swagger:ignore
	OriginDocument *jsonutils.JSONDict `json:"origin_document"`
}

type CloudpolicyAssignGroupInput struct {

	// 权限组Id
	CloudgroupId string `json:"cloudgroup_id"`
}

type CloudpolicyRevokeGroupInput struct {

	// 权限组Id
	CloudgroupId string `json:"cloudgroup_id"`
}
