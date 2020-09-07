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

const (
	CLOUD_GROUP_STATUS_AVAILABLE            = "available"            // 可用
	CLOUD_GROUP_STATUS_DELETING             = "deleting"             // 删除中
	CLOUD_GROUP_STATUS_DELETE_FAILED        = "delete_failed"        // 删除失败
	CLOUD_GROUP_STATUS_SYNC_POLICIES        = "sync_policies"        // 同步权限中
	CLOUD_GROUP_STATUS_SYNC_POLICIES_FAILED = "sync_policies_failed" // 同步权限失败
	CLOUD_GROUP_STATUS_SYNC_USERS           = "sync_users"           // 同步用户中
	CLOUD_GROUP_STATUS_SYNC_USERS_FAILED    = "sync_users_failed"    // 同步用户失败
	CLOUD_GROUP_STATUS_SYNC_STATUS          = "sync_status"          // 同步状态
)

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

	// 根据平台过滤
	Provider []string `json:"provider"`

	// 过滤子账号所在的权限组
	ClouduserId string `json:"clouduser_id"`

	// 根据权限过滤权限组
	CloudpolicyId string `json:"cloudpolicy_id"`

	// 是否可用
	Usable *bool `json:"usable"`
}

type SCloudIdBaseResource struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type CloudgroupDetails struct {
	apis.StatusInfrasResourceBaseDetails
	SCloudgroup

	// 公有云子用户数量
	ClouduserCount int `json:"clouduser_count"`
	// 权限数量
	CloudpolicyCount int `json:"cloudpolicy_count"`
	// 公有云权限组缓存数量
	CloudgroupcacheCount int `json:"cloudgroupcache_count"`

	Cloudpolicies []SCloudIdBaseResource `json:"cloudpolicies"`
	Cloudusers    []SCloudIdBaseResource `json:"cloudusers"`
}

type CloudgroupCreateInput struct {
	apis.StatusInfrasResourceBaseCreateInput

	// 平台, 目前支持Qcloud,Google,Azure,Aliyun,Huawei,Aws
	Provider string `json:"provider"`

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
