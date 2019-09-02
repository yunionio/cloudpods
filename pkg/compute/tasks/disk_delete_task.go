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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
)

type DiskDeleteTask struct {
	SDiskBaseTask
}

func init() {
	taskman.RegisterTask(DiskDeleteTask{})
}

func (self *DiskDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)

	cnt, err := disk.GetGuestDiskCount()
	if err != nil {
		reason := "Disk GetGuestDiskCount fail: " + err.Error()
		self.SetStageFailed(ctx, reason)
		db.OpsLog.LogEvent(disk, db.ACT_DELOCATE_FAIL, reason, self.UserCred)
		return
	}
	if cnt > 0 {
		reason := "Disk has been attached to server"
		self.SetStageFailed(ctx, reason)
		db.OpsLog.LogEvent(disk, db.ACT_DELOCATE_FAIL, reason, self.UserCred)
		return
	}

	isPurge := jsonutils.QueryBoolean(self.Params, "purge", false)
	overridePendingDelete := jsonutils.QueryBoolean(self.Params, "override_pending_delete", false)
	if options.Options.EnablePendingDelete && !isPurge && !overridePendingDelete {
		if disk.PendingDeleted {
			self.SetStageComplete(ctx, nil)
			return
		}
		self.startPendingDeleteDisk(ctx, disk)
	} else {
		self.startDeleteDisk(ctx, disk)
	}
}

func (self *DiskDeleteTask) startDeleteDisk(ctx context.Context, disk *models.SDisk) {
	db.OpsLog.LogEvent(disk, db.ACT_DELOCATING, disk.GetShortDesc(ctx), self.UserCred)
	if disk.Status == api.DISK_INIT {
		self.OnGuestDiskDeleteComplete(ctx, disk, nil)
		return
	}
	var (
		storage *models.SStorage
		host    *models.SHost
	)

	storage = disk.GetStorage()
	if storage != nil {
		host = storage.GetMasterHost()
	}

	isPurge := false
	if (host == nil || !host.Enabled) && jsonutils.QueryBoolean(self.Params, "purge", false) {
		isPurge = true
	}
	disk.SetStatus(self.UserCred, api.DISK_DEALLOC, "")
	if isPurge {
		self.OnGuestDiskDeleteComplete(ctx, disk, nil)
	} else {
		if len(disk.BackupStorageId) > 0 {
			self.SetStage("OnMasterStorageDeleteDiskComplete", nil)
		} else {
			self.SetStage("OnGuestDiskDeleteComplete", nil)
		}
		if host == nil {
			self.OnGuestDiskDeleteCompleteFailed(ctx, disk, jsonutils.NewString("fail to find master host"))
		} else if err := host.GetHostDriver().RequestDeallocateDiskOnHost(ctx, host, storage, disk, self); err != nil {
			self.OnGuestDiskDeleteCompleteFailed(ctx, disk, jsonutils.NewString(err.Error()))
		}
	}
}

func (self *DiskDeleteTask) OnMasterStorageDeleteDiskComplete(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	self.SetStage("OnGuestDiskDeleteComplete", nil)
	storage := models.StorageManager.FetchStorageById(disk.BackupStorageId)
	host := storage.GetMasterHost()
	if host == nil {
		self.OnGuestDiskDeleteCompleteFailed(ctx, disk, jsonutils.NewString(fmt.Sprintf("backup storage %s fail to find master host", disk.BackupStorageId)))
	} else if err := host.GetHostDriver().RequestDeallocateDiskOnHost(ctx, host, storage, disk, self); err != nil {
		self.OnGuestDiskDeleteCompleteFailed(ctx, disk, jsonutils.NewString(err.Error()))
	}
}

func (self *DiskDeleteTask) OnMasterStorageDeleteDiskCompleteFailed(ctx context.Context, disk *models.SDisk, reason jsonutils.JSONObject) {
	self.OnGuestDiskDeleteCompleteFailed(ctx, disk, reason)
}

func (self *DiskDeleteTask) startPendingDeleteDisk(ctx context.Context, disk *models.SDisk) {
	err := disk.DoPendingDelete(ctx, self.UserCred)
	if err != nil {
		self.OnGuestDiskDeleteCompleteFailed(ctx, disk, jsonutils.NewString("pending delete disk failed"))
		return
	}
	err = models.SnapshotPolicyDiskManager.SyncDetachByDisk(ctx, self.UserCred, nil, disk)
	if err != nil {
		self.OnGuestDiskDeleteCompleteFailed(ctx, disk,
			jsonutils.NewString("detach all snapshotpolicies of disk failed"))
		return
	}
	self.SetStageComplete(ctx, nil)
}

func (self *DiskDeleteTask) OnGuestDiskDeleteComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	if obj == nil {
		self.SetStageComplete(ctx, nil)
		return
	}
	disk := obj.(*models.SDisk)
	self.CleanHostSchedCache(disk)
	db.OpsLog.LogEvent(disk, db.ACT_DELOCATE, disk.GetShortDesc(ctx), self.UserCred)
	if len(disk.SnapshotId) > 0 && disk.GetMetadata("merge_snapshot", nil) == "true" {
		models.SnapshotManager.AddRefCount(disk.SnapshotId, -1)
	}

	disk.RealDelete(ctx, self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *DiskDeleteTask) OnGuestDiskDeleteCompleteFailed(ctx context.Context, disk *models.SDisk, reason jsonutils.JSONObject) {
	disk.SetStatus(self.GetUserCred(), api.DISK_DEALLOC_FAILED, reason.String())
	self.SetStageFailed(ctx, reason.String())
	db.OpsLog.LogEvent(disk, db.ACT_DELOCATE_FAIL, disk.GetShortDesc(ctx), self.GetUserCred())
}
