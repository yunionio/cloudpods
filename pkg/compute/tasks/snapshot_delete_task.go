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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

func init() {
	taskman.RegisterTask(SnapshotDeleteTask{})
	taskman.RegisterTask(BatchSnapshotsDeleteTask{})
	taskman.RegisterTask(GuestDeleteSnapshotsTask{})
	taskman.RegisterTask(DiskDeleteSnapshotsTask{})
}

/***************************** Snapshot Delete Task *****************************/

type SnapshotDeleteTask struct {
	taskman.STask
}

func (self *SnapshotDeleteTask) OnRequestSnapshotFailed(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, snapshot, data)
}

func (self *SnapshotDeleteTask) OnRequestSnapshot(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	err := snapshot.GetRegionDriver().OnSnapshotDelete(ctx, snapshot, self, data)
	if err != nil {
		self.TaskFailed(ctx, snapshot, jsonutils.NewString(err.Error()))
	}
}

func (self *SnapshotDeleteTask) OnManagedSnapshotDelete(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	snapshot.RealDelete(ctx, self.GetUserCred())
	self.TaskComplete(ctx, snapshot, nil)
}

func (self *SnapshotDeleteTask) OnKvmSnapshotDelete(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	snapshot.SetStatus(self.UserCred, api.SNAPSHOT_READY, "")
	if jsonutils.QueryBoolean(self.Params, "reload_disk", false) && snapshot.OutOfChain {
		self.SetStage("OnReloadDiskSnapshot", nil)
		self.OnReloadDiskSnapshot(ctx, snapshot, data)
	} else {
		self.SetStage("OnDeleteSnapshot", nil)
		self.OnDeleteSnapshot(ctx, snapshot, data)
	}
}

func (self *SnapshotDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	snapshot := obj.(*models.SSnapshot)
	regionDriver := snapshot.GetRegionDriver()

	self.SetStage("OnRequestSnapshot", nil)
	err := regionDriver.RequestDeleteSnapshot(ctx, snapshot, self)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.ScheduleRun(jsonutils.Marshal(map[string]bool{"deleted": true}))
			return
		}
		self.TaskFailed(ctx, snapshot, jsonutils.NewString(err.Error()))
	}
}

func (self *SnapshotDeleteTask) OnDeleteSnapshot(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	if !jsonutils.QueryBoolean(data, "deleted", false) {
		log.Infof("OnDeleteSnapshot with no deleted")
		return
	}

	snapshot.SetStatus(self.UserCred, api.SNAPSHOT_READY, "OnDeleteSnapshot")
	snapshot.RealDelete(ctx, self.UserCred)
	self.TaskComplete(ctx, snapshot, nil)
}

func (self *SnapshotDeleteTask) OnDeleteSnapshotFailed(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, snapshot, data)
}

func (self *SnapshotDeleteTask) OnReloadDiskSnapshot(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	if !jsonutils.QueryBoolean(data, "reopen", false) {
		log.Infof("OnReloadDiskSnapshot with no reopen")
		return
	}

	guest, err := snapshot.GetGuest()
	if err != nil {
		self.TaskFailed(ctx, snapshot, jsonutils.NewString(err.Error()))
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
			self.TaskFailed(ctx, snapshot, jsonutils.NewString(err.Error()))
		}
	} else {
		self.TaskComplete(ctx, snapshot, nil)
	}
}

func (self *SnapshotDeleteTask) TaskComplete(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(snapshot, db.ACT_SNAPSHOT_DELETE, snapshot.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, snapshot, logclient.ACT_DELOCATE, nil, self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    snapshot,
		Action: notifyclient.ActionDelete,
	})
	self.SetStageComplete(ctx, nil)
	guest, err := snapshot.GetGuest()
	if err != nil {
		log.Errorln(err.Error())
		return
	}
	guest.StartSyncstatus(ctx, self.UserCred, "")
}

func (self *SnapshotDeleteTask) TaskFailed(ctx context.Context, snapshot *models.SSnapshot, reason jsonutils.JSONObject) {
	snapshot.SetStatus(self.UserCred, api.SNAPSHOT_DELETE_FAILED, reason.String())
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
	host, err := snapshot.GetHost()
	if err != nil {
		self.SetStageFailed(ctx, jsonutils.NewString(errors.Wrapf(err, "snapshot.GetHost").Error()))
		return
	}

	snapshotIds := []string{}
	err = self.Params.Unmarshal(&snapshotIds, "snapshot_ids")
	if err != nil {
		self.SetStageFailed(ctx, jsonutils.NewString(errors.Wrapf(err, "unmarshal snapshot ids").Error()))
		return
	}

	self.SetStage("OnStorageDeleteSnapshot", nil)
	err = host.GetHostDriver().RequestDeleteSnapshotsWithStorage(ctx, host, snapshot, self, snapshotIds)
	if err != nil {
		self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
	}
}

func (self *BatchSnapshotsDeleteTask) OnStorageDeleteSnapshot(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	snapshots := models.SnapshotManager.GetDiskSnapshots(snapshot.DiskId)
	for i := 0; i < len(snapshots); i++ {
		snapshots[i].RealDelete(ctx, self.UserCred)
	}
	self.SetStageComplete(ctx, nil)
}

type GuestDeleteSnapshotsTask struct {
	taskman.STask
}

func (self *GuestDeleteSnapshotsTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	instanceSnapshots, _ := guest.GetInstanceSnapshots()
	self.StartDeleteInstanceSnapshots(ctx, guest, instanceSnapshots)
}

func (self *GuestDeleteSnapshotsTask) StartDeleteInstanceSnapshots(
	ctx context.Context, guest *models.SGuest, instanceSnapshots []models.SInstanceSnapshot) {
	if len(instanceSnapshots) > 0 {
		instanceSnapshot := instanceSnapshots[0]
		instanceSnapshots := instanceSnapshots[1:]
		self.Params.Set("instance_snapshots", jsonutils.Marshal(instanceSnapshots))
		self.SetStage("OnInstanceSnapshotDelete", nil)
		instanceSnapshot.SetModelManager(models.InstanceSnapshotManager, &instanceSnapshot)
		instanceSnapshot.StartInstanceSnapshotDeleteTask(ctx, self.UserCred, self.Id)
		return
	}
	snapshots, _ := guest.GetDiskSnapshotsNotInInstanceSnapshots()
	self.StartDeleteDiskSnapshots(ctx, guest, snapshots)
}

func (self *GuestDeleteSnapshotsTask) OnInstanceSnapshotDelete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	instanceSnapshots := make([]models.SInstanceSnapshot, 0)
	self.Params.Unmarshal(&instanceSnapshots, "instance_snapshots")
	self.StartDeleteInstanceSnapshots(ctx, guest, instanceSnapshots)
}

func (self *GuestDeleteSnapshotsTask) OnInstanceSnapshotDeleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	log.Errorln(data.String())
	instanceSnapshots := make([]models.SInstanceSnapshot, 0)
	self.Params.Unmarshal(&instanceSnapshots, "instance_snapshots")
	self.StartDeleteInstanceSnapshots(ctx, guest, instanceSnapshots)
}

func (self *GuestDeleteSnapshotsTask) StartDeleteDiskSnapshots(
	ctx context.Context, guest *models.SGuest, snapshots []models.SSnapshot) {
	if len(snapshots) > 0 {
		snapshot := snapshots[0]
		snapshots := snapshots[1:]
		self.Params.Set("snapshots", jsonutils.Marshal(snapshots))
		self.SetStage("OnSnapshotDelete", nil)
		snapshot.SetModelManager(models.SnapshotManager, &snapshot)
		snapshot.StartSnapshotDeleteTask(ctx, self.UserCred, false, self.Id)
		return
	}
	self.SetStageComplete(ctx, nil)
}

func (self *GuestDeleteSnapshotsTask) OnSnapshotDelete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	snapshots := make([]models.SSnapshot, 0)
	self.Params.Unmarshal(&snapshots, "snapshots")
	self.StartDeleteDiskSnapshots(ctx, guest, snapshots)
}

func (self *GuestDeleteSnapshotsTask) OnSnapshotDeleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	log.Errorln(data.String())
	snapshots := make([]models.SSnapshot, 0)
	self.Params.Unmarshal(&snapshots, "snapshots")
	self.StartDeleteDiskSnapshots(ctx, guest, snapshots)
}

type DiskDeleteSnapshotsTask struct {
	taskman.STask
}

func (self *DiskDeleteSnapshotsTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)
	snapshots, _ := disk.GetSnapshotsNotInInstanceSnapshot()
	self.StartDeleteDiskSnapshots(ctx, disk, snapshots)
}

func (self *DiskDeleteSnapshotsTask) StartDeleteDiskSnapshots(
	ctx context.Context, disk *models.SDisk, snapshots []models.SSnapshot) {
	if len(snapshots) > 0 {
		snapshot := snapshots[0]
		snapshots := snapshots[1:]
		self.Params.Set("snapshots", jsonutils.Marshal(snapshots))
		self.SetStage("OnSnapshotDelete", nil)
		snapshot.SetModelManager(models.SnapshotManager, &snapshot)
		snapshot.StartSnapshotDeleteTask(ctx, self.UserCred, false, self.Id)
		return
	}
	self.SetStageComplete(ctx, nil)
}

func (self *DiskDeleteSnapshotsTask) OnSnapshotDelete(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	snapshots := make([]models.SSnapshot, 0)
	self.Params.Unmarshal(&snapshots, "snapshots")
	self.StartDeleteDiskSnapshots(ctx, disk, snapshots)
}

func (self *DiskDeleteSnapshotsTask) OnSnapshotDeleteFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	log.Errorf("Delete disk snapshots failed %s", data.String())
	self.SetStageFailed(ctx, data)
}
