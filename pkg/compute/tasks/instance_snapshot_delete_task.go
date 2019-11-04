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

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type InstanceSnapshotDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(InstanceSnapshotDeleteTask{})
}

func (self *InstanceSnapshotDeleteTask) taskFail(
	ctx context.Context, isp *models.SInstanceSnapshot, reason string) {

	isp.SetStatus(self.UserCred, compute.INSTANCE_SNAPSHOT_DELETE_FAILED, "on delete failed")
	db.OpsLog.LogEvent(isp, db.ACT_DELETE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithContext(ctx, isp, logclient.ACT_DELETE, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *InstanceSnapshotDeleteTask) taskComplete(
	ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {

	isp.RealDelete(ctx, self.UserCred)
	logclient.AddActionLogWithContext(ctx, isp, logclient.ACT_DELETE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *InstanceSnapshotDeleteTask) OnInit(
	ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {

	isp := obj.(*models.SInstanceSnapshot)
	self.SetStage("OnSnapshotDelete", nil)
	self.StartSnapshotDelete(ctx, isp)
}

func (self *InstanceSnapshotDeleteTask) StartSnapshotDelete(
	ctx context.Context, isp *models.SInstanceSnapshot) {

	snapshots, err := isp.GetSnapshots()
	if err != nil {
		self.taskFail(ctx, isp, err.Error())
		return
	}
	if len(snapshots) == 0 {
		self.taskComplete(ctx, isp, nil)
		return
	}

	// detach snapshot and instance
	isjp := new(models.SInstanceSnapshotJoint)
	err = models.InstanceSnapshotJointManager.Query().
		Equals("instance_snapshot_id", isp.Id).Equals("snapshot_id", snapshots[0].Id).First(isjp)
	if err != nil {
		self.taskFail(ctx, isp, err.Error())
		return
	}
	isjp.SetModelManager(models.InstanceSnapshotJointManager, isjp)
	err = isjp.Delete(ctx, self.UserCred)
	if err != nil {
		self.taskFail(ctx, isp, err.Error())
		return
	}

	err = snapshots[0].StartSnapshotDeleteTask(ctx, self.UserCred, false, self.Id)
	if err != nil {
		self.taskFail(ctx, isp, err.Error())
		return
	}
}

func (self *InstanceSnapshotDeleteTask) OnSnapshotDelete(
	ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {

	self.StartSnapshotDelete(ctx, isp)
}

func (self *InstanceSnapshotDeleteTask) OnSnapshotDeleteFailed(
	ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {

	self.taskFail(ctx, isp, data.String())
}
