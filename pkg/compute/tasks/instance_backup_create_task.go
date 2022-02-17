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
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type InstanceBackupCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(InstanceBackupCreateTask{})
}

func (self *InstanceBackupCreateTask) taskFailed(ctx context.Context, ib *models.SInstanceBackup, guest *models.SGuest, reason jsonutils.JSONObject) {
	if guest != nil {
		guest.SetStatus(self.UserCred, compute.VM_INSTANCE_BACKUP_FAILED, reason.String())
	}
	reasonStr, _ := reason.GetString()
	ib.SetStatus(self.UserCred, compute.INSTANCE_BACKUP_STATUS_CREATE_FAILED, reasonStr)
	logclient.AddActionLogWithStartable(self, ib, logclient.ACT_CREATE, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *InstanceBackupCreateTask) taskSuccess(ctx context.Context, ib *models.SInstanceBackup) {
	ib.SetStatus(self.UserCred, compute.INSTANCE_BACKUP_STATUS_READY, "")
	logclient.AddActionLogWithStartable(self, ib, logclient.ACT_CREATE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *InstanceBackupCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	ib := obj.(*models.SInstanceBackup)
	self.SetStage("OnInstanceBackup", nil)
	guest := models.GuestManager.FetchGuestById(ib.GuestId)
	params := jsonutils.NewDict()
	if err := ib.GetRegionDriver().RequestCreateInstanceBackup(ctx, guest, ib, self, params); err != nil {
		self.taskFailed(ctx, ib, guest, jsonutils.NewString(err.Error()))
	}
}

func (self *InstanceBackupCreateTask) OnKvmDisksSnapshot(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	subTasks := taskman.SubTaskManager.GetTotalSubtasks(self.Id, "OnKvmDisksSnapshot", "")
	guest := models.GuestManager.FetchGuestById(ib.GuestId)
	self.SetStage("OnInstanceBackup", nil)
	for i := range subTasks {
		log.Infof("subsTask %s result: %s", subTasks[i].SubtaskId, subTasks[i].Result)
		result, err := jsonutils.ParseString(subTasks[i].Result)
		if err != nil {
			self.taskFailed(ctx, ib, guest, jsonutils.NewString(fmt.Sprintf("unable to parse %s", subTasks[i].Result)))
			return
		}
		if subTasks[i].Status == taskman.SUBTASK_FAIL {
			self.taskFailed(ctx, ib, guest, result)
			return
		}
		snapshotId, _ := result.GetString("snapshot_id")
		diskBakcupId, _ := result.GetString("disk_backup_id")
		ibackup, err := models.DiskBackupManager.FetchById(diskBakcupId)
		if err != nil {
			self.taskFailed(ctx, ib, guest, jsonutils.NewString(err.Error()))
			return
		}
		backup := ibackup.(*models.SDiskBackup)
		params := jsonutils.NewDict()
		params.Set("snapshot_id", jsonutils.NewString(snapshotId))
		if err := backup.StartBackupCreateTask(ctx, self.UserCred, params, self.Id); err != nil {
			self.taskFailed(ctx, ib, guest, jsonutils.NewString(err.Error()))
			return
		}
	}
	guest.StartSyncstatus(ctx, self.UserCred, "")
}

func (self *InstanceBackupCreateTask) OnKvmDisksSnapshotFailed(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	self.taskFailed(ctx, ib, nil, data)
}

func (self *InstanceBackupCreateTask) OnInstanceBackup(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	subTasks := taskman.SubTaskManager.GetTotalSubtasks(self.Id, "OnInstanceBackup", "")
	for i := range subTasks {
		if subTasks[i].Status == taskman.SUBTASK_SUCC {
			continue
		}
		result, err := jsonutils.ParseString(subTasks[i].Result)
		if err != nil {
			self.taskFailed(ctx, ib, nil, jsonutils.NewString(fmt.Sprintf("unable to parse %s", subTasks[i].Result)))
			return
		}
		self.taskFailed(ctx, ib, nil, result)
	}
	// update size_mb
	backups, err := ib.GetBackups()
	if err != nil {
		self.taskFailed(ctx, ib, nil, jsonutils.NewString(err.Error()))
		return
	}
	var sizeMb int
	for i := range backups {
		sizeMb += backups[i].SizeMb
	}
	db.Update(ib, func() error {
		ib.SizeMb = sizeMb
		return nil
	})
	self.taskSuccess(ctx, ib)
}

func (self *InstanceBackupCreateTask) OnInstanceBackupFailed(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	self.taskFailed(ctx, ib, nil, data)
}
