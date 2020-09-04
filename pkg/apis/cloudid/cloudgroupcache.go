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
	CLOUD_GROUP_CACHE_STATUS_AVAILABLE     = "available"     // 正常
	CLOUD_GROUP_CACHE_STATUS_CREATING      = "creating"      // 创建中
	CLOUD_GROUP_CACHE_STATUS_CREATE_FAILED = "create_failed" // 创建失败
	CLOUD_GROUP_CACHE_STATUS_DELETING      = "deleting"      // 删除中
	CLOUD_GROUP_CACHE_STATUS_DELETE_FAILED = "delete_failed" // 删除失败
	CLOUD_GROUP_CACHE_STATUS_SYNC_STATUS   = "sync_status"   // 同步状态中
	CLOUD_GROUP_CACHE_STATUS_UNKNOWN       = "unknown"       // 未知
)

type CloudgroupcacheListInput struct {
	apis.StatusStandaloneResourceListInput
	CloudaccountResourceListInput

	// 根据权限组过滤
	CloudgroupId string `json:"cloudgroup_id"`
}

type CloudgroupcacheCreateInput struct {
}

type CloudgroupcacheSyncstatusInput struct {
}

type CloudgroupcacheDetails struct {
	apis.StatusStandaloneResourceDetails
	SCloudgroupcache

	CloudaccountResourceDetails
}
