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

import "yunion.io/x/onecloud/pkg/apis"

type DomainDetails struct {
	apis.StandaloneResourceDetails
	IdpResourceInfo

	SDomain

	UserCout    int `json:"user_count"`
	GroupCount  int `json:"group_count"`
	ProjectCout int `json:"project_count"`
	RoleCount   int `json:"role_count"`
	PolicyCount int `json:"policy_count"`
	IdpCount    int `json:"idp_count"`

	ExternalResourceInfo
}

type DomainUpdateInput struct {
	apis.StandaloneResourceBaseUpdateInput

	// 显示名
	Displayname string `json:"displayname"`

	// 是否启用
	Enabled *bool `json:"enabled"`
}

type DomainCreateInput struct {
	apis.StandaloneResourceCreateInput

	// 显示名
	Displayname string `json:"displayname"`

	// 是否启用
	Enabled *bool `json:"enabled"`
}
