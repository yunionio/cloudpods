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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type GuestDetachAllDisksTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestDetachAllDisksTask{})
}

func (self *GuestDetachAllDisksTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStage("OnDiskDeleteComplete", nil)
	self.OnDiskDeleteComplete(ctx, obj, data)
}

func (self *GuestDetachAllDisksTask) OnDiskDeleteComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	cnt, err := guest.DiskCount()
	if err != nil {
		self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
	if cnt == 0 {
		self.SetStageComplete(ctx, nil)
		return
	}
	host, _ := guest.GetHost()
	purge := false
	if (host == nil || !host.GetEnabled()) && jsonutils.QueryBoolean(self.Params, "purge", false) {
		purge = true
	}
	guestDisks, _ := guest.GetGuestDisks()
	for _, guestdisk := range guestDisks {
		taskData := jsonutils.NewDict()
		taskData.Add(jsonutils.NewString(guestdisk.DiskId), "disk_id")
		if purge {
			taskData.Add(jsonutils.JSONTrue, "purge")
		}
		if jsonutils.QueryBoolean(self.Params, "override_pending_delete", false) {
			taskData.Add(jsonutils.JSONTrue, "override_pending_delete")
		}
		taskData.Add(jsonutils.JSONFalse, "keep_disk")
		task, err := taskman.TaskManager.NewTask(ctx, "GuestDetachDiskTask", guest, self.UserCred, taskData, self.GetTaskId(), "", nil)
		if err != nil {
			self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		} else {
			task.ScheduleRun(nil)
		}
		break
	}
}
