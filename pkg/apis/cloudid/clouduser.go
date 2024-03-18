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
	CLOUD_USER_STATUS_RESET_PASSWORD        = "reset_password"        // 重置密码中
	CLOUD_USER_STATUS_RESET_PASSWORD_FAILED = "reset_password_failed" // 重置密码失败
)

type ClouduserCreateInput struct {
	apis.StatusDomainLevelUserResourceCreateInput
	apis.StatusBaseResourceCreateInput

	// 云订阅ID
	ManagerId string `json:"manager_id"`
	// 云账号ID
	// Azure云账号需要有User administrator权限，否则删操作会出现Insufficient privileges to complete the operation错误信息
	CloudaccountId string `json:"cloudaccount_id"`

	// 用户密码, 若is_console_login = true时, 此参数不传时会生成12位随机密码
	//
	//
	//
	// | 云平台   | 说明                                        |
	// |----------|---------------------------------------------|
	// | Google   | 不支持此参数                                |
	// | Aliyun   | 支持                                        |
	// | Huawei   | 支持                                        |
	// | Azure    | 支持                                        |
	// | 腾讯云   | 支持                                        |
	Password string `json:"password"`
	// 是否可控制台登录
	// default: false
	IsConsoleLogin *bool `json:"is_console_login"`

	// 邮箱地址
	// example: test@example.com
	Email string `json:"email"`
	// 手机号码
	// example: 86-1868888****
	MobilePhone string `json:"mobile_phone"`

	// 初始的权限Id列表, 权限必须属于指定的云账号
	//
	//
	//
	// | 云平台   | 说明                                        |
	// |----------|---------------------------------------------|
	// | Google   | 至少需要一个初始权限                        |
	// | Aliyun   | 支持                                        |
	// | Huawei   | 不支持                                      |
	// | Azure    | 支持                                        |
	// | 腾讯云   | 支持                                        |
	CloudpolicyIds []string `json:"cloudpolicy_ids"`

	// 初始化权限组Id列表, 权限组必须和云账号平台属性相同
	CloudgroupIds []string `json:"cloudgroup_ids"`

	// swagger:ignore
	ExternalId string `json:"external_id"`

	// 是否发送邮件通知(仅设置email不为空生效)
	// default: false
	Notify bool `json:"notify"`
}

type ClouduserListInput struct {
	apis.StatusDomainLevelUserResourceListInput

	CloudaccountResourceListInput
	CloudproviderResourceListInput

	// 过滤绑定权限的子账号
	CloudpolicyId string `json:"cloudpolicy_id"`

	// 过滤属于指定权限组的子账号
	CloudgroupId string `json:"cloudgroup_id"`
}

type ClouduserDetails struct {
	apis.StatusDomainLevelUserResourceDetails
	SClouduser

	CloudaccountResourceDetails
	CloudproviderResourceDetails

	// 权限数量
	CloudpolicyCount int `json:"cloudpolicy_count"`

	// 权限组数量
	CloudgroupCount int `json:"cloudgroup_count"`

	Cloudgroups   []SCloudIdBaseResource `json:"cloudgroups"`
	Cloudpolicies []SCloudIdBaseResource `json:"cloudpolicies"`
}

type ClouduserJointResourceDetails struct {
	apis.JointResourceBaseDetails

	ClouduserResourceDetails
}

type ClouduserJointsListInput struct {
	apis.JointResourceBaseListInput

	ClouduserResourceListInput
}

type ClouduserJointBaseUpdateInput struct {
	apis.JointResourceBaseUpdateInput
}

type ClouduserPolicyDetails struct {
	ClouduserJointResourceDetails

	CloudpolicyResourceDetails
}

type ClouduserPolicyListInput struct {
	ClouduserJointsListInput

	CloudpolicyResourceListInput
}

type ClouduserResourceListInput struct {
	// 根据公有云用户过滤资源
	Clouduser string `json:"clouduser"`

	// swagger:ignore
	ClouduserId string `json:"clouduser_id" yunion-deprecated-by:"clouduser"`
}

type ClouduserResourceDetails struct {
	// 公有云用户名称
	Clouduser string `json:"clouduser"`

	// 云账号名称
	Cloudaccount string `json:"cloudaccount"`

	// 云订阅名称
	Cloudprovider string `json:"cloudprovider"`
}

type ClouduserAttachPolicyInput struct {
	// 权限Id
	//
	//
	//
	// | 云平台   | 说明                                        |
	// |----------|---------------------------------------------|
	// | Google   | 支持                                        |
	// | Aliyun   | 支持                                        |
	// | Huawei   | 不支持                                      |
	// | Azure    | 支持                                        |
	// | 腾讯云   | 支持                                        |
	CloudpolicyId string `json:"cloudpolicy_id"`
}

type ClouduserSetPoliciesInput struct {
	// 权限Ids
	CloudpolicyIds []string `json:"cloudpolicy_ids"`
}

type ClouduserSetGroupsInput struct {
	// 权限组Ids
	CloudgroupIds []string `json:"cloudgroup_ids"`
}

type ClouduserJoinGroupInput struct {

	// 权限组Id
	CloudgroupId string `json:"cloudgroup_id"`
}

type ClouduserLeaveGroupInput struct {
	// 权限组Id
	CloudgroupId string `json:"cloudgroup_id"`
}

type ClouduserDetachPolicyInput struct {
	// 权限Id
	//
	//
	//
	// | 云平台   | 说明                                        |
	// |----------|---------------------------------------------|
	// | Google   | 支持，但最少需要保留一个权限                |
	// | Aliyun   | 支持                                        |
	// | Huawei   | 不支持                                      |
	// | Azure    | 不支持                                      |
	// | 腾讯云   | 支持                                        |
	CloudpolicyId string `json:"cloudpolicy_id"`
}

type ClouduserSyncstatusInput struct {
}

type ClouduserSyncInput struct {
}

type ClouduserUpdateInput struct {
}

type ClouduserResetPasswordInput struct {
	// 若此参数为空, 默认会生成随机12位密码
	//
	//
	//
	// | 云平台   | 说明                                        |
	// |----------|---------------------------------------------|
	// | Google   | 不支持                                      |
	// | Aliyun   | 支持                                        |
	// | Huawei   | 支持                                        |
	// | Azure    | 支持                                        |
	// | 腾讯云   | 支持                                        |
	Password string `json:"password"`
}

type ClouduserChangeOwnerInput struct {

	// 本地用户Id
	UserId string `json:"user_id"`
}

type ClouduserCreateAccessKeyInput struct {
	Name string `json:"name"`
}

type ClouduserDeleteAccessKeyInput struct {
	AccessKey string `json:"access_key"`
}

type ClouduserListAccessKeyInput struct {
}
