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

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

func init() {
	taskman.RegisterTask(GuestDeleteBackupTask{})
}

type GuestDeleteBackupTask struct {
	SGuestBaseTask
}

func (self *GuestDeleteBackupTask) OnFail(ctx context.Context, guest *models.SGuest, reason string) {
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_DELETE_BACKUP, reason, self.UserCred, false)
	db.OpsLog.LogEvent(guest, db.ACT_DELETE_BACKUP_FAILED, reason, self.UserCred)
	guest.SetStatus(self.UserCred, compute.VM_BACKUP_DELETE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *GuestDeleteBackupTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	host := models.HostManager.FetchHostById(guest.HostId)
	if host == nil {
		self.OnFail(ctx, guest, "Host not found")
		return
	}

	self.SetStage("OnCancelBlockJobs", nil)
	url := fmt.Sprintf("%s/servers/%s/cancel-block-jobs", host.ManagerUri, guest.Id)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(),
		ctx, "POST", url, self.GetTaskRequestHeader(), nil, false)
	if err != nil {
		self.OnFail(ctx, guest, err.Error())
		return
	}
}

func (self *GuestDeleteBackupTask) OnCancelBlockJobs(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.StartDeleteBackupOnHost(ctx, guest)
}

func (self *GuestDeleteBackupTask) StartDeleteBackupOnHost(ctx context.Context, guest *models.SGuest) {
	taskData := jsonutils.NewDict()
	taskData.Set("purge", jsonutils.NewBool(jsonutils.QueryBoolean(self.Params, "purge", false)))
	taskData.Set("host_id", jsonutils.NewString(guest.BackupHostId))
	taskData.Set("failed_status", jsonutils.NewString(compute.VM_BACKUP_DELETE_FAILED))
	if task, err := taskman.TaskManager.NewTask(
		ctx, "GuestDeleteOnHostTask", guest, self.UserCred, taskData, "", "", nil); err != nil {
		self.OnFail(ctx, guest, err.Error())
		return
	} else {
		task.ScheduleRun(nil)
		self.SetStageComplete(ctx, nil)
	}
}

func (self *GuestDeleteBackupTask) OnCancelBlockJobsFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.OnFail(ctx, guest, data.String())
}
