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
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/vpcagent"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestStartTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestStartTask{})
	taskman.RegisterTask(GuestSchedStartTask{})
}

func (self *GuestStartTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_STARTING, nil, self.UserCred)
	self.RequestStart(ctx, guest)
}

func (self *GuestStartTask) RequestStart(ctx context.Context, guest *models.SGuest) {
	self.SetStage("OnStartComplete", nil)
	host, _ := guest.GetHost()
	guest.SetStatus(self.UserCred, api.VM_STARTING, "")
	err := guest.GetDriver().RequestStartOnHost(ctx, guest, host, self.UserCred, self)
	if err != nil {
		self.OnStartCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
}

func (task *GuestStartTask) OnStartComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	isVpc, err := guest.IsOneCloudVpcNetwork()
	if err != nil {
		log.Errorf("IsOneCloudVpcNetwork fail: %s", err)
	} else if isVpc {
		// force update VPC topo
		err := vpcagent.VpcAgent.DoSync(auth.GetAdminSession(ctx, options.Options.Region))
		if err != nil {
			log.Errorf("vpcagent.VpcAgent.DoSync fail %s", err)
		}
	}
	db.OpsLog.LogEvent(guest, db.ACT_START, guest.GetShortDesc(ctx), task.UserCred)
	logclient.AddActionLogWithStartable(task, guest, logclient.ACT_VM_START, guest.GetShortDesc(ctx), task.UserCred, true)
	task.taskComplete(ctx, guest)
}

func (self *GuestStartTask) OnStartCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, api.VM_START_FAILED, err.String())
	db.OpsLog.LogEvent(guest, db.ACT_START_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_START, err, self.UserCred, false)
	self.SetStageFailed(ctx, err)
}

func (self *GuestStartTask) taskComplete(ctx context.Context, guest *models.SGuest) {
	models.HostManager.ClearSchedDescCache(guest.HostId)
	self.SetStageComplete(ctx, nil)
}

type GuestSchedStartTask struct {
	SGuestBaseTask
}

func (self *GuestSchedStartTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.StartScheduler(ctx, guest)
}

func (self *GuestSchedStartTask) StartScheduler(ctx context.Context, guest *models.SGuest) {
	host, _ := guest.GetHost()
	if guestsMem := host.GetRunningGuestMemorySize(); guestsMem < 0 {
		self.TaskFailed(ctx, guest, jsonutils.NewString("Guest Start Failed: Can't Get Host Guests Memory"))
	} else {
		if float32(guestsMem+guest.VmemSize) > host.GetVirtualMemorySize() {
			self.ScheduleFailed(ctx, guest)
		} else {
			self.ScheduleSucc(ctx, guest)
		}
	}
}

func (self *GuestSchedStartTask) ScheduleFailed(ctx context.Context, guest *models.SGuest) {
	self.SetStage("OnGuestMigrate", nil)
	guest.StartMigrateTask(ctx, self.UserCred, false, false, guest.Status, "", self.GetId())
}

func (self *GuestSchedStartTask) OnGuestMigrate(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
	guest.GuestNonSchedStartTask(ctx, self.UserCred, nil, "")
}

func (self *GuestSchedStartTask) OnGuestMigrateFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, guest, data)
}

func (self *GuestSchedStartTask) ScheduleSucc(ctx context.Context, guest *models.SGuest) {
	self.SetStageComplete(ctx, nil)
	guest.GuestNonSchedStartTask(ctx, self.UserCred, nil, "")
}

func (self *GuestSchedStartTask) TaskFailed(ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject) {
	self.SetStageFailed(ctx, reason)
	guest.SetStatus(self.UserCred, api.VM_START_FAILED, reason.String())
	db.OpsLog.LogEvent(guest, db.ACT_START_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(
		self, guest, logclient.ACT_VM_START, reason, self.UserCred, false)
}
