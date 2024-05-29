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

type InstanceBackupSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(InstanceBackupSyncstatusTask{})
}

func (self *InstanceBackupSyncstatusTask) taskFailed(ctx context.Context, ib *models.SInstanceBackup, err jsonutils.JSONObject) {
	logclient.AddActionLogWithContext(ctx, ib, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
	self.SetStageFailed(ctx, err)
}

func (self *InstanceBackupSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	ib := obj.(*models.SInstanceBackup)
	self.SetStage("OnInstnaceBackupSyncstatus", nil)
	rd := ib.GetRegionDriver()
	err := rd.RequestSyncInstanceBackupStatus(ctx, self.GetUserCred(), ib, self)
	if err != nil {
		self.taskFailed(ctx, ib, jsonutils.NewString(err.Error()))
		return
	}
	return
}

func (self *InstanceBackupSyncstatusTask) OnKvmBackupSyncstatus(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	subTasks := taskman.SubTaskManager.GetSubtasks(self.Id, "OnKvmDisksSnapshot", "")
	for i := range subTasks {
		log.Infof("subsTask %s result: %s", subTasks[i].SubtaskId, subTasks[i].Result)
		result, err := jsonutils.ParseString(subTasks[i].Result)
		if err != nil {
			self.taskFailed(ctx, ib, jsonutils.NewString(fmt.Sprintf("unable to parse %s", subTasks[i].Result)))
			return
		}
		if subTasks[i].Status == taskman.SUBTASK_FAIL {
			self.taskFailed(ctx, ib, result)
			return
		}
	}
	backups, err := ib.GetBackups()
	if err != nil {
		self.taskFailed(ctx, ib, jsonutils.NewString(err.Error()))
		return
	}
	var status string
	for i := range backups {
		if backups[i].Status == compute.BACKUP_STATUS_UNKNOWN {
			status = compute.BACKUP_STATUS_UNKNOWN
			break
		}
		if backups[i].Status != compute.BACKUP_STATUS_READY {
			status = ""
			break
		}
		status = compute.BACKUP_STATUS_READY
	}
	if status == "" {
		originStatus, _ := self.Params.GetString("origin_status")
		ib.SetStatus(ctx, self.GetUserCred(), originStatus, "")
	} else {
		ib.SetStatus(ctx, self.GetUserCred(), status, "")
	}
	self.OnInstnaceBackupSyncstatus(ctx, ib, data)
	return
}

func (self *InstanceBackupSyncstatusTask) OnKvmBackupSyncstatusFailed(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	self.taskFailed(ctx, ib, data)
	return
}

func (self *InstanceBackupSyncstatusTask) OnInstnaceBackupSyncstatus(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
	return
}

func (self *InstanceBackupSyncstatusTask) OnInstnaceBackupSyncstatusFailed(ctx context.Context, ib *models.SInstanceBackup, data jsonutils.JSONObject) {
	self.taskFailed(ctx, ib, data)
	return
}
