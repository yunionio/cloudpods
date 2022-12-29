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

package compute

import (
	"yunion.io/x/cloudmux/pkg/apis/compute"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	EXTERNAL_PROJECT_STATUS_AVAILABLE   = compute.EXTERNAL_PROJECT_STATUS_AVAILABLE   // 可用
	EXTERNAL_PROJECT_STATUS_UNAVAILABLE = compute.EXTERNAL_PROJECT_STATUS_UNAVAILABLE // 不可用
	EXTERNAL_PROJECT_STATUS_CREATING    = compute.EXTERNAL_PROJECT_STATUS_CREATING    // 创建中
	EXTERNAL_PROJECT_STATUS_DELETING    = compute.EXTERNAL_PROJECT_STATUS_DELETING    // 删除中
	EXTERNAL_PROJECT_STATUS_UNKNOWN     = compute.EXTERNAL_PROJECT_STATUS_UNKNOWN     // 未知
)

var (
	MANGER_EXTERNAL_PROJECT_PROVIDERS = []string{
		CLOUD_PROVIDER_AZURE,
	}
)

type ExternalProjectDetails struct {
	apis.VirtualResourceDetails
	ManagedResourceInfo

	SExternalProject
}

type ExternalProjectChangeProjectInput struct {
	apis.DomainizedResourceInput
	apis.ProjectizedResourceInput
}

type ExternalProjectCreateInput struct {
	apis.VirtualResourceCreateInput

	CloudaccountId string `json:"cloudaccount_id"`
	ManagerId      string `json:"manager_id"`

	// swagger:ignore
	ExternalId string `json:"external_id"`
}
