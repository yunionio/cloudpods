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

package models

import (
	"github.com/jinzhu/gorm"

	"yunion.io/x/pkg/utils"
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
