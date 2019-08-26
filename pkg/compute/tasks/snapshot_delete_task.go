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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

func init() {
	taskman.RegisterTask(SnapshotDeleteTask{})
	taskman.RegisterTask(BatchSnapshotsDeleteTask{})
}

/***************************** Snapshot Delete Task *****************************/

type SnapshotDeleteTask struct {
	taskman.STask
}

func (self *SnapshotDeleteTask) OnRequestSnapshotFailed(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, snapshot, data.String())
}

func (self *SnapshotDeleteTask) OnRequestSnapshot(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	err := snapshot.GetRegionDriver().OnSnapshotDelete(ctx, snapshot, self, data)
	if err != nil {
		self.TaskFailed(ctx, snapshot, err.Error())
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
	if err := regionDriver.RequestDeleteSnapshot(ctx, snapshot, self); err != nil {
		self.TaskFailed(ctx, snapshot, err.Error())
	}
}

func (self *SnapshotDeleteTask) OnDeleteSnapshot(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
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
