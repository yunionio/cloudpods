package models

import "yunion.io/x/onecloud/pkg/cloudcommon/db"

var llmBackupManager *SLLMBackupManager

func init() {
	GetLLMBackupManager()
}

func GetLLMBackupManager() *SLLMBackupManager {
	if llmBackupManager != nil {
		return llmBackupManager
	}
	llmBackupManager = &SLLMBackupManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SLLMBackup{},
			"llm_backups_tbl",
			"llm_backup",
			"llm_backups",
		),
	}

	llmBackupManager.SetVirtualObject(llmBackupManager)
	llmBackupManager.TableSpec().AddIndex(true, "deleted", "llm_id", "volume_id")

	return llmBackupManager
}

type SLLMBackupManager struct {
	db.SVirtualResourceBaseManager
	SMountedModelsResourceManager
}

type SLLMBackup struct {
	db.SVirtualResourceBase
	SMountedModelsResource

	// llm 规格ID，不超过32个字节。
	LLMSkuId string `width:"128" charset:"ascii" nullable:"true" list:"user"`
	// llm 镜像ID，不超过32个字节。
	LLMImageId string `width:"128" charset:"ascii" nullable:"true" list:"user"`
	// llm ID
	LLMId string `width:"128" charset:"ascii" nullable:"true" list:"user"`
	// llm 名称
	LLMName string `width:"128" charset:"utf8" nullable:"true" list:"user"`

	// 数据盘ID
	VolumeId string `width:"128" charset:"ascii" nullable:"true" list:"user"`
	// 云手机数据盘名称
	VolumeName string `width:"128" charset:"utf8" nullable:"true" list:"user"`

	VolumeSizeMB int `list:"user"`

	StorageType string `width:"16" charset:"ascii" nullable:"true" list:"user"`

	TemplateId string `width:"128" charset:"ascii" nullable:"true" list:"user"`

	IncludeFiles []string `charset:"utf8" list:"user"`
	ExcludeFiles []string `charset:"utf8" list:"user"`

	// 磁盘备份ID
	DiskbackupId string `width:"128" charset:"ascii" nullable:"true" list:"user"`
	// 磁盘备份大小
	BackupSizeMb int `list:"user"`

	// MountedApps []string `charset:"utf8" list:"user"`
}
