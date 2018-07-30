package models

import (
	"github.com/jinzhu/gorm"

	"github.com/yunionio/pkg/utils"
)

const (
	DiskResourceName = "disk"

	DiskInit          = "init"
	DiskRebuild       = "rebuild"
	DiskAllocFailed   = "alloc_failed"
	DiskStartAlloc    = "start_alloc"
	DiskAllocating    = "allocating"
	DiskReady         = "ready"
	DiskFrozen        = "frozen"
	DiskDealloc       = "deallocating"
	DiskDeallocFailed = "dealloc_failed"

	DiskStartSave = "start_save"
	DiskSaving    = "saving"

	DiskStartResize = "start_resize"
	DiskResizing    = "resizing"

	DiskStartMigrate = "start_migrate"
	DiskPostMigrate  = "post_migrate"
	DiskMigrating    = "migrating"

	TakeMebsSnapshot        = "take_mebs_snapshot"
	TakeMebsSnapshotFailed  = "take_mebs_snapshot_failed"
	ApplyMebsSnapshot       = "apply_mebs_snapshot"
	ApplyMebsSnapshotFailed = "apply_mebs_snapshot_failed"
	CloneMebsSnapshot       = "clone_mebs_snapshot"
	CloneMebsSnapshotFailed = "clone_mebs_snapshot_failed"
	PerformMebsBackup       = "perform_mebs_backup"
	PerformMebsBackupFailed = "perform_mebs_backup_failed"
	RestoreMebsBackup       = "restore_mebs_backup"
	RestoreMebsBackupFailed = "restore_mebs_backup_failed"
	SaveMebsTemplate        = "save_mebs_template"
	SaveMebsTemplateFailed  = "save_mebs_template_failed"

	ActionThrottle = "throttle"
	ActionFreeze   = "freeze"
	ActionUnfreeze = "unfreeze"
)

var (
	IOThrottleActions = []string{ActionThrottle, ActionFreeze, ActionUnfreeze}
)

type Disk struct {
	SharableVirtualResourceModel
	DiskFormat string `json:"disk_format" gorm:"column:disk_format;not null"`
	DiskSize   int64  `json:"disk_size" gorm:"column:disk_size;not null"`
	AccessPath string `json:"access_path" gorm:"column:access_path;not null"`
	AutoDelete bool   `json:"auto_delete" gorm:"column:auto_delete;not null"`
	StorageID  string `json:"storage_id" gorm:"column:storage_id;not null"`
	MebsInfo   string `json:"mebs_info" gorm:"column:mebs_info;type:text"`
}

func (d Disk) TableName() string {
	return disksTable
}

func (d Disk) String() string {
	str, _ := JsonString(d)
	return str
}

func NewDiskResource(db *gorm.DB) (Resourcer, error) {
	return newResource(db, disksTable,
		func() interface{} { return &Disk{} },
		func() interface{} { return &([]Disk{}) })
}

func (d Disk) Storage() (*Storage, error) {
	s, err := FetchByID(Storages, d.StorageID)
	if err != nil {
		return nil, err
	}
	return s.(*Storage), nil
}

func (d Disk) IsLocal() (bool, error) {
	s, err := d.Storage()
	if err != nil {
		return false, err
	}
	return utils.IsLocalStorage(s.StorageType), nil
}
