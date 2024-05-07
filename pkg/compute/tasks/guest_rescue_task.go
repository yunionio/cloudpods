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

type StartRescueTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(StartRescueTask{})
}

func (self *StartRescueTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	// Flow: stop -> modify startvm script for rescue -> start
	guest := obj.(*models.SGuest)
	// Check if guest is running
	if guest.Status == api.VM_RUNNING {
		self.StopServer(ctx, guest)
	} else {
		self.PrepareRescue(ctx, guest)
	}
}

func (self *StartRescueTask) StopServer(ctx context.Context, guest *models.SGuest) {
	db.OpsLog.LogEvent(guest, db.ACT_STOPPING, nil, self.UserCred)
	guest.SetStatus(ctx, self.UserCred, api.VM_STOPPING, "StopServer")
	self.SetStage("OnServerStopComplete", nil)
	guest.StartGuestStopTask(ctx, self.UserCred, 0, true, false, self.GetTaskId())
}

func (self *StartRescueTask) OnServerStopComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_STOP, guest.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_STOP, guest.GetShortDesc(ctx), self.UserCred, true)

	self.PrepareRescue(ctx, guest)
}

func (self *StartRescueTask) OnServerStopCompleteFailed(ctx context.Context, guest *models.SGuest, err jsonutils.JSONObject) {
	guest.SetStatus(ctx, self.UserCred, api.VM_STOP_FAILED, err.String())
	db.OpsLog.LogEvent(guest, db.ACT_STOP_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_STOP, err, self.UserCred, false)
	self.SetStageFailed(ctx, err)
}

func (self *StartRescueTask) PrepareRescue(ctx context.Context, guest *models.SGuest) {
	db.OpsLog.LogEvent(guest, db.ACT_START_RESCUE, nil, self.UserCred)
	guest.SetStatus(ctx, self.UserCred, api.VM_START_RESCUE, "PrepareRescue")

	host, _ := guest.GetHost()
	drv, err := guest.GetDriver()
	if err != nil {
		self.OnRescuePrepareCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	err = drv.RequestStartRescue(ctx, self, jsonutils.NewDict(), host, guest)
	if err != nil {
		self.OnRescuePrepareCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	self.OnRescuePrepareComplete(ctx, guest, nil)
}

func (self *StartRescueTask) OnRescuePrepareComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_START_RESCUE, guest.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_START_RESCUE, guest.GetShortDesc(ctx), self.UserCred, true)
	guest.UpdateRescueMode(true)
	self.RescueStartServer(ctx, guest)
}

func (self *StartRescueTask) OnRescuePrepareCompleteFailed(ctx context.Context, guest *models.SGuest, err jsonutils.JSONObject) {
	guest.SetStatus(ctx, self.UserCred, api.VM_START_RESCUE_FAILED, err.String())
	db.OpsLog.LogEvent(guest, db.ACT_STOP_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_STOP, err, self.UserCred, false)
	guest.UpdateRescueMode(false)
	self.SetStageFailed(ctx, err)
}

func (self *StartRescueTask) RescueStartServer(ctx context.Context, guest *models.SGuest) {
	guest.SetStatus(ctx, self.UserCred, api.VM_START_RESCUE, "RescueStartServer")
	self.SetStage("OnRescueStartServerComplete", nil)

	// Set Guest rescue params to guest start params
	host, _ := guest.GetHost()
	drv, err := guest.GetDriver()
	if err != nil {
		self.OnRescueStartServerCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	err = drv.RequestStartOnHost(ctx, guest, host, self.UserCred, self)
	if err != nil {
		self.OnRescueStartServerCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *StartRescueTask) OnRescueStartServerComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_START_RESCUE, guest.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_START_RESCUE, guest.GetShortDesc(ctx), self.UserCred, true)

	// Set guest status to rescue running
	guest.SetStatus(ctx, self.UserCred, api.VM_RESCUE, "OnRescueStartServerComplete")
	self.SetStageComplete(ctx, nil)
}

func (self *StartRescueTask) OnRescueStartServerCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(ctx, self.UserCred, api.VM_START_RESCUE_FAILED, err.String())
	db.OpsLog.LogEvent(guest, db.ACT_START_RESCUE_FAILED, guest.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_START, guest.GetShortDesc(ctx), self.UserCred, true)
}

type StopRescueTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(StopRescueTask{})
}

func (self *StopRescueTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	// Flow: stop -> modify startvm script for rescue -> start
	guest := obj.(*models.SGuest)
	// Check if guest is running
	if guest.Status == api.VM_RUNNING || guest.Status == api.VM_RESCUE {
		self.StopServer(ctx, guest)
	} else {
		self.ClearRescue(ctx, guest)
	}
}

func (self *StopRescueTask) StopServer(ctx context.Context, guest *models.SGuest) {
	db.OpsLog.LogEvent(guest, db.ACT_STOPPING, nil, self.UserCred)
	guest.SetStatus(ctx, self.UserCred, api.VM_STOPPING, "StopServer")
	self.SetStage("OnServerStopComplete", nil)
	guest.StartGuestStopTask(ctx, self.UserCred, 0, true, false, self.GetTaskId())
}

func (self *StopRescueTask) OnServerStopComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_STOP, guest.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_STOP, guest.GetShortDesc(ctx), self.UserCred, true)

	self.ClearRescue(ctx, guest)
}

func (self *StopRescueTask) OnServerStopCompleteFailed(ctx context.Context, guest *models.SGuest, err jsonutils.JSONObject) {
	guest.SetStatus(ctx, self.UserCred, api.VM_STOP_FAILED, err.String())
	db.OpsLog.LogEvent(guest, db.ACT_STOP_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_STOP, err, self.UserCred, false)
	self.SetStageFailed(ctx, err)
}

func (self *StopRescueTask) ClearRescue(ctx context.Context, guest *models.SGuest) {
	db.OpsLog.LogEvent(guest, db.ACT_STOP_RESCUE, nil, self.UserCred)
	guest.SetStatus(ctx, self.UserCred, api.VM_STOP_RESCUE, "ClearRescue")
	self.OnRescueClearComplete(ctx, guest, nil)
}

func (self *StopRescueTask) OnRescueClearComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_STOP_RESCUE, guest.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_STOP_RESCUE, guest.GetShortDesc(ctx), self.UserCred, true)
	if err := guest.UpdateRescueMode(false); err != nil {
		self.OnRescueClearCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	self.RescueStartServer(ctx, guest)
}

func (self *StopRescueTask) OnRescueClearCompleteFailed(ctx context.Context, guest *models.SGuest, err jsonutils.JSONObject) {
	guest.SetStatus(ctx, self.UserCred, api.VM_STOP_RESCUE_FAILED, err.String())
	db.OpsLog.LogEvent(guest, db.ACT_STOP_RESCUE_FAILED, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_STOP_RESCUE, err, self.UserCred, false)
	self.SetStageFailed(ctx, err)
}

func (self *StopRescueTask) RescueStartServer(ctx context.Context, guest *models.SGuest) {
	guest.SetStatus(ctx, self.UserCred, api.VM_STARTING, "RescueStartServer")
	self.SetStage("OnRescueStartServerComplete", nil)

	// Set Guest rescue params to guest start params
	host, _ := guest.GetHost()
	drv, err := guest.GetDriver()
	if err != nil {
		self.OnRescueStartServerCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	err = drv.RequestStartOnHost(ctx, guest, host, self.UserCred, self)
	if err != nil {
		self.OnRescueStartServerCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *StopRescueTask) OnRescueStartServerComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_START, guest.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_START, guest.GetShortDesc(ctx), self.UserCred, true)

	// Set guest status to rescue running
	guest.SetStatus(ctx, self.UserCred, api.VM_RUNNING, "OnRescueStartServerComplete")
	self.SetStageComplete(ctx, nil)
}

func (self *StopRescueTask) OnRescueStartServerCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(ctx, self.UserCred, api.VM_START_FAILED, err.String())
	db.OpsLog.LogEvent(guest, db.ACT_START_FAIL, guest.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_START, guest.GetShortDesc(ctx), self.UserCred, true)
}
