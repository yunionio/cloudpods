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
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

func init() {
	taskman.RegisterTask(HADiskCreateTask{})
	taskman.RegisterTask(DiskCreateBackupTask{})
}

type HADiskCreateTask struct {
	DiskCreateTask
}

func (self *HADiskCreateTask) OnStorageCacheImageComplete(
	ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject,
) {
	storage := models.StorageManager.FetchStorageById(disk.BackupStorageId)
	storagecache := storage.GetStoragecache()
	imageId := disk.GetTemplateId()
	self.SetStage("OnBackupStorageCacheImageComplete", nil)
	input := api.CacheImageInput{
		ImageId:      imageId,
		Format:       disk.DiskFormat,
		ParentTaskId: self.GetTaskId(),
	}
	storagecache.StartImageCacheTask(ctx, self.UserCred, input)
}

func (self *HADiskCreateTask) OnBackupStorageCacheImageComplete(
	ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject,
) {
	self.DiskCreateTask.OnStorageCacheImageComplete(ctx, disk, data)
}

func (self *HADiskCreateTask) OnDiskReady(
	ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject,
) {
	rebuild, _ := self.GetParams().Bool("rebuild")
	snapshot, _ := self.GetParams().GetString("snapshot")
	storage := models.StorageManager.FetchStorageById(disk.BackupStorageId)
	host, err := storage.GetMasterHost()
	if err != nil {
		self.OnBackupAllocateFailed(ctx, disk, jsonutils.NewString(errors.Wrapf(err, "storage.GetMasterHost").Error()))
		return
	}
	db.OpsLog.LogEvent(disk, db.ACT_BACKUP_ALLOCATING, disk.GetShortDesc(ctx), self.GetUserCred())
	disk.SetStatus(ctx, self.UserCred, api.DISK_BACKUP_STARTALLOC,
		fmt.Sprintf("Backup disk start alloc use host %s(%s)", host.Name, host.Id),
	)
	self.SetStage("OnSlaveDiskReady", nil)
	if err := disk.StartAllocate(ctx, host, storage,
		self.GetTaskId(), self.GetUserCred(), rebuild, snapshot, self,
	); err != nil {
		self.OnBackupAllocateFailed(ctx, disk,
			jsonutils.NewString(fmt.Sprintf("Backup disk alloctate failed: %s", err.Error())))
	}
}

func (self *HADiskCreateTask) OnBackupAllocateFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	disk.SetStatus(ctx, self.UserCred, api.DISK_BACKUP_ALLOC_FAILED, data.String())
	self.SetStageFailed(ctx, data)
}

func (self *HADiskCreateTask) OnSlaveDiskReady(
	ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject,
) {
	self.DiskCreateTask.OnDiskReady(ctx, disk, data)
}

type DiskCreateBackupTask struct {
	HADiskCreateTask
}

func (self *DiskCreateBackupTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)
	storage := models.StorageManager.FetchStorageById(disk.BackupStorageId)
	storagecache := storage.GetStoragecache()
	imageId := disk.GetTemplateId()
	if len(imageId) > 0 {
		self.SetStage("OnBackupStorageCacheImageComplete", nil)
		input := api.CacheImageInput{
			ImageId:      imageId,
			Format:       disk.DiskFormat,
			ParentTaskId: self.GetTaskId(),
		}
		storagecache.StartImageCacheTask(ctx, self.UserCred, input)
	} else {
		self.OnBackupStorageCacheImageComplete(ctx, disk, nil)
	}
}

func (self *DiskCreateBackupTask) OnBackupStorageCacheImageComplete(
	ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject,
) {
	self.HADiskCreateTask.OnDiskReady(ctx, disk, data)
}

func (self *DiskCreateBackupTask) OnSlaveDiskReady(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	bkStorage := models.StorageManager.FetchStorageById(disk.BackupStorageId)
	bkStorage.ClearSchedDescCache()
	disk.SetStatus(ctx, self.UserCred, api.DISK_READY, "")
	db.OpsLog.LogEvent(disk, db.ACT_BACKUP_ALLOCATE, disk.GetShortDesc(ctx), self.UserCred)
	self.SetStageComplete(ctx, nil)
}
