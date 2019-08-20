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

type SnapshotCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SnapshotCreateTask{})
	taskman.RegisterTask(GuestDiskSnapshotTask{})
}

func (self *SnapshotCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	snapshot := obj.(*models.SSnapshot)
	self.DoDiskSnapshot(ctx, snapshot)
}

func (self *SnapshotCreateTask) TaskFailed(ctx context.Context, snapshot *models.SSnapshot, reason string) {
	db.Update(snapshot, func() error {
		snapshot.Status = api.SNAPSHOT_FAILED
		return nil
	})
	db.OpsLog.LogEvent(snapshot, db.ACT_SNAPSHOT_FAIL, reason, self.UserCred)
	self.SetStageFailed(ctx, reason)
}

func (self *SnapshotCreateTask) TaskComplete(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	db.Update(snapshot, func() error {
		snapshot.Status = api.SNAPSHOT_READY
		return nil
	})
	db.OpsLog.LogEvent(snapshot, db.ACT_SNAPSHOT_DONE, snapshot.GetShortDesc(ctx), self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *SnapshotCreateTask) DoDiskSnapshot(ctx context.Context, snapshot *models.SSnapshot) {
	self.SetStage("OnCreateSnapshot", nil)
	if err := snapshot.GetRegionDriver().RequestCreateSnapshot(ctx, snapshot, self); err != nil {
		self.TaskFailed(ctx, snapshot, err.Error())
	}
}

func (self *SnapshotCreateTask) OnCreateSnapshot(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	extSnapshotId, err := data.GetString("snapshot_id")
	if err == nil { // Managed snapshot
		_, err := db.Update(snapshot, func() error {
			snapshot.ExternalId = extSnapshotId
			snapshot.Status = api.SNAPSHOT_READY
			return nil
		})
		if err != nil {
			self.TaskFailed(ctx, snapshot, err.Error())
			return
		}
	} else {
		_, err = db.Update(snapshot, func() error {
			snapshot.Status = api.SNAPSHOT_READY
			return nil
		})
		if err != nil {
			self.TaskFailed(ctx, snapshot, err.Error())
			return
		}
	}
	self.TaskComplete(ctx, snapshot, nil)
}

func (self *SnapshotCreateTask) OnCreateSnapshotFailed(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, snapshot, data.String())
}

// =================================================================================================================

type GuestDiskSnapshotTask struct {
	SGuestBaseTask
}

func (self *GuestDiskSnapshotTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, api.VM_START_SNAPSHOT, "StartDiskSnapshot")
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

	guest.SetStatus(self.UserCred, api.VM_SNAPSHOT_SUCC, "")
	self.TaskComplete(ctx, guest, nil)
}

func (self *GuestDiskSnapshotTask) OnDiskSnapshotCompleteFailed(ctx context.Context, guest *models.SGuest, err jsonutils.JSONObject) {
	self.TaskFailed(ctx, guest, err.String())
}

func (self *GuestDiskSnapshotTask) TaskComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_DISK_CREATE_SNAPSHOT, nil, self.UserCred, true)
	guest.StartSyncstatus(ctx, self.UserCred, "")
}

func (self *GuestDiskSnapshotTask) TaskFailed(ctx context.Context, guest *models.SGuest, reason string) {
	guest.SetStatus(self.UserCred, api.VM_SNAPSHOT_FAILED, reason)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_DISK_CREATE_SNAPSHOT, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}
