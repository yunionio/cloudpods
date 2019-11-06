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

type GuestSaveGuestImageTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestSaveGuestImageTask{})
}

func (self *GuestSaveGuestImageTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	// prepare save image
	guest := obj.(*models.SGuest)

	self.SetStage("OnSaveRootImageComplete", nil)
	disks := guest.CategorizeDisks()
	imageIds, _ := self.Params.GetArray("image_ids")
	self.Params.Remove("image_ids")

	// data disk
	for index, dataDisk := range disks.Data {
		params := jsonutils.DeepCopy(self.Params).(*jsonutils.JSONDict)
		params.Add(imageIds[index], "image_id")
		if err := dataDisk.StartDiskSaveTask(ctx, self.UserCred, params, self.GetTaskId()); err != nil {
			self.taskFailed(ctx, guest, err.Error())
		}
	}

	self.Params.Add(imageIds[len(imageIds)-1], "image_id")
	if err := disks.Root.StartDiskSaveTask(ctx, self.UserCred, self.Params, self.GetTaskId()); err != nil {
		self.taskFailed(ctx, guest, err.Error())
	}
}

func (self *GuestSaveGuestImageTask) OnSaveRootImageComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	subTasks := taskman.SubTaskManager.GetTotalSubtasks(self.Id, "on_save_root_image_complete", taskman.SUBTASK_FAIL)

	if len(subTasks) > 0 {
		self.taskFailed(ctx, guest, "subtask failed")
	}

	if restart, _ := self.GetParams().Bool("auto_start"); restart {
		self.SetStage("on_start_server_complete", nil)
		guest.StartGueststartTask(ctx, self.GetUserCred(), nil, self.GetTaskId())
	} else {
		guest.SetStatus(self.UserCred, api.VM_READY, "")
		self.taskSuc(ctx, guest)
	}
}

func (self *GuestSaveGuestImageTask) OnSaveRootImageCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	log.Errorf("Guest save image failed: %s", data.PrettyString())
	self.taskFailed(ctx, guest, "")
}

func (self *GuestSaveGuestImageTask) OnStartServerComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.taskSuc(ctx, guest)
}

func (self *GuestSaveGuestImageTask) OnStartServerCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	// even if start server failed, the task that save guest image is successful
	self.taskSuc(ctx, guest)
}

func (self *GuestSaveGuestImageTask) taskSuc(ctx context.Context, guest *models.SGuest) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestSaveGuestImageTask) taskFailed(ctx context.Context, guest *models.SGuest, reason string) {

	guest.SetStatus(self.UserCred, api.VM_SAVE_DISK_FAILED, reason)
	db.OpsLog.LogEvent(guest, db.ACT_GUEST_SAVE_GUEST_IMAGE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_IMAGE_SAVE, reason, self.UserCred, false)

	self.SetStageFailed(ctx, reason)
}
