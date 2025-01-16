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
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SnapshotCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SnapshotCreateTask{})
	taskman.RegisterTask(GuestDisksSnapshotPolicyExecuteTask{})
}

func (self *SnapshotCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	snapshot := obj.(*models.SSnapshot)
	self.DoDiskSnapshot(ctx, snapshot)
}

func (self *SnapshotCreateTask) TaskFailed(ctx context.Context, snapshot *models.SSnapshot, reason jsonutils.JSONObject) {
	snapshot.SetStatus(ctx, self.UserCred, api.SNAPSHOT_FAILED, reason.String())
	db.OpsLog.LogEvent(snapshot, db.ACT_SNAPSHOT_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, snapshot, logclient.ACT_CREATE, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *SnapshotCreateTask) TaskComplete(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	snapshot.SetStatus(ctx, self.UserCred, api.SNAPSHOT_READY, "")
	db.OpsLog.LogEvent(snapshot, db.ACT_SNAPSHOT_DONE, snapshot.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, snapshot, logclient.ACT_CREATE, snapshot.GetShortDesc(ctx), self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    snapshot,
		Action: notifyclient.ActionCreate,
	})
	self.SetStageComplete(ctx, nil)
}

func (self *SnapshotCreateTask) DoDiskSnapshot(ctx context.Context, snapshot *models.SSnapshot) {
	self.SetStage("OnCreateSnapshot", nil)
	if err := snapshot.GetRegionDriver().RequestCreateSnapshot(ctx, snapshot, self); err != nil {
		self.TaskFailed(ctx, snapshot, jsonutils.NewString(err.Error()))
	}
}

func (self *SnapshotCreateTask) OnCreateSnapshot(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	self.TaskComplete(ctx, snapshot, nil)
}

func (self *SnapshotCreateTask) OnCreateSnapshotFailed(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, snapshot, data)
}

type GuestDisksSnapshotPolicyExecuteTask struct {
	taskman.STask
}

func (self *GuestDisksSnapshotPolicyExecuteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.OnDiskSnapshot(ctx, obj, data)
}

func (self *GuestDisksSnapshotPolicyExecuteTask) OnDiskSnapshot(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	snapshotPolicyDisks := make([]models.SSnapshotPolicyDisk, 0)
	self.Params.Unmarshal(&snapshotPolicyDisks, "snapshot_policy_disks")
	if len(snapshotPolicyDisks) == 0 {
		self.SetStageComplete(ctx, nil)
		return
	}
	snapshotPolicyDisk := snapshotPolicyDisks[0]
	self.Params.Set("snapshot_policy_disks", jsonutils.Marshal(snapshotPolicyDisks[1:]))
	self.SetStage("OnDiskSnapshot", nil)

	disk, err := snapshotPolicyDisk.GetDisk()
	if err != nil {
		log.Errorf("disk snapshot policy %s failed get disk %s", snapshotPolicyDisk.SnapshotpolicyId, err)
		self.OnDiskSnapshot(ctx, obj, data)
		return
	}
	err = models.DiskManager.DoAutoSnapshot(ctx, self.UserCred, &snapshotPolicyDisk, disk, self.GetTaskId())
	if err != nil {
		log.Errorf("disk.CreateSnapshotAuto failed %s %s", disk.Id, err)
		db.OpsLog.LogEvent(disk, db.ACT_DISK_AUTO_SNAPSHOT_FAIL, err.Error(), self.UserCred)
		notifyclient.NotifySystemErrorWithCtx(ctx, disk.Id, disk.Name, db.ACT_DISK_AUTO_SNAPSHOT_FAIL, errors.Wrapf(err, "Disk auto create snapshot").Error())

		self.OnDiskSnapshot(ctx, obj, data)
		return
	}
}

func (self *GuestDisksSnapshotPolicyExecuteTask) OnDiskSnapshotFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	log.Errorf("Guest create snapshot failed %s: %s", obj.GetId(), data)
	self.OnDiskSnapshot(ctx, obj, data)
}
