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
)

type GuestSaveImageTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestSaveImageTask{})
}

func (self *GuestSaveImageTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	log.Infof("Saving server image: %s", guest.Name)
	if restart, _ := self.GetParams().Bool("restart"); restart {
		self.SetStage("on_stop_server_complete", nil)
		guest.StartGuestStopTask(ctx, self.GetUserCred(), false, self.GetTaskId())
	} else {
		self.OnStopServerComplete(ctx, guest, nil)
	}
}

func (self *GuestSaveImageTask) OnStopServerComplete(ctx context.Context, guest *models.SGuest, body jsonutils.JSONObject) {
	self.SetStage("on_save_root_image_complete", nil)
	disks := guest.CategorizeDisks()
	if err := disks.Root.StartDiskSaveTask(ctx, self.GetUserCred(), self.GetParams(), self.GetTaskId()); err != nil {
		self.SetStageFailed(ctx, err.Error())
	}
}

func (self *GuestSaveImageTask) OnSaveRootImageComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if restart, _ := self.GetParams().Bool("restart"); restart {
		self.SetStage("on_start_server_complete", nil)
		guest.StartGueststartTask(ctx, self.GetUserCred(), nil, self.GetTaskId())
	} else {
		self.SetStageComplete(ctx, nil)
	}
}

func (self *GuestSaveImageTask) OnSaveRootImageCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	log.Errorf("Guest save root image failed: %s", data.PrettyString())
	guest.SetStatus(self.GetUserCred(), api.VM_SAVE_DISK_FAILED, data.PrettyString())
	self.SetStageFailed(ctx, data.PrettyString())
}

func (self *GuestSaveImageTask) OnStartServerComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestSaveImageTask) OnStartServerCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}
