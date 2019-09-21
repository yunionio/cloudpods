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

package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/stringutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type DiskCreateTask struct {
	SDiskBaseTask
}

func (self *DiskCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)

	storagecache := disk.GetStorage().GetStoragecache()
	imageId := disk.GetTemplateId()
	if len(imageId) > 0 {
		if len(disk.BackupStorageId) > 0 {
			self.SetStage("OnMasterStorageCacheImageComplete", nil)
		} else {
			self.SetStage("OnStorageCacheImageComplete", nil)
		}
		storagecache.StartImageCacheTask(ctx, self.UserCred, imageId, disk.DiskFormat, false, self.GetTaskId())
	} else {
		self.OnStorageCacheImageComplete(ctx, disk, nil)
	}
}

func (self *DiskCreateTask) OnMasterStorageCacheImageComplete(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	storage := models.StorageManager.FetchStorageById(disk.BackupStorageId)
	storagecache := storage.GetStoragecache()
	imageId := disk.GetTemplateId()
	self.SetStage("OnStorageCacheImageComplete", nil)
	storagecache.StartImageCacheTask(ctx, self.UserCred, imageId, disk.DiskFormat, false, self.GetTaskId())
}

func (self *DiskCreateTask) OnStorageCacheImageComplete(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	rebuild, _ := self.GetParams().Bool("rebuild")
	snapshot, _ := self.GetParams().GetString("snapshot")
	if rebuild {
		db.OpsLog.LogEvent(disk, db.ACT_DELOCATE, disk.GetShortDesc(ctx), self.GetUserCred())
	}
	storage := disk.GetStorage()
	host := storage.GetMasterHost()
	db.OpsLog.LogEvent(disk, db.ACT_ALLOCATING, disk.GetShortDesc(ctx), self.GetUserCred())
	disk.SetStatus(self.GetUserCred(), api.DISK_STARTALLOC, fmt.Sprintf("Disk start alloc use host %s(%s)", host.Name, host.Id))
	if len(disk.BackupStorageId) > 0 {
		self.SetStage("OnMasterStorageCreateDiskComplete", nil)
	} else {
		if rebuild && storage.StorageType == api.STORAGE_RBD {
			if count, _ := disk.GetSnapshotCount(); count > 0 {
				backingDiskId := stringutils.UUID4()
				self.Params.Set("backing_disk_id", jsonutils.NewString(backingDiskId))
			}
		}
		self.SetStage("OnDiskReady", nil)
	}
	if err := disk.StartAllocate(ctx, host, storage, self.GetTaskId(), self.GetUserCred(), rebuild, snapshot, self); err != nil {
		self.OnStartAllocateFailed(ctx, disk, jsonutils.NewString(err.Error()))
	}
}

func (self *DiskCreateTask) OnMasterStorageCreateDiskComplete(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	rebuild, _ := self.GetParams().Bool("rebuild")
	snapshot, _ := self.GetParams().GetString("snapshot")
	storage := models.StorageManager.FetchStorageById(disk.BackupStorageId)
	host := storage.GetMasterHost()
	db.OpsLog.LogEvent(disk, db.ACT_BACKUP_ALLOCATING, disk.GetShortDesc(ctx), self.GetUserCred())
	disk.SetStatus(self.UserCred, api.DISK_BACKUP_STARTALLOC, fmt.Sprintf("Backup disk start alloc use host %s(%s)", host.Name, host.Id))
	self.SetStage("OnDiskReady", nil)
	if err := disk.StartAllocate(ctx, host, storage, self.GetTaskId(), self.GetUserCred(), rebuild, snapshot, self); err != nil {
		self.OnBackupAllocateFailed(ctx, disk, jsonutils.NewString(fmt.Sprintf("Backup disk alloctate failed: %s", err.Error())))
	}
}

func (self *DiskCreateTask) OnBackupAllocateFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	disk.SetStatus(self.UserCred, api.DISK_BACKUP_ALLOC_FAILED, data.String())
	self.SetStageFailed(ctx, data.String())
}

func (self *DiskCreateTask) OnStartAllocateFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	disk.SetStatus(self.UserCred, api.DISK_ALLOC_FAILED, data.String())
	self.SetStageFailed(ctx, data.String())
}

func (self *DiskCreateTask) OnDiskReady(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	diskSize, _ := data.Int("disk_size")
	if _, err := db.Update(disk, func() error {
		disk.DiskSize = int(diskSize)
		disk.DiskFormat, _ = data.GetString("disk_format")
		disk.AccessPath, _ = data.GetString("disk_path")
		return nil
	}); err != nil {
		log.Errorf("update disk info error: %v", err)
	}
	if jsonutils.QueryBoolean(self.Params, "rebuild", false) {
		backingDiskId, _ := self.Params.GetString("backing_disk_id")
		if len(backingDiskId) > 0 {
			err := disk.UpdataSnapshotsBackingDisk(backingDiskId)
			if err != nil {
				log.Errorf("update disk snapshots backing disk fiailed %s", err)
			}
		}
	}

	disk.SetStatus(self.UserCred, api.DISK_READY, "")
	self.CleanHostSchedCache(disk)
	db.OpsLog.LogEvent(disk, db.ACT_ALLOCATE, disk.GetShortDesc(ctx), self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *DiskCreateTask) OnDiskReadyFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	disk.SetStatus(self.UserCred, api.DISK_ALLOC_FAILED, data.String())
	self.SetStageFailed(ctx, data.String())
}

type DiskCreateBackupTask struct {
	DiskCreateTask
}

func (self *DiskCreateBackupTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)
	storage := models.StorageManager.FetchStorageById(disk.BackupStorageId)
	storagecache := storage.GetStoragecache()
	imageId := disk.GetTemplateId()
	if len(imageId) > 0 {
		self.SetStage("OnMasterStorageCreateDiskComplete", nil)
		storagecache.StartImageCacheTask(ctx, self.UserCred, imageId, disk.DiskFormat, false, self.GetTaskId())
	} else {
		self.OnMasterStorageCreateDiskComplete(ctx, disk, nil)
	}
}

func (self *DiskCreateBackupTask) OnDiskReady(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	bkStorage := models.StorageManager.FetchStorageById(disk.BackupStorageId)
	bkStorage.ClearSchedDescCache()
	disk.SetStatus(self.UserCred, api.DISK_READY, "")
	db.OpsLog.LogEvent(disk, db.ACT_BACKUP_ALLOCATE, disk.GetShortDesc(ctx), self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func init() {
	taskman.RegisterTask(DiskCreateTask{})
	taskman.RegisterTask(DiskCreateBackupTask{})
}
