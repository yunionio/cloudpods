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
	"database/sql"
	"fmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DiskDeleteTask struct {
	SDiskBaseTask
}

func init() {
	taskman.RegisterTask(DiskDeleteTask{})
	taskman.RegisterTask(StorageDeleteRbdDiskTask{})
}

func (self *DiskDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)

	cnt, err := disk.GetGuestDiskCount()
	if err != nil {
		reason := "Disk GetGuestDiskCount fail: " + err.Error()
		self.SetStageFailed(ctx, jsonutils.NewString(reason))
		db.OpsLog.LogEvent(disk, db.ACT_DELOCATE_FAIL, reason, self.UserCred)
		return
	}
	if cnt > 0 {
		reason := "Disk has been attached to server"
		self.SetStageFailed(ctx, jsonutils.NewString(reason))
		db.OpsLog.LogEvent(disk, db.ACT_DELOCATE_FAIL, reason, self.UserCred)
		return
	}
	if jsonutils.QueryBoolean(self.Params, "delete_snapshots", false) {
		self.SetStage("OnDiskSnapshotDelete", nil)
		self.StartDeleteDiskSnapshots(ctx, disk)
	} else {
		self.OnDeleteSnapshots(ctx, disk)
	}
}

func (self *DiskDeleteTask) OnDeleteSnapshots(ctx context.Context, disk *models.SDisk) {
	isPurge := jsonutils.QueryBoolean(self.Params, "purge", false)
	overridePendingDelete := jsonutils.QueryBoolean(self.Params, "override_pending_delete", false)
	if len(disk.ExternalId) > 0 {
		_, err := disk.GetIDisk(ctx)
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			overridePendingDelete = true
		}
	}
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

func (self *DiskDeleteTask) StartDeleteDiskSnapshots(ctx context.Context, disk *models.SDisk) {
	disk.DeleteSnapshots(ctx, self.UserCred, self.GetId())
}

func (self *DiskDeleteTask) OnDiskSnapshotDelete(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	self.OnDeleteSnapshots(ctx, disk)
}

func (self *DiskDeleteTask) OnDiskSnapshotDeleteFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	log.Errorf("Delete disk snapshots failed %s", data.String())
	self.OnGuestDiskDeleteCompleteFailed(ctx, disk, data)
}

func (self *DiskDeleteTask) startDeleteDisk(ctx context.Context, disk *models.SDisk) {
	db.OpsLog.LogEvent(disk, db.ACT_DELOCATING, disk.GetShortDesc(ctx), self.UserCred)
	if disk.Status == api.DISK_INIT {
		self.OnGuestDiskDeleteComplete(ctx, disk, nil)
		return
	}

	storage, err := disk.GetStorage()
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows { // dirty data
			self.OnGuestDiskDeleteComplete(ctx, disk, nil)
			return
		}
		self.OnGuestDiskDeleteCompleteFailed(ctx, disk, jsonutils.NewString("disk.GetStorage"))
		return
	}

	purgeParams := jsonutils.QueryBoolean(self.Params, "purge", false)

	var host *models.SHost
	if hostId := disk.GetLastAttachedHost(ctx, self.UserCred); hostId != "" {
		host = models.HostManager.FetchHostById(hostId)
	}
	if host == nil {
		host, err = storage.GetMasterHost()
		if err != nil && errors.Cause(err) != sql.ErrNoRows && !purgeParams {
			self.OnGuestDiskDeleteCompleteFailed(ctx, disk, jsonutils.NewString("storage.GetMasterHost"))
			return
		}
	}

	if host != nil {
		disk.RecordDiskSnapshotsLastHost(ctx, self.UserCred, host.Id)
	}

	isPurge := false
	if (host == nil || !host.GetEnabled()) && purgeParams {
		isPurge = true
	}
	disk.SetStatus(ctx, self.UserCred, api.DISK_DEALLOC, "")
	if isPurge {
		self.OnGuestDiskDeleteComplete(ctx, disk, nil)
		return
	}
	if isNeed, _ := disk.IsNeedWaitSnapshotsDeleted(); isNeed { // for kvm rbd disk
		self.OnGuestDiskDeleteComplete(ctx, disk, nil)
		return
	}
	if len(disk.BackupStorageId) > 0 {
		self.SetStage("OnMasterStorageDeleteDiskComplete", nil)
	} else {
		self.SetStage("OnGuestDiskDeleteComplete", nil)
	}
	if host == nil {
		self.OnGuestDiskDeleteCompleteFailed(ctx, disk, jsonutils.NewString("fail to find master host"))
		return
	}
	driver, err := host.GetHostDriver()
	if err != nil {
		self.OnGuestDiskDeleteCompleteFailed(ctx, disk, jsonutils.NewString(errors.Wrapf(err, "GetHostDriver").Error()))
		return
	}
	err = driver.RequestDeallocateDiskOnHost(ctx, host, storage, disk, false, self)
	if err != nil {
		self.OnGuestDiskDeleteCompleteFailed(ctx, disk, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *DiskDeleteTask) OnMasterStorageDeleteDiskComplete(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	self.SetStage("OnGuestDiskDeleteComplete", nil)
	storage := models.StorageManager.FetchStorageById(disk.BackupStorageId)
	host, err := storage.GetMasterHost()
	if err != nil {
		self.OnGuestDiskDeleteCompleteFailed(ctx, disk, jsonutils.NewString(fmt.Sprintf("backup storage %s fail to find master host", disk.BackupStorageId)))
		return
	}
	driver, err := host.GetHostDriver()
	if err != nil {
		self.OnGuestDiskDeleteCompleteFailed(ctx, disk, jsonutils.NewString(err.Error()))
		return
	}
	err = driver.RequestDeallocateDiskOnHost(ctx, host, storage, disk, false, self)
	if err != nil {
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
	err = models.SnapshotPolicyDiskManager.RemoveByDisk(disk.Id)
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
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    disk,
		Action: notifyclient.ActionDelete,
	})
	if len(disk.SnapshotId) > 0 && disk.GetMetadata(ctx, "merge_snapshot", nil) == "true" {
		models.SnapshotManager.AddRefCount(disk.SnapshotId, -1)
	}
	disk.RealDelete(ctx, self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *DiskDeleteTask) OnGuestDiskDeleteCompleteFailed(ctx context.Context, disk *models.SDisk, reason jsonutils.JSONObject) {
	disk.SetStatus(ctx, self.GetUserCred(), api.DISK_DEALLOC_FAILED, reason.String())
	self.SetStageFailed(ctx, reason)
	db.OpsLog.LogEvent(disk, db.ACT_DELOCATE_FAIL, disk.GetShortDesc(ctx), self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, disk, logclient.ACT_DELOCATE, reason, self.UserCred, false)
}

type StorageDeleteRbdDiskTask struct {
	taskman.STask
}

func (self *StorageDeleteRbdDiskTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	storage := obj.(*models.SStorage)
	self.DeleteDisk(ctx, storage, self.Params)
}

func (self *StorageDeleteRbdDiskTask) OnDeleteDisk(ctx context.Context, storage *models.SStorage, data jsonutils.JSONObject) {
	self.DeleteDisk(ctx, storage, data)
}

func (self *StorageDeleteRbdDiskTask) DeleteDisk(ctx context.Context, storage *models.SStorage, data jsonutils.JSONObject) {
	disksId := make([]string, 0)
	data.Unmarshal(&disksId, "disks_id")
	if len(disksId) == 0 {
		self.SetStageComplete(ctx, nil)
		return
	}
	params := jsonutils.NewDict()
	params.Set("disks_id", jsonutils.Marshal(disksId[1:]))
	params.Set("delete_disk", jsonutils.NewString(disksId[0]))
	self.SetStage("OnDeleteDisk", params)
	header := self.GetTaskRequestHeader()
	url := fmt.Sprintf("/disks/%s/delete/%s", storage.Id, disksId[0])
	body := jsonutils.NewDict()
	host, err := storage.GetMasterHost()
	if err != nil {
		self.OnDeleteDiskFailed(ctx, storage, jsonutils.NewString("storage.GetMasterHost"))
		return
	}
	_, err = host.Request(ctx, self.GetUserCred(), "POST", url, header, body)
	if err != nil {
		params.Set("err", jsonutils.NewString(err.Error()))
		self.OnDeleteDiskFailed(ctx, storage, params)
	}
}

func (self *StorageDeleteRbdDiskTask) OnDeleteDiskFailed(ctx context.Context, storage *models.SStorage, data jsonutils.JSONObject) {
	deleteDisk, _ := data.GetString("delete_disk")
	db.OpsLog.LogEvent(storage, db.ACT_DELETE_OBJECT, fmt.Sprintf("delete disk %s failed", deleteDisk), self.UserCred)
	self.DeleteDisk(ctx, storage, self.Params)
}
