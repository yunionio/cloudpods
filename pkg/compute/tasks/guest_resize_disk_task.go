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

type GuestResizeDiskTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestResizeDiskTask{})
}

func (task *GuestResizeDiskTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	guest.SetStatus(task.GetUserCred(), api.VM_RESIZE_DISK, "")
	db.OpsLog.LogEvent(guest, db.ACT_RESIZING, task.Params, task.UserCred)

	diskId, _ := task.Params.GetString("disk_id")
	sizeMb, _ := task.Params.Int("size")

	diskObj, err := models.DiskManager.FetchById(diskId)
	if err != nil {
		task.OnTaskFailed(ctx, guest, err.Error())
		return
	}

	pendingUsage := models.SQuota{}
	err = task.GetPendingUsage(&pendingUsage, 0)
	if err != nil {
		task.OnTaskFailed(ctx, guest, err.Error())
		return
	}

	task.SetStage("OnDiskResizeComplete", nil)

	diskObj.(*models.SDisk).StartDiskResizeTask(ctx, task.UserCred, sizeMb, task.GetId(), &pendingUsage)
}

func (task *GuestResizeDiskTask) OnTaskFailed(ctx context.Context, guest *models.SGuest, reason string) {
	log.Errorf("GuestResizeDiskTask fail: %s", reason)
	guest.SetStatus(task.UserCred, api.VM_RESIZE_DISK_FAILED, reason)
	db.OpsLog.LogEvent(guest, db.ACT_RESIZE_FAIL, reason, task.UserCred)
	logclient.AddActionLogWithStartable(task, guest, logclient.ACT_RESIZE, reason, task.UserCred, false)
	task.SetStageFailed(ctx, reason)
}

func (task *GuestResizeDiskTask) OnDiskResizeComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_RESIZE, task.Params, task.UserCred)
	logclient.AddActionLogWithStartable(task, guest, logclient.ACT_RESIZE, task.Params, task.UserCred, true)
	task.SetStage("TaskComplete", nil)
	if task.HasParentTask() {
		guest.StartSyncTaskWithoutSyncstatus(ctx, task.UserCred, false, task.GetId())
	} else {
		guest.StartSyncTask(ctx, task.UserCred, false, task.GetId())
	}
}

func (task *GuestResizeDiskTask) OnDiskResizeCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	task.OnTaskFailed(ctx, guest, data.String())
}

func (task *GuestResizeDiskTask) TaskComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	task.SetStageComplete(ctx, guest.GetShortDesc(ctx))
}

func (task *GuestResizeDiskTask) TaskCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	task.SetStageFailed(ctx, data.String())
}
