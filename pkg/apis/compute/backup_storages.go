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

type BackupstorageResourceInput struct {
	// 备份存储（ID或Name）
	BackupstorageId string `json:"backupstorage_id"`
	// swagger:ignore
	// Deprecated
	// filter by backupstorage_id
	Backupstorage string `json:"backupstorage" yunion-deprecated-by:"backupstorage_id"`
}

type BackupstorageFilterListInputBase struct {
	BackupstorageResourceInput

	// 以备份存储名称排序
	// pattern:asc|desc
	OrderByBackupstorage string `json:"order_by_backupstorage"`
}

type BackupstorageResourceInfo struct {
	// 备份存储名称
	Backupstorage string `json:"backupstorage"`

	// 备份存储类型
	BackupstorageType TBackupStorageType `json:"backupstorage_type"`

	// 备份存储状态
	BackupstorageStatus string `json:"backupstorage_status"`
}

type BackupstorageFilterListInput struct {
	BackupstorageFilterListInputBase
}
