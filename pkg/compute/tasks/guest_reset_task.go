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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

func init() {
	taskman.RegisterTask(GuestSoftResetTask{})
	taskman.RegisterTask(GuestHardResetTask{})
	taskman.RegisterTask(GuestRestartTask{})
}

type GuestSoftResetTask struct {
	SGuestBaseTask
}

func (self *GuestSoftResetTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	drv, err := guest.GetDriver()
	if err != nil {
		self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
	err = drv.RequestSoftReset(ctx, guest, self)
	if err != nil {
		self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
	self.SetStageComplete(ctx, nil)
}

type GuestHardResetTask struct {
	SGuestBaseTask
}

func (self *GuestHardResetTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.StopServer(ctx, guest)
}

func (self *GuestHardResetTask) StopServer(ctx context.Context, guest *models.SGuest) {
	guest.SetStatus(ctx, self.UserCred, api.VM_STOPPING, "")
	self.SetStage("OnServerStopComplete", nil)
	guest.StartGuestStopTask(ctx, self.UserCred, 30, false, false, self.GetTaskId())
	// logclient.AddActionLogWith(guest, logclient.ACT_VM_RESTART, `{"is_force": true}`, self.UserCred, true)
}

func (self *GuestHardResetTask) OnServerStopComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.StartServer(ctx, guest)
}

func (self *GuestHardResetTask) StartServer(ctx context.Context, guest *models.SGuest) {
	self.SetStage("OnServerStartComplete", nil)
	guest.StartGueststartTask(ctx, self.UserCred, nil, self.GetTaskId())
}

func (self *GuestHardResetTask) OnServerStartComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_RESTART, nil, self.GetUserCred(), true)
	self.SetStageComplete(ctx, nil)
}

type GuestRestartTask struct {
	GuestHardResetTask
}

func (self *GuestRestartTask) StopServer(ctx context.Context, guest *models.SGuest) {
	self.SetStage("OnServerStopComplete", nil)
	isForce := jsonutils.QueryBoolean(self.Params, "is_force", false)
	guest.StartGuestStopTask(ctx, self.UserCred, 60, isForce, false, self.GetTaskId())
	// logclient.AddActionLog(guest, logclient.ACT_VM_RESTART, `{"is_force": false}`, self.UserCred, true)
}
