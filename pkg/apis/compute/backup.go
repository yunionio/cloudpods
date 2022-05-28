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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	BACKUPSTORAGE_TYPE_NFS       = "nfs"
	BACKUPSTORAGE_STATUS_ONLINE  = "online"
	BACKUPSTORAGE_STATUS_OFFLINE = "offline"

	BACKUP_STATUS_CREATING                = "creating"
	BACKUP_STATUS_CREATE_FAILED           = "create_failed"
	BACKUP_STATUS_SNAPSHOT                = "snapshot"
	BACKUP_STATUS_SNAPSHOT_FAILED         = "snapshot_failed"
	BACKUP_STATUS_SAVING                  = "saving"
	BACKUP_STATUS_SAVE_FAILED             = "save_failed"
	BACKUP_STATUS_CLEANUP_SNAPSHOT        = "clean_snapshot"
	BACKUP_STATUS_CLEANUP_SNAPSHOT_FAILED = "clean_snapshot_failed"
	BACKUP_STATUS_DELETING                = "deleting"
	BACKUP_STATUS_DELETE_FAILED           = "delete_failed"
	BACKUP_STATUS_READY                   = "ready"
	BACKUP_STATUS_RECOVERY                = "recovery"
	BACKUP_STATUS_RECOVERY_FAILED         = "recovery_failed"
	BACKUP_STATUS_UNKNOWN                 = "unknown"

	BACKUP_EXIST     = "exist"
	BACKUP_NOT_EXIST = "not_exist"
)

const (
	BackupStorageOffline = "backup storage offline"
)

type BackupStorageCreateInput struct {
	apis.EnabledStatusInfrasResourceBaseCreateInput

	// description: storage type
	// enum: nfs
	StorageType string `json:"storage_type"`

	// description: host of nfs, storage_type 为 nfs 时, 此参数必传
	// example: 192.168.222.2
	NfsHost string `json:"nfs_host"`

	// description: shared dir of nfs, storage_type 为 nfs 时, 此参数必传
	// example: /nfs_root/
	NfsSharedDir string `json:"nfs_shared_dir"`

	// description: Capacity size in MB
	CapacityMb int `json:"capacity_mb"`
}

type BackupStorageAccessInfo struct {
	AccessUrl string
}

type BackupStorageDetails struct {
	apis.EnabledStatusInfrasResourceBaseDetails

	NfsHost      string
	NfsSharedDir string
}

type BackupStorageListInput struct {
	apis.EnabledStatusInfrasResourceBaseListInput
}

type DiskBackupListInput struct {
	apis.VirtualResourceListInput
	ManagedResourceListInput
	RegionalFilterListInput
	apis.MultiArchResourceBaseListInput
	// description: disk id
	DiskId string `json:"disk_id"`
	// description: backup storage id
	BackupStorageId string `json:"backup_storage_id"`
	// description: 是否为主机备份的一部分
	IsInstanceBackup *bool `json:"is_instance_backup"`
}

type DiskBackupDetails struct {
	apis.VirtualResourceDetails
	ManagedResourceInfo
	CloudregionResourceInfo
	apis.EncryptedResourceDetails

	// description: disk name
	DiskName string `json:"disk_name"`
	// description: backup storage name
	BackupStorageName string `json:"backup_storage_name"`
	// description: 是否是子备份
	IsSubBackup bool `json:"is_sub_backup"`
}

type DiskBackupCreateInput struct {
	apis.VirtualResourceCreateInput
	apis.EncryptedResourceCreateInput

	// description: disk id
	DiskId string `json:"disk_id"`
	// description: backup storage id
	BackupStorageId string `json:"back_storage_id"`
	// swagger: ignore
	CloudregionId string `json:"cloudregion_id"`
	// swagger:ignore
	ManagerId string `json:"manager_id"`
}

type DiskBackupRecoveryInput struct {
	// description: name of disk
	Name string
}

type DiskBackupSyncstatusInput struct {
}

type DiskBackupPackMetadata struct {
	OsArch     string
	SizeMb     int
	DiskSizeMb int
	DiskType   string
	// 操作系统类型
	OsType     string
	DiskConfig *SBackupDiskConfig
}

type InstanceBackupPackMetadata struct {
	OsArch         string
	ServerConfig   jsonutils.JSONObject
	ServerMetadata jsonutils.JSONObject
	SecGroups      jsonutils.JSONObject
	KeypairId      string
	OsType         string
	InstanceType   string
	SizeMb         int
	DiskMetadatas  []DiskBackupPackMetadata

	// 加密密钥ID
	EncryptKeyId string
	// Instance Backup metadata
	Metadata map[string]string `json:"metadata"`
}

type InstanceBackupManagerSyncstatusInput struct {
}
