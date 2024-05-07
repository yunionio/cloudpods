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

type GuestSaveImageTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestSaveImageTask{})
}

func (self *GuestSaveImageTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	log.Infof("Saving server image: %s", guest.Name)
	restart := jsonutils.QueryBoolean(self.Params, "restart", false)
	if restart && guest.Status != api.VM_READY {
		self.SetStage("OnStopServerComplete", nil)
		guest.StartGuestStopTask(ctx, self.GetUserCred(), 60, false, false, self.GetTaskId())
		return
	}
	self.OnStopServerComplete(ctx, guest, nil)
}

func (self *GuestSaveImageTask) OnStopServerComplete(ctx context.Context, guest *models.SGuest, body jsonutils.JSONObject) {
	self.SetStage("OnSaveRootImageComplete", nil)
	drv, err := guest.GetDriver()
	if err != nil {
		self.OnSaveRootImageCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	err = drv.RequestSaveImage(ctx, self.GetUserCred(), guest, self)
	if err != nil {
		self.OnSaveRootImageCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *GuestSaveImageTask) OnStopServerCompleteFailed(ctx context.Context, guest *models.SGuest, body jsonutils.JSONObject) {
	guest.SetStatus(ctx, self.GetUserCred(), api.VM_SAVE_DISK_FAILED, body.String())
	self.SetStageFailed(ctx, nil)
}

func (self *GuestSaveImageTask) OnSaveRootImageComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	restart := jsonutils.QueryBoolean(self.Params, "restart", false)
	autoStart := jsonutils.QueryBoolean(self.Params, "auto_start", false)
	if restart || autoStart {
		self.SetStage("OnStartServerComplete", nil)
		guest.StartGueststartTask(ctx, self.GetUserCred(), nil, self.GetTaskId())
		return
	}
	self.SetStage("OnSyncstatusComplete", nil)
	guest.StartSyncstatus(ctx, self.GetUserCred(), self.GetTaskId())
}

func (self *GuestSaveImageTask) OnSaveRootImageCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_SAVE_IMAGE, data, self.UserCred, false)
	guest.SetStatus(ctx, self.GetUserCred(), api.VM_SAVE_DISK_FAILED, data.String())
	self.SetStageFailed(ctx, data)
}

func (self *GuestSaveImageTask) OnStartServerComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestSaveImageTask) OnStartServerCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, nil)
}

func (self *GuestSaveImageTask) OnSyncstatusComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestSaveImageTask) OnSyncstatusCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, nil)
}
