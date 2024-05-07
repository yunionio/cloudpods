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
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestResetNicTrafficsTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestResetNicTrafficsTask{})
	taskman.RegisterTask(GuestSetNicTrafficsTask{})
}

func (self *GuestResetNicTrafficsTask) taskFailed(ctx context.Context, guest *models.SGuest, reason string) {
	guest.SetStatus(ctx, self.UserCred, compute.VM_SYNC_TRAFFIC_LIMIT, "PerformResetNicTrafficLimit")
	db.OpsLog.LogEvent(guest, db.ACT_SYNC_TRAFFIC_LIMIT_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_SYNC_TRAFFIC_LIMIT, reason, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(reason))
}

func (self *GuestResetNicTrafficsTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	host, err := guest.GetHost()
	if err != nil {
		self.taskFailed(ctx, guest, fmt.Sprintf("get host %s", err))
		return
	}
	input := compute.ServerNicTrafficLimit{}
	self.GetParams().Unmarshal(&input)
	self.SetStage("OnResetNicTrafficLimit", nil)
	drv, err := guest.GetDriver()
	if err != nil {
		self.taskFailed(ctx, guest, err.Error())
		return
	}
	err = drv.RequestResetNicTrafficLimit(ctx, self, host, guest, []compute.ServerNicTrafficLimit{input})
	if err != nil {
		self.taskFailed(ctx, guest, err.Error())
	}
}

func (self *GuestResetNicTrafficsTask) OnResetNicTrafficLimit(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	input := &compute.ServerNicTrafficLimit{}
	self.GetParams().Unmarshal(input)
	gn, _ := guest.GetGuestnetworkByMac(input.Mac)
	err := gn.UpdateNicTrafficLimit(input.RxTrafficLimit, input.TxTrafficLimit)
	if err != nil {
		self.taskFailed(ctx, guest, fmt.Sprintf("failed update guest nic traffic limit %s", err))
		return
	}
	err = gn.UpdateNicTrafficUsed(0, 0)
	if err != nil {
		self.taskFailed(ctx, guest, fmt.Sprintf("failed update guest nic traffic used %s", err))
		return
	}

	oldStatus, _ := self.Params.GetString("old_status")
	guest.SetStatus(ctx, self.UserCred, oldStatus, "OnResetNicTrafficLimit")
	db.OpsLog.LogEvent(guest, db.ACT_SYNC_TRAFFIC_LIMIT, "OnResetNicTrafficLimit", self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_SYNC_TRAFFIC_LIMIT, "OnResetNicTrafficLimit", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

type GuestSetNicTrafficsTask struct {
	SGuestBaseTask
}

func (self *GuestSetNicTrafficsTask) taskFailed(ctx context.Context, guest *models.SGuest, reason string) {
	guest.SetStatus(ctx, self.UserCred, compute.VM_SYNC_TRAFFIC_LIMIT, "PerformResetNicTrafficLimit")
	db.OpsLog.LogEvent(guest, db.ACT_SYNC_TRAFFIC_LIMIT_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_SYNC_TRAFFIC_LIMIT, reason, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(reason))
}

func (self *GuestSetNicTrafficsTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	host, err := guest.GetHost()
	if err != nil {
		self.taskFailed(ctx, guest, fmt.Sprintf("get host %s", err))
		return
	}
	input := compute.ServerNicTrafficLimit{}
	self.GetParams().Unmarshal(&input)
	self.SetStage("OnSetNicTrafficLimit", nil)
	drv, err := guest.GetDriver()
	if err != nil {
		self.taskFailed(ctx, guest, err.Error())
		return
	}
	err = drv.RequestSetNicTrafficLimit(ctx, self, host, guest, []compute.ServerNicTrafficLimit{input})
	if err != nil {
		self.taskFailed(ctx, guest, err.Error())
	}
}

func (self *GuestSetNicTrafficsTask) OnSetNicTrafficLimit(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	input := &compute.ServerNicTrafficLimit{}
	self.GetParams().Unmarshal(input)
	gn, _ := guest.GetGuestnetworkByMac(input.Mac)
	err := gn.UpdateNicTrafficLimit(input.RxTrafficLimit, input.TxTrafficLimit)
	if err != nil {
		self.taskFailed(ctx, guest, fmt.Sprintf("failed update guest nic traffic limit %s", err))
		return
	}

	oldStatus, _ := self.Params.GetString("old_status")
	guest.SetStatus(ctx, self.UserCred, oldStatus, "OnSetNicTrafficLimit")
	db.OpsLog.LogEvent(guest, db.ACT_SYNC_TRAFFIC_LIMIT, "OnSetNicTrafficLimit", self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_SYNC_TRAFFIC_LIMIT, "OnSetNicTrafficLimit", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
