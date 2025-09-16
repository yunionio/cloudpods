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

import "yunion.io/x/onecloud/pkg/apis"

type CloudgroupJointResourceDetails struct {
	apis.JointResourceBaseDetails

	// 公有云用户组名称
	Cloudgroup string `json:"cloudgroup"`
}

type CloudgroupJointsListInput struct {
	apis.JointResourceBaseListInput

	// 根据公有云用户组过滤资源
	Cloudgroup string `json:"cloudgroup"`
	// swagger:ignore
	CloudgroupId string `json:"cloudgroup_id" yunion-deprecated-by:"cloudgroup"`
}

type CloudgroupJointBaseUpdateInput struct {
	apis.JointResourceBaseUpdateInput
}

type CloudgroupUserDetails struct {
	CloudgroupJointResourceDetails
	SCloudgroup

	ClouduserResourceDetails
}

type CloudgroupUserListInput struct {
	CloudgroupJointsListInput

	ClouduserResourceListInput
}

type CloudgroupPolicyDetails struct {
	CloudgroupJointResourceDetails

	CloudpolicyResourceDetails
}

type CloudgroupPolicyListInput struct {
	CloudgroupJointsListInput

	CloudpolicyResourceListInput
}

type CloudgroupListInput struct {
	apis.StatusInfrasResourceBaseListInput
	CloudaccountResourceListInput
	CloudproviderResourceListInput

	// 过滤子账号所在的权限组
	ClouduserId string `json:"clouduser_id"`

	// 根据权限过滤权限组
	CloudpolicyId string `json:"cloudpolicy_id"`
}

type SCloudIdBaseResource struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type CloudgroupDetails struct {
	apis.StatusInfrasResourceBaseDetails
	CloudaccountResourceDetails
	CloudproviderResourceDetails

	SCloudgroup

	// 公有云子用户数量
	ClouduserCount int `json:"clouduser_count"`
	// 权限数量
	CloudpolicyCount int `json:"cloudpolicy_count"`

	Cloudpolicies []SCloudIdBaseResource `json:"cloudpolicies"`
	Cloudusers    []SCloudIdBaseResource `json:"cloudusers"`
}

type CloudgroupCreateInput struct {
	apis.StatusInfrasResourceBaseCreateInput

	// 子账号Id
	ManagerId string `json:"manager_id"`
	// swagger: ignore
	CloudaccountId string `json:"cloudaccount_id"`

	// 权限Id列表, 权限provider必须和权限组provider一致
	CloudpolicyIds []string `json:"cloudpolicy_ids"`
}

type CloudgroupAddUserInput struct {

	// 用户Id
	ClouduserId string `json:"clouduser_id"`
}

type CloudgroupRemoveUserInput struct {

	// 用户Id
	ClouduserId string `json:"clouduser_id"`
}

type CloudgroupAttachPolicyInput struct {

	// 权限Id
	CloudpolicyId string `json:"cloudpolicy_id"`
}

type CloudgroupSetUsersInput struct {

	// 公有云子账号Ids
	ClouduserIds []string `json:"clouduser_ids"`
}

type CloudgroupSetPoliciesInput struct {

	// 权限Ids
	CloudpolicyIds []string `json:"cloudpolicy_ids"`
}

type CloudgroupDetachPolicyInput struct {

	// 权限Id
	CloudpolicyId string `json:"cloudpolicy_id"`
}

type CloudgroupSyncstatusInput struct {
}

type CloudgroupSyncInput struct {
}

type CloudgroupUpdateInput struct {
}

type CloudgroupResourceListInput struct {
	// 根据权限组Id过滤资源
	CloudgroupId string `json:"cloudgroup_id"`
}

type CloudgroupResourceDetails struct {
	// 公有云用户名称
	Cloudgroup string `json:"cloudgroup"`
}

type SPolicy struct {
	Name       string
	ExternalId string
	PolicyType string
}

type GroupUser struct {
	Name       string
	ExternalId string
}

type SGroup struct {
	Id   string
	Name string
}

type GetCloudaccountSamlOutput struct {
	// cloudaccount SAML ServiceProvider entity ID
	EntityId string `json:"entity_id,allowempty"`
	// redirect login URL for this cloudaccount
	RedirectLoginUrl string `json:"redirect_login_url,allowempty"`
	// redirect logout URL for this cloudaccount
	RedirectLogoutUrl string `json:"redirect_logout_url,allowempty"`
	// metadata URL for this cloudaccount
	MetadataUrl string `json:"metadata_url,allowempty"`
	// initial SAML SSO login URL for this cloudaccount
	InitLoginUrl string `json:"init_login_url,allowempty"`
}
