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

import "yunion.io/x/onecloud/pkg/apis"

const (
	ACCESS_GROUP_STATUS_AVAILABLE         = "available"
	ACCESS_GROUP_STATUS_DELETING          = "deleting"
	ACCESS_GROUP_STATUS_DELETE_FAILED     = "delete_failed"
	ACCESS_GROUP_STATUS_CREATING          = "creating"
	ACCESS_GROUP_STATUS_SYNC_RULES        = "sync_rules"
	ACCESS_GROUP_STATUS_SYNC_RULES_FAILED = "sync_rules_failed"
	ACCESS_GROUP_STATUS_UNKNOWN           = "unknown"
)

type AccessGroupListInput struct {
	apis.StatusInfrasResourceBaseListInput
	apis.ExternalizedResourceBaseListInput
	ManagedResourceListInput
	RegionalFilterListInput
}

type AccessGroupDetails struct {
	apis.StatusInfrasResourceBaseDetails
	ManagedResourceInfo
	CloudregionResourceInfo
}

type AccessGroupResourceInfo struct {
	// 权限组名称
	AccessGroup string `json:"access_group"`
}

type AccessGroupFilterListInput struct {
	// 权限组Id
	AccessGroupId string `json:"access_group_id"`

	// 以权限组排序
	OrderByAccessGroup string `json:"order_by_access_group"`
}

type AccessGroupCreateInput struct {
	apis.StatusInfrasResourceBaseCreateInput

	CloudproviderResourceInput
	CloudregionResourceInput
}
