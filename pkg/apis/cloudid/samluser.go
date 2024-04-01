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

type SamluserCreateInput struct {
	apis.StatusDomainLevelUserResourceCreateInput

	// 权限组和账号必须是同一平台
	CloudgroupId string `json:"cloudgroup_id"`

	// 权限组和账号必须是同一平台
	CloudaccountId string `json:"cloudaccount_id"`

	Email string `json:"email"`
}

type SamluserListInput struct {
	apis.StatusDomainLevelUserResourceListInput
	CloudgroupResourceListInput
	CloudaccountResourceListInput
}

type SamluserDetails struct {
	apis.StatusDomainLevelUserResourceDetails
	CloudgroupResourceDetails

	CloudaccountId string `json:"cloudaccount_id"`
	Cloudaccount   string `json:"cloudaccount"`

	ManagerId string
	Manager   string

	Provider string
	Brand    string

	SSamluser
}
