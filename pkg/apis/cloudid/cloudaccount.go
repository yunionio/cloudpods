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

type CloudaccountResourceListInput struct {
	// 根据云账号名称过滤资源
	Cloudaccount string `json:"cloudaccount" yunion-deprecated-by:"cloudaccount_id"`

	// 根据平台过滤
	Provider []string `json:"provider"`

	// swagger:ignore
	CloudaccountId string `json:"cloudaccount_id"`
}

type CloudproviderResourceListInput struct {
	// 根据云订阅过滤资源
	ManagerId string `json:"manager_id"`

	// swagger:ignore
	CloudproviderId string `json:"cloudprovider_id" yunion-deprecated-by:"manager_id"`
}

type CloudaccountResourceDetails struct {
	// 云账号名称
	Cloudaccount string `json:"cloudaccount"`
	// 平台信息
	Provider string `json:"provider"`
	// 品牌信息
	Brand string `json:"brand"`

	// 公有云账号登录地址
	IamLoginUrl string `json:"iam_login_url"`
}

type CloudproviderResourceDetails struct {
	// 云订阅名称
	Manager string `json:"manager"`
}
