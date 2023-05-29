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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/stringutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DiskCreateTask struct {
	SDiskBaseTask
}

func (self *DiskCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)

	storage, _ := disk.GetStorage()
	storagecache := storage.GetStoragecache()
	imageId := disk.GetTemplateId()
	// use image only if disk not created from snapshot or backup
	if len(imageId) > 0 && len(disk.SnapshotId) == 0 && len(disk.BackupId) == 0 {
		self.SetStage("OnStorageCacheImageComplete", nil)
		input := api.CacheImageInput{
			ImageId:      imageId,
			Format:       disk.GetCacheImageFormat(),
			ParentTaskId: self.GetTaskId(),
		}
		guest := disk.GetGuest()
		if guest != nil {
			input.ServerId = guest.Id
		}
		storagecache.StartImageCacheTask(ctx, self.UserCred, input)
	} else {
		self.OnStorageCacheImageComplete(ctx, disk, nil)
	}
}

func (self *DiskCreateTask) OnStorageCacheImageComplete(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	rebuild, _ := self.GetParams().Bool("rebuild")
	snapshot, _ := self.GetParams().GetString("snapshot")
	if rebuild {
		db.OpsLog.LogEvent(disk, db.ACT_DELOCATE, disk.GetShortDesc(ctx), self.GetUserCred())
	} else {
		guest := disk.GetGuest()
		if guest != nil {
			guest.SetStatus(self.GetUserCred(), api.VM_CREATE_DISK, "OnStorageCacheImageComplete")
		}
	}
	storage, err := disk.GetStorage()
	if err != nil {
		self.OnStartAllocateFailed(ctx, disk, jsonutils.NewString(errors.Wrapf(err, "disk.GetStorage").Error()))
		return
	}
	host, err := disk.GetMasterHost()
	if err != nil {
		self.OnStartAllocateFailed(ctx, disk, jsonutils.NewString(errors.Wrapf(err, "GetMasterHost").Error()))
		return
	}
	db.OpsLog.LogEvent(disk, db.ACT_ALLOCATING, disk.GetShortDesc(ctx), self.GetUserCred())
	disk.SetStatus(self.GetUserCred(), api.DISK_STARTALLOC, fmt.Sprintf("Disk start alloc use host %s(%s)", host.Name, host.Id))
	if rebuild && storage.StorageType == api.STORAGE_RBD {
		if count, _ := disk.GetSnapshotCount(); count > 0 {
			backingDiskId := stringutils.UUID4()
			self.Params.Set("backing_disk_id", jsonutils.NewString(backingDiskId))
		}
	}
	self.SetStage("OnDiskReady", nil)
	if err := disk.StartAllocate(ctx, host, storage, self.GetTaskId(), self.GetUserCred(), rebuild, snapshot, self); err != nil {
		self.OnStartAllocateFailed(ctx, disk, jsonutils.NewString(err.Error()))
	}
}

func (self *DiskCreateTask) OnStartAllocateFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	disk.SetStatus(self.UserCred, api.DISK_ALLOC_FAILED, data.String())
	logclient.AddActionLogWithStartable(self, disk, logclient.ACT_ALLOCATE, data, self.UserCred, false)
	self.SetStageFailed(ctx, data)
}

func (self *DiskCreateTask) OnDiskReady(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	diskSize, _ := data.Int("disk_size")
	if _, err := db.Update(disk, func() error {
		disk.DiskSize = int(diskSize)
		diskFromat, _ := data.GetString("disk_format")
		if len(diskFromat) > 0 {
			disk.DiskFormat = diskFromat
		}
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
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    disk,
		Action: notifyclient.ActionCreate,
	})
	self.SetStageComplete(ctx, nil)
}

func (self *DiskCreateTask) OnDiskReadyFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	disk.SetStatus(self.UserCred, api.DISK_ALLOC_FAILED, data.String())
	logclient.AddActionLogWithStartable(self, disk, logclient.ACT_ALLOCATE, data, self.UserCred, false)
	self.SetStageFailed(ctx, data)
}

func init() {
	taskman.RegisterTask(DiskCreateTask{})
}
