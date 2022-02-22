package compute

import "yunion.io/x/onecloud/pkg/apis"

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
)

type InstanceBackupListInput struct {
	apis.VirtualResourceListInput
	apis.MultiArchResourceBaseListInput

	ManagedResourceListInput

	ServerFilterListInput

	// 操作系统类型
	OsType []string `json:"os_type"`
}

type InstanceBackupDetails struct {
	apis.VirtualResourceDetails
	ManagedResourceInfo

	// 云主机状态
	GuestStatus string `json:"guest_status"`
	// 云主机名称
	Guest string `json:"guest"`

	// 存储类型
	BackupStorageName string `json:"backup_storage_name"`

	// 主机快照大小
	Size int `json:"size"`
}

type InstanceBackupRecoveryInput struct {
	// description: name of guest
	Name string
}

type InstanceBackupPackInput struct {
	PackageName string
}

type InstanceBackupManagerCreateFromPackageInput struct {
	BackupStorageId string
	PackageName     string
	Name            string
}
