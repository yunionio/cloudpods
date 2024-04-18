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
	"yunion.io/x/onecloud/pkg/apis"
)

type HostBackupstorageDetails struct {
	HostJointResourceDetails

	SHostBackupstorage

	// 存储名称
	Backupstorage string `json:"backupstorage"`
	// 存储大小
	CapacityMb int64 `json:"capacity_mb"`
	// 存储类型
	// example: local
	StorageType TBackupStorageType `json:"storage_type"`
	// 是否启用
	Enabled bool `json:"enabled"`
}

type HostBackupstorageListInput struct {
	HostJointsListInput

	BackupstorageFilterListInput
}

type HostBackupstorageCreateInput struct {
	apis.JoinResourceBaseCreateInput
	BackupstorageId string `json:"backupstorage_id"`
	HostId          string `json:"host_id"`
}
