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

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SnapshotPolicyCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SnapshotPolicyCreateTask{})
	taskman.RegisterTask(SnapshotPolicyApplyTask{})
	taskman.RegisterTask(SnapshotPolicyCancelTask{})
}

func (self *SnapshotPolicyCreateTask) taskFail(ctx context.Context, sp *models.SSnapshotPolicy, reason string) {
	sp.SetStatus(self.UserCred, compute.SNAPSHOT_POLICY_CREATE_FAILED, "")
	db.OpsLog.LogEvent(sp, db.ACT_ALLOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, sp, logclient.ACT_CREATE, false, self.UserCred, false)
	notifyclient.NotifySystemError(sp.GetId(), sp.Name, compute.SNAPSHOT_POLICY_CREATE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *SnapshotPolicyCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	snapshotPolicy := obj.(*models.SSnapshotPolicy)

	region := snapshotPolicy.GetRegion()
	if region == nil {
		self.taskFail(ctx, snapshotPolicy, fmt.Sprintf("failed to find region for snapshot policy %s", snapshotPolicy.Name))
		return
	}
	self.SetStage("OnSnapshotPolicyCreate", nil)
	if err := region.GetDriver().RequestCreateSnapshotPolicy(ctx, self.GetUserCred(), snapshotPolicy, self); err != nil {
		self.taskFail(ctx, snapshotPolicy, err.Error())
	}
}

func (self *SnapshotPolicyCreateTask) OnSnapshotPolicyCreate(
	ctx context.Context, sp *models.SSnapshotPolicy, data jsonutils.JSONObject,
) {
	sp.SetStatus(self.UserCred, compute.SNAPSHOT_POLICY_READY, "")
	db.OpsLog.LogEvent(sp, db.ACT_ALLOCATE, sp.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, sp, logclient.ACT_CREATE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *SnapshotPolicyCreateTask) OnSnapshotPolicyCreateFailed(
	ctx context.Context, sp *models.SSnapshotPolicy, data jsonutils.JSONObject,
) {
	self.taskFail(ctx, sp, data.String())
}

// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-

type SnapshotPolicyApplyTask struct {
	taskman.STask
}

func (self *SnapshotPolicyApplyTask) taskFail(ctx context.Context, disk *models.SDisk, snapshotPolicyId, reason string) {
	jointModel, err := db.FetchJointByIds(models.SnapshotPolicyDiskManager, disk.Id, snapshotPolicyId, jsonutils.JSONNull)
	if err != nil {
		log.Errorf("Fetch SnapshotPolicy %s Disk %s joint model failed %s", disk.Id, snapshotPolicyId, reason)
		return
	}
	snapshotPolicyDisk := jointModel.(*models.SSnapshotPolicyDisk)
	err = snapshotPolicyDisk.Detach(ctx, self.UserCred)
	if err != nil {
		log.Errorf("Delete SnapshotPolicy %s Disk %s joint model failed, need to delete", self.Id, snapshotPolicyId)
	}

	disk.SetStatus(self.UserCred, compute.DISK_APPLY_SNAPSHOT_FAIL, reason)
	db.OpsLog.LogEvent(disk, db.ACT_APPLY_SNAPSHOT_POLICY_FAILED, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, disk, logclient.ACT_APPLY_SNAPSHOT_POLICY, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *SnapshotPolicyApplyTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)
	snapshotPolicyID, _ := self.Params.GetString("snapshot_policy_id")

	// fetch disk model by diksID
	model, err := models.SnapshotPolicyManager.FetchById(snapshotPolicyID)
	if err != nil {
		self.taskFail(ctx, disk, snapshotPolicyID, fmt.Sprintf("failed to fetch disk by id %s: %s", snapshotPolicyID, err.Error()))
		return
	}
	snapshotPolicy := model.(*models.SSnapshotPolicy)

	self.SetStage("OnSnapshotPolicyApply", nil)
	if err := disk.GetStorage().GetRegion().GetDriver().
		RequestApplySnapshotPolicy(ctx, self.UserCred, snapshotPolicy, self, disk.ExternalId); err != nil {
		self.taskFail(ctx, disk, snapshotPolicyID, fmt.Sprintf("faile to attach snapshot policy %s and disk %s: %s", snapshotPolicy.Id, disk.Id, err.Error()))
	}

}

func (self *SnapshotPolicyApplyTask) OnSnapshotPolicyApply(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(disk, db.ACT_APPLY_SNAPSHOT_POLICY, "", self.UserCred)
	logclient.AddActionLogWithStartable(self, disk, logclient.ACT_APPLY_SNAPSHOT_POLICY, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-

type SnapshotPolicyCancelTask struct {
	taskman.STask
}

func (self *SnapshotPolicyCancelTask) taskFail(ctx context.Context, disk *models.SDisk, snapshotPolicyId, reason string) {
	jointModel, err := db.FetchJointByIds(models.SnapshotPolicyDiskManager, disk.Id, snapshotPolicyId, jsonutils.JSONNull)
	if err != nil {
		log.Errorf("Fetch SnapshotPolicy %s Disk %s joint model failed %s", disk.Id, snapshotPolicyId, reason)
		return
	}
	snapshotPolicyDisk := jointModel.(*models.SSnapshotPolicyDisk)
	err = snapshotPolicyDisk.MarkUnDelete()
	if err != nil {
		log.Errorf("Mark undelete joint model snapshotPolicy %s and disk %s failes", snapshotPolicyId, disk.Id)
	}
	disk.SetStatus(self.UserCred, compute.DISK_CALCEL_SNAPSHOT_FAIL, reason)
	db.OpsLog.LogEvent(disk, db.ACT_CANCEL_SNAPSHOT_POLICY_FAILED, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, disk, logclient.ACT_CANCEL_SNAPSHOT_POLICY, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *SnapshotPolicyCancelTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)
	snapshotPolicyID, _ := self.GetParams().GetString("snapshot_policy_id")

	model, err := models.SnapshotPolicyManager.FetchById(snapshotPolicyID)
	if err != nil {
		self.taskFail(ctx, disk, snapshotPolicyID, fmt.Sprintf("failed to fetch disk by id %s: %s", snapshotPolicyID, err.Error()))
		return
	}
	snapshotPolicy := model.(*models.SSnapshotPolicy)
	self.SetStage("OnSnapshotPolicyCancel", nil)
	if err := disk.GetStorage().GetRegion().GetDriver().
		RequestCancelSnapshotPolicy(ctx, self.UserCred, snapshotPolicy, self, disk.ExternalId); err != nil {
		self.taskFail(ctx, disk, snapshotPolicyID, fmt.Sprintf("faile to detach snapshot policy %s and disk %s: %s", snapshotPolicy.Id, disk.Id, err.Error()))
	}
}

func (self *SnapshotPolicyCancelTask) OnSnapshotPolicyCancel(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(disk, db.ACT_CANCEL_SNAPSHOT_POLICY, "", self.UserCred)
	logclient.AddActionLogWithStartable(self, disk, logclient.ACT_CANCEL_SNAPSHOT_POLICY, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
