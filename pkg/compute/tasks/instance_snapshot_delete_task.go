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
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
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
	ctx context.Context, isp *models.SInstanceSnapshot, reason jsonutils.JSONObject) {

	isp.SetStatus(ctx, self.UserCred, compute.INSTANCE_SNAPSHOT_DELETE_FAILED, "on delete failed")
	db.OpsLog.LogEvent(isp, db.ACT_DELETE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithContext(ctx, isp, logclient.ACT_DELETE, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *InstanceSnapshotDeleteTask) taskComplete(
	ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {

	isp.RealDelete(ctx, self.UserCred)
	logclient.AddActionLogWithContext(ctx, isp, logclient.ACT_DELETE, nil, self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    isp,
		Action: notifyclient.ActionDelete,
	})
	self.SetStageComplete(ctx, nil)
}

func (self *InstanceSnapshotDeleteTask) OnInit(
	ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {

	isp := obj.(*models.SInstanceSnapshot)
	sps, err := isp.GetSnapshots()
	if err != nil {
		self.taskFail(ctx, isp, jsonutils.NewString(err.Error()))
		return
	}
	snapshotCnt := len(sps)
	self.Params.Set("snapshot_total_count", jsonutils.NewInt(int64(snapshotCnt)))
	self.SetStage("OnInstanceSnapshotDelete", nil)
	if err := isp.GetRegionDriver().RequestDeleteInstanceSnapshot(ctx, isp, self); err != nil {
		self.taskFail(ctx, isp, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *InstanceSnapshotDeleteTask) OnKvmSnapshotDelete(
	ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {
	snapshotId, _ := self.Params.GetString("del_snapshot_id")
	// detach snapshot and instance
	isjp := new(models.SInstanceSnapshotJoint)
	err := models.InstanceSnapshotJointManager.Query().
		Equals("instance_snapshot_id", isp.Id).Equals("snapshot_id", snapshotId).First(isjp)
	if err != nil {
		self.taskFail(ctx, isp, jsonutils.NewString(err.Error()))
		return
	}
	isjp.SetModelManager(models.InstanceSnapshotJointManager, isjp)
	err = isjp.Delete(ctx, self.UserCred)
	if err != nil {
		self.taskFail(ctx, isp, jsonutils.NewString(err.Error()))
		return
	}

	if err := isp.GetRegionDriver().RequestDeleteInstanceSnapshot(ctx, isp, self); err != nil {
		self.taskFail(ctx, isp, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *InstanceSnapshotDeleteTask) OnKvmSnapshotDeleteFailed(
	ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {
	self.taskFail(ctx, isp, data)
}

func (self *InstanceSnapshotDeleteTask) OnInstanceSnapshotDelete(ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {
	err := isp.Delete(ctx, self.UserCred)
	if err != nil {
		self.taskFail(ctx, isp, jsonutils.NewString(err.Error()))
		return
	}
	self.taskComplete(ctx, isp, data)

}

func (self *InstanceSnapshotDeleteTask) OnInstanceSnapshotDeleteFailed(ctx context.Context, isp *models.SInstanceSnapshot, data jsonutils.JSONObject) {
	self.taskFail(ctx, isp, data)
}
