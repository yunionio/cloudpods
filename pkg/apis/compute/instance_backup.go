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
	"time"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	INSTANCE_BACKUP_STATUS_CREATING        = "creating"
	INSTANCE_BACKUP_STATUS_CREATE_FAILED   = "create_failed"
	INSTANCE_BACKUP_STATUS_DELETING        = "deleting"
	INSTANCE_BACKUP_STATUS_DELETE_FAILED   = "delete_failed"
	INSTANCE_BACKUP_STATUS_RECOVERY        = "recovery"
	INSTANCE_BACKUP_STATUS_RECOVERY_FAILED = "recovery_failed"
	INSTANCE_BACKUP_STATUS_READY           = "ready"
	INSTANCE_BACKUP_STATUS_PACK            = "pack"
	INSTANCE_BACKUP_STATUS_PACK_FAILED     = "pack_failed"

	INSTANCE_BACKUP_STATUS_CREATING_FROM_PACKAGE      = "creating_from_package"
	INSTANCE_BACKUP_STATUS_CREATE_FROM_PACKAGE_FAILED = "create_from_package_failed"

	INSTANCE_BACKUP_STATUS_SNAPSHOT        = "snapshot"
	INSTANCE_BACKUP_STATUS_SNAPSHOT_FAILED = "snapshot_failed"
	INSTANCE_BACKUP_STATUS_SAVING          = "saving"
	INSTANCE_BACKUP_STATUS_SAVE_FAILED     = "save_failed"
)

type InstanceBackupListInput struct {
	apis.VirtualResourceListInput
	apis.MultiArchResourceBaseListInput

	ManagedResourceListInput

	ServerFilterListInput

	// 按虚拟机名称排序
	// pattern:asc|desc
	OrderByGuest string `json:"order_by_guest"`

	// 操作系统类型
	OsType []string `json:"os_type"`
}

type InstanceBackupDetails struct {
	apis.VirtualResourceDetails
	ManagedResourceInfo
	apis.EncryptedResourceDetails

	// 云主机状态
	GuestStatus string `json:"guest_status"`
	// 云主机名称
	Guest string `json:"guest"`

	// 存储类型
	BackupStorageName string `json:"backup_storage_name"`

	// 主机快照大小
	Size int `json:"size"`

	// 硬盘备份详情
	DiskBackups []SSimpleBackup `json:"disk_backups"`
}

type SSimpleBackup struct {
	// 备份Id
	Id string `json:"id"`
	// 备份名称
	Name string `json:"name"`
	// 备份大小
	SizeMb int `json:"size_mb"`
	// 磁盘大小
	DiskSizeMb int `json:"disk_size_mb"`
	// 磁盘类型
	DiskType string `json:"disk_type"`
	// 备份存储Id
	BackupStorageId string `json:"backup_storage_id"`
	// 快照状态
	Status string `json:"status"`
	// 密钥
	EncryptKeyId string `json:"encrypt_key_id"`
	// 创建时间
	CreatedAt time.Time `json:"created_at"`
}

type InstanceBackupRecoveryInput struct {
	// description: name of guest
	Name string
}

type InstanceBackupPackInput struct {
	PackageName string
}

type InstanceBackupManagerCreateFromPackageInput struct {
	apis.VirtualResourceCreateInput

	BackupStorageId string
	PackageName     string
}
