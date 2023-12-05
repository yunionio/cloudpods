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

func (tsk *InstanceBackupDeleteTask) taskFailed(ctx context.Context, ib *models.SInstanceBackup, reason jsonutils.JSONObject) {
	reasonStr, _ := reason.GetString()
	ib.SetStatus(tsk.UserCred, compute.INSTANCE_BACKUP_STATUS_DELETE_FAILED, reasonStr)
	logclient.AddActionLogWithStartable(tsk, ib, logclient.ACT_DELETE, reason, tsk.UserCred, false)
	tsk.SetStageFailed(ctx, reason)
}

func (tsk *InstanceBackupDeleteTask) taskSuccess(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	ib.RealDelete(ctx, tsk.UserCred)
	logclient.AddActionLogWithContext(ctx, ib, logclient.ACT_DELETE, ib.GetShortDesc(ctx), tsk.UserCred, true)
	tsk.SetStageComplete(ctx, nil)
}

func (tsk *InstanceBackupDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	ib := obj.(*models.SInstanceBackup)
	tsk.SetStage("OnInstanceBackupDelete", nil)
	if err := ib.GetRegionDriver().RequestDeleteInstanceBackup(ctx, ib, tsk); err != nil {
		tsk.taskFailed(ctx, ib, jsonutils.NewString(err.Error()))
		return
	}
}

func (tsk *InstanceBackupDeleteTask) OnKvmDiskBackupDelete(
	ctx context.Context, isp *models.SInstanceBackup, data jsonutils.JSONObject) {
	backupId, _ := tsk.Params.GetString("del_backup_id")
	// detach backup and instance
	isjp := new(models.SInstanceBackupJoint)
	isjp.SetModelManager(models.InstanceBackupJointManager, isjp)
	err := models.InstanceBackupJointManager.Query().
		Equals("instance_backup_id", isp.Id).Equals("disk_backup_id", backupId).First(isjp)
	if err != nil {
		tsk.taskFailed(ctx, isp, jsonutils.NewString(err.Error()))
		return
	}
	err = isjp.Delete(ctx, tsk.UserCred)
	if err != nil {
		tsk.taskFailed(ctx, isp, jsonutils.NewString(err.Error()))
		return
	}
	if err := isp.GetRegionDriver().RequestDeleteInstanceBackup(ctx, isp, tsk); err != nil {
		tsk.taskFailed(ctx, isp, jsonutils.NewString(err.Error()))
		return
	}
}

func (tsk *InstanceBackupDeleteTask) OnKvmDiskBackupDeleteFailed(
	ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	tsk.taskFailed(ctx, ib, data)
}

func (tsk *InstanceBackupDeleteTask) OnInstanceBackupDelete(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	tsk.taskSuccess(ctx, ib, data)
}

func (tsk *InstanceBackupDeleteTask) OnInstanceBackupDeleteFailed(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	tsk.taskFailed(ctx, ib, data)
}
