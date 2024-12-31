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

type TBackupStorageType string

const (
	BACKUPSTORAGE_TYPE_NFS            = TBackupStorageType("nfs")
	BACKUPSTORAGE_TYPE_OBJECT_STORAGE = TBackupStorageType("object")

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
	// enum: ["nfs"]
	StorageType string `json:"storage_type"`

	SBackupStorageAccessInfo

	// description: Capacity size in MB
	CapacityMb int `json:"capacity_mb"`
}

type BackupStorageUpdateInput struct {
	apis.EnabledStatusInfrasResourceBaseUpdateInput

	SBackupStorageAccessInfo
}

/*type BackupStorageAccessInfo struct {
	AccessUrl string
}*/

type BackupStorageDetails struct {
	apis.EnabledStatusInfrasResourceBaseDetails

	SBackupStorageAccessInfo
}

type BackupStorageListInput struct {
	apis.EnabledStatusInfrasResourceBaseListInput

	// filter by server_id
	ServerId string `json:"server_id"`
	// filter by disk_id
	DiskId string `json:"disk_id"`
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
	// 按硬盘名称排序
	OrderByDiskName string `json:"order_by_disk_name"`
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

	SDiskBackup
}

type DiskBackupAsTarInput struct {
	IncludeFiles       []string `json:"include_files"`
	ExcludeFiles       []string `json:"exclude_files"`
	ContainerId        string   `json:"container_id"`
	IgnoreNotExistFile bool     `json:"ignore_not_exist_file"`
}

type DiskBackupCreateInput struct {
	apis.VirtualResourceCreateInput
	apis.EncryptedResourceCreateInput

	// description: disk id
	DiskId string `json:"disk_id"`
	// swagger: ignore
	BackStorageId string `json:"back_storage_id" yunion-deprecated-by:"backup_storage_id"`
	// description: backup storage id
	BackupStorageId string `json:"backup_storage_id"`
	// swagger: ignore
	CloudregionId string `json:"cloudregion_id"`
	// swagger:ignore
	ManagerId   string                `json:"manager_id"`
	BackupAsTar *DiskBackupAsTarInput `json:"backup_as_tar"`
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

type SBackupStorageAccessInfo struct {
	// description: host of nfs, storage_type 为 nfs 时, 此参数必传
	// example: 192.168.222.2
	NfsHost string `json:"nfs_host"`

	// description: shared dir of nfs, storage_type 为 nfs 时, 此参数必传
	// example: /nfs_root/
	NfsSharedDir string `json:"nfs_shared_dir"`

	// description: access url of object storage bucket
	// example: https://qxxxxxo.tos-cn-beijing.volces.com
	ObjectBucketUrl string `json:"object_bucket_url"`
	// description: access key of object storage
	ObjectAccessKey string `json:"object_access_key"`
	// description: secret of object storage
	ObjectSecret string `json:"object_secret"`
	// description: signing version, can be v2/v4, default is v4
	ObjectSignVer string `json:"object_sign_ver"`
}

func (ba *SBackupStorageAccessInfo) String() string {
	return jsonutils.Marshal(ba).String()
}

func (ba *SBackupStorageAccessInfo) IsZero() bool {
	return ba == nil
}

type ServerCreateInstanceBackupInput struct {
	// 主机备份名称
	Name string `json:"name"`
	// 主机备份的生成名称
	GenerateName string `json:"generate_name"`
	// 备份存储ID
	BackupStorageId string `json:"backup_storage_id"`
}
