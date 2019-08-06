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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestDiskSnapshotTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestDiskSnapshotTask{})
	taskman.RegisterTask(SnapshotDeleteTask{})
	taskman.RegisterTask(BatchSnapshotsDeleteTask{})
}

func (self *GuestDiskSnapshotTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.DoDiskSnapshot(ctx, guest)
}

func (self *GuestDiskSnapshotTask) DoDiskSnapshot(ctx context.Context, guest *models.SGuest) {
	diskId, err := self.Params.GetString("disk_id")
	if err != nil {
		self.TaskFailed(ctx, guest, err.Error())
		return
	}
	snapshotId, err := self.Params.GetString("snapshot_id")
	if err != nil {
		self.TaskFailed(ctx, guest, err.Error())
		return
	}
	self.SetStage("OnDiskSnapshotComplete", nil)
	guest.SetStatus(self.UserCred, api.VM_SNAPSHOT, "")
	err = guest.GetDriver().RequestDiskSnapshot(ctx, guest, self, snapshotId, diskId)
	if err != nil {
		self.TaskFailed(ctx, guest, err.Error())
		return
	}
}

func (self *GuestDiskSnapshotTask) OnDiskSnapshotComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	res := data.(*jsonutils.JSONDict)
	snapshotId, _ := self.Params.GetString("snapshot_id")
	iSnapshot, _ := models.SnapshotManager.FetchById(snapshotId)
	snapshot := iSnapshot.(*models.SSnapshot)
	if guest.Hypervisor == api.HYPERVISOR_KVM {
		location, err := res.GetString("location")
		if err != nil {
			log.Infof("OnDiskSnapshotComplete called with data no location")
			return
		}
		db.Update(snapshot, func() error {
			snapshot.Location = location
			snapshot.Status = api.SNAPSHOT_READY
			return nil
		})
	} else {
		extSnapshotId, _ := data.GetString("snapshot_id")
		db.Update(snapshot, func() error {
			snapshot.ExternalId = extSnapshotId
			snapshot.Status = api.SNAPSHOT_READY
			return nil
		})
	}
	guest.SetStatus(self.UserCred, api.VM_SNAPSHOT_SUCC, "")
	self.TaskComplete(ctx, guest, nil)
}

func (self *GuestDiskSnapshotTask) OnDiskSnapshotCompleteFailed(ctx context.Context, guest *models.SGuest, err jsonutils.JSONObject) {
	self.TaskFailed(ctx, guest, err.String())
}

func (self *GuestDiskSnapshotTask) OnAutoDeleteSnapshot(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.TaskComplete(ctx, guest, data)
}

func (self *GuestDiskSnapshotTask) OnAutoDeleteSnapshotFailed(ctx context.Context, guest *models.SGuest, err jsonutils.JSONObject) {
	log.Errorf("Auto Delete Snapshot Failed %s", err.String())
	self.TaskComplete(ctx, guest, err)
}

func (self *GuestDiskSnapshotTask) TaskComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	snapshotId, _ := self.Params.GetString("snapshot_id")
	iSnapshot, _ := models.SnapshotManager.FetchById(snapshotId)
	db.OpsLog.LogEvent(iSnapshot, db.ACT_SNAPSHOT_DONE, iSnapshot.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_DISK_CREATE_SNAPSHOT, nil, self.UserCred, true)
	guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
	self.SetStage("OnSyncStatus", nil)
}

func (self *GuestDiskSnapshotTask) OnSyncStatus(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestDiskSnapshotTask) TaskFailed(ctx context.Context, guest *models.SGuest, reason string) {
	snapshotId, _ := self.Params.GetString("snapshot_id")
	iSnapshot, _ := models.SnapshotManager.FetchById(snapshotId)
	snapshot := iSnapshot.(*models.SSnapshot)
	db.Update(snapshot, func() error {
		snapshot.Status = api.SNAPSHOT_FAILED
		return nil
	})
	self.SetStageFailed(ctx, reason)
	guest.SetStatus(self.UserCred, api.VM_SNAPSHOT_FAILED, reason)
	db.OpsLog.LogEvent(iSnapshot, db.ACT_SNAPSHOT_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_DISK_CREATE_SNAPSHOT, reason, self.UserCred, false)
}

/***************************** Snapshot Delete Task *****************************/

type SnapshotDeleteTask struct {
	taskman.STask
}

func (self *SnapshotDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	snapshot := obj.(*models.SSnapshot)
	if len(snapshot.ExternalId) > 0 {
		err := self.deleteExternalSnapshot(ctx, snapshot)
		if err != nil {
			self.TaskFailed(ctx, snapshot, err.Error())
		} else {
			snapshot.RealDelete(ctx, self.GetUserCred())
			self.TaskComplete(ctx, snapshot, nil)
		}
		return
	}
	guest, err := snapshot.GetGuest()
	if err != nil {
		if err != sql.ErrNoRows {
			self.TaskFailed(ctx, snapshot, err.Error())
			return
		} else {
			// if snapshot is not used
			self.DeleteStaticSnapshot(ctx, snapshot)
			return
		}
	}
	if jsonutils.QueryBoolean(self.Params, "reload_disk", false) && snapshot.OutOfChain {
		self.StartReloadDisk(ctx, snapshot, guest)
	} else {
		self.StartDeleteSnapshot(ctx, snapshot, guest)
	}
}

func (self *SnapshotDeleteTask) deleteExternalSnapshot(ctx context.Context, snapshot *models.SSnapshot) error {
	cloudRegion, err := snapshot.GetISnapshotRegion()
	if err != nil {
		log.Errorln(err, cloudRegion, snapshot.CloudregionId)
		return err
	}
	cloudSnapshot, err := cloudRegion.GetISnapshotById(snapshot.ExternalId)
	if err != nil {
		if err == cloudprovider.ErrNotFound {
			return nil
		}
		log.Errorln(err, cloudSnapshot)
		return err
	}
	if err := cloudSnapshot.Delete(); err != nil {
		return err
	}
	return cloudprovider.WaitDeleted(cloudSnapshot, 10*time.Second, 300*time.Second)
}

func (self *SnapshotDeleteTask) StartReloadDisk(ctx context.Context, snapshot *models.SSnapshot, guest *models.SGuest) {
	self.SetStage("OnReloadDiskSnapshot", nil)
	guest.SetStatus(self.UserCred, api.VM_SNAPSHOT, "Start Reload Snapshot")
	params := jsonutils.NewDict()
	params.Set("disk_id", jsonutils.NewString(snapshot.DiskId))
	err := guest.GetDriver().RequestReloadDiskSnapshot(ctx, guest, self, params)
	if err != nil {
		self.TaskFailed(ctx, snapshot, err.Error())
	}
}

func (self *SnapshotDeleteTask) StartDeleteSnapshot(ctx context.Context, snapshot *models.SSnapshot, guest *models.SGuest) {
	snapshot.SetStatus(self.UserCred, api.SNAPSHOT_DELETING, "On SnapshotDeleteTask StartDeleteSnapshot")
	params := jsonutils.NewDict()
	convertSnapshot, err := models.SnapshotManager.GetConvertSnapshot(snapshot)
	if err != nil {
		self.TaskFailed(ctx, snapshot, err.Error())
		return
	}
	if convertSnapshot == nil {
		self.TaskFailed(ctx, snapshot, "snapshot dose not have convert snapshot")
		return
	}
	params.Set("delete_snapshot", jsonutils.NewString(snapshot.Id))
	params.Set("disk_id", jsonutils.NewString(snapshot.DiskId))
	if !snapshot.OutOfChain {
		params.Set("convert_snapshot", jsonutils.NewString(convertSnapshot.Id))
		var FakeDelete = jsonutils.JSONFalse
		if snapshot.CreatedBy == api.SNAPSHOT_MANUAL && snapshot.FakeDeleted == false {
			FakeDelete = jsonutils.JSONTrue
		}
		params.Set("pending_delete", FakeDelete)
	} else {
		params.Set("auto_deleted", jsonutils.JSONTrue)
	}
	guest.SetStatus(self.UserCred, api.VM_SNAPSHOT_DELETE, "Start Delete Snapshot")

	self.SetStage("OnDeleteSnapshot", nil)
	err = guest.GetDriver().RequestDeleteSnapshot(ctx, guest, self, params)
	if err != nil {
		self.TaskFailed(ctx, snapshot, err.Error())
	}
}

func (self *SnapshotDeleteTask) DeleteStaticSnapshot(ctx context.Context, snapshot *models.SSnapshot) {
	err := snapshot.FakeDelete(self.UserCred)
	if err != nil {
		self.TaskFailed(ctx, snapshot, err.Error())
		return
	}
	self.TaskComplete(ctx, snapshot, nil)
}

func (self *SnapshotDeleteTask) OnDeleteSnapshot(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	if len(snapshot.ExternalId) == 0 {
		if !jsonutils.QueryBoolean(data, "deleted", false) {
			log.Infof("OnDeleteSnapshot with no deleted")
			return
		}
		snapshot.SetStatus(self.UserCred, api.SNAPSHOT_READY, "OnDeleteSnapshot")
		if snapshot.OutOfChain {
			snapshot.RealDelete(ctx, self.UserCred)
			self.TaskComplete(ctx, snapshot, nil)
		} else {
			var FakeDelete = false
			if snapshot.CreatedBy == api.SNAPSHOT_MANUAL && snapshot.FakeDeleted == false {
				FakeDelete = true
			}
			if FakeDelete {
				db.Update(snapshot, func() error {
					snapshot.OutOfChain = true
					return nil
				})
			} else {
				snapshot.RealDelete(ctx, self.UserCred)
			}
			self.TaskComplete(ctx, snapshot, nil)
		}
	} else {
		snapshot.SetStatus(self.UserCred, api.SNAPSHOT_READY, "OnDeleteSnapshot")
		snapshot.RealDelete(ctx, self.UserCred)
		self.TaskComplete(ctx, snapshot, nil)
	}
}

func (self *SnapshotDeleteTask) OnDeleteSnapshotFailed(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, snapshot, data.String())
}

func (self *SnapshotDeleteTask) OnReloadDiskSnapshot(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	if !jsonutils.QueryBoolean(data, "reopen", false) {
		log.Infof("OnReloadDiskSnapshot with no reopen")
		return
	}

	guest, err := snapshot.GetGuest()
	if err != nil {
		self.TaskFailed(ctx, snapshot, err.Error())
		return
	}
	if snapshot.FakeDeleted {
		params := jsonutils.NewDict()
		params.Set("delete_snapshot", jsonutils.NewString(snapshot.Id))
		params.Set("disk_id", jsonutils.NewString(snapshot.DiskId))
		params.Set("auto_deleted", jsonutils.JSONTrue)
		self.SetStage("OnDeleteSnapshot", nil)
		err = guest.GetDriver().RequestDeleteSnapshot(ctx, guest, self, params)
		if err != nil {
			self.TaskFailed(ctx, snapshot, err.Error())
		}
	} else {
		self.TaskComplete(ctx, snapshot, nil)
	}
}

func (self *SnapshotDeleteTask) TaskComplete(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(snapshot, db.ACT_SNAPSHOT_DELETE, snapshot.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, snapshot, logclient.ACT_DELOCATE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
	guest, err := snapshot.GetGuest()
	if err != nil {
		log.Errorln(err.Error())
		return
	}
	guest.StartSyncstatus(ctx, self.UserCred, "")
}

func (self *SnapshotDeleteTask) TaskFailed(ctx context.Context, snapshot *models.SSnapshot, reason string) {
	if snapshot.Status == api.SNAPSHOT_DELETING {
		snapshot.SetStatus(self.UserCred, api.SNAPSHOT_READY, "On SnapshotDeleteTask TaskFailed")
	}
	db.OpsLog.LogEvent(snapshot, db.ACT_SNAPSHOT_DELETE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, snapshot, logclient.ACT_DELOCATE, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
	guest, err := snapshot.GetGuest()
	if err != nil {
		log.Errorln(err.Error())
		return
	}
	guest.StartSyncstatus(ctx, self.UserCred, "")
}

/***************************** Batch Snapshots Delete Task *****************************/

type BatchSnapshotsDeleteTask struct {
	taskman.STask
}

func (self *BatchSnapshotsDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	snapshot := obj.(*models.SSnapshot)
	self.StartStorageDeleteSnapshot(ctx, snapshot)
}

func (self *BatchSnapshotsDeleteTask) StartStorageDeleteSnapshot(ctx context.Context, snapshot *models.SSnapshot) {
	host := snapshot.GetHost()
	if host == nil {
		self.SetStageFailed(ctx, "Cannot found snapshot host")
		return
	}
	self.SetStage("OnStorageDeleteSnapshot", nil)
	err := host.GetHostDriver().RequestDeleteSnapshotsWithStorage(ctx, host, snapshot, self)
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
	}
}

func (self *BatchSnapshotsDeleteTask) OnStorageDeleteSnapshot(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	snapshots := models.SnapshotManager.GetDiskSnapshots(snapshot.DiskId)
	for i := 0; i < len(snapshots); i++ {
		snapshots[i].RealDelete(ctx, self.UserCred)
	}
	self.SetStageComplete(ctx, nil)
}
