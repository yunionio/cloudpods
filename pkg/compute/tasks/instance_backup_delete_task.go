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

type InstanceBackupDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(InstanceBackupDeleteTask{})
}

func (self *InstanceBackupDeleteTask) taskFailed(ctx context.Context, ib *models.SInstanceBackup, reason jsonutils.JSONObject) {
	reasonStr, _ := reason.GetString()
	ib.SetStatus(ctx, self.UserCred, compute.INSTANCE_BACKUP_STATUS_DELETE_FAILED, reasonStr)
	logclient.AddActionLogWithStartable(self, ib, logclient.ACT_DELETE, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *InstanceBackupDeleteTask) taskSuccess(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	ib.RealDelete(ctx, self.UserCred)
	logclient.AddActionLogWithContext(ctx, ib, logclient.ACT_DELETE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *InstanceBackupDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	ib := obj.(*models.SInstanceBackup)
	self.SetStage("OnInstanceBackupDelete", nil)
	if err := ib.GetRegionDriver().RequestDeleteInstanceBackup(ctx, ib, self); err != nil {
		self.taskFailed(ctx, ib, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *InstanceBackupDeleteTask) OnKvmDiskBackupDelete(
	ctx context.Context, isp *models.SInstanceBackup, data jsonutils.JSONObject) {
	backupId, _ := self.Params.GetString("del_backup_id")
	// detach backup and instance
	isjp := new(models.SInstanceBackupJoint)
	isjp.SetModelManager(models.InstanceBackupJointManager, isjp)
	err := models.InstanceBackupJointManager.Query().
		Equals("instance_backup_id", isp.Id).Equals("disk_backup_id", backupId).First(isjp)
	if err != nil {
		self.taskFailed(ctx, isp, jsonutils.NewString(err.Error()))
		return
	}
	err = isjp.Delete(ctx, self.UserCred)
	if err != nil {
		self.taskFailed(ctx, isp, jsonutils.NewString(err.Error()))
		return
	}
	if err := isp.GetRegionDriver().RequestDeleteInstanceBackup(ctx, isp, self); err != nil {
		self.taskFailed(ctx, isp, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *InstanceBackupDeleteTask) OnKvmDiskBackupDeleteFailed(
	ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	self.taskFailed(ctx, ib, data)
}

func (self *InstanceBackupDeleteTask) OnInstanceBackupDelete(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	self.taskSuccess(ctx, ib, data)
}

func (self *InstanceBackupDeleteTask) OnInstanceBackupDeleteFailed(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	self.taskFailed(ctx, ib, data)
}
