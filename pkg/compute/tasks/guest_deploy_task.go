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
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestDeployTask struct {
	SGuestBaseTask
}

func (self *GuestDeployTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if !guest.IsNetworkAllocated() {
		self.SetStageFailed(ctx, jsonutils.NewString(fmt.Sprintf("Guest %s network not ready!!", guest.Name)))
	} else {
		self.OnGuestNetworkReady(ctx, guest)
	}
}

func (self *GuestDeployTask) OnGuestNetworkReady(ctx context.Context, guest *models.SGuest) {
	self.SetStage("OnDeployWaitServerStop", nil)
	if jsonutils.QueryBoolean(self.Params, "restart", false) {
		guest.StartGuestStopTask(ctx, self.UserCred, 60, false, false, self.GetTaskId())
	} else {
		// Note: have to use LocalTaskRun, run to another place implement OnDeployWaitServerStop
		taskman.LocalTaskRun(self, func() (jsonutils.JSONObject, error) {
			return nil, nil
		})
	}
}

func (self *GuestDeployTask) OnDeployWaitServerStop(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStage("OnDeployGuestComplete", nil)
	targetHostId, _ := self.Params.GetString("target_host_id")
	if len(targetHostId) == 0 {
		targetHostId = guest.HostId
	}
	host := models.HostManager.FetchHostById(targetHostId)
	self.DeployOnHost(ctx, guest, host)
}

func (self *GuestDeployTask) DeployOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost) {
	drv, err := guest.GetDriver()
	if err != nil {
		self.OnDeployGuestFail(ctx, guest, err)
		return
	}
	err = drv.RequestDeployGuestOnHost(ctx, guest, host, self)
	if err != nil {
		log.Errorf("request_deploy_guest_on_host %s", err)
		self.OnDeployGuestFail(ctx, guest, err)
	} else {
		guest.SetStatus(ctx, self.UserCred, api.VM_DEPLOYING, "")
	}
}

func (self *GuestDeployTask) OnDeployGuestFail(ctx context.Context, guest *models.SGuest, err error) {
	guest.SetStatus(ctx, self.UserCred, api.VM_DEPLOY_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_DEPLOY, err, self.UserCred, false)
	db.OpsLog.LogEvent(guest, db.ACT_VM_DEPLOY_FAIL, err.Error(), self.UserCred)
}

func (self *GuestDeployTask) OnDeployGuestComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	log.Infof("on_guest_deploy_task_data_received %s", data)
	guest := obj.(*models.SGuest)
	drv, err := guest.GetDriver()
	if err != nil {
		self.OnDeployGuestCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	drv.OnGuestDeployTaskDataReceived(ctx, guest, self, data)
	action, _ := self.Params.GetString("deploy_action")
	keypair, _ := self.Params.GetString("keypair")
	reset_password := jsonutils.QueryBoolean(self.Params, "reset_password", false)
	unbind_kp := jsonutils.QueryBoolean(self.Params, "__delete_keypair__", false)
	_log := false
	if action == "deploy" {
		if len(keypair) >= 32 {
			if unbind_kp {
				logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_UNBIND_KEYPAIR, nil, self.UserCred, true)
				_log = true
			} else {
				logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_BIND_KEYPAIR, nil, self.UserCred, true)
				_log = true
			}

		} else if reset_password {
			logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_RESET_PSWD, "", self.UserCred, true)
			guest.EventNotify(ctx, self.UserCred, notifyclient.ActionResetPassword)
			_log = true
		}
	}
	if !_log {
		// 如果 deploy 有其他事件，统一记在这里。
		logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_DEPLOY, action, self.UserCred, true)
	}

	if self.HasParentTask() {
		self.SetStageComplete(ctx, nil)
		return
	}

	if jsonutils.QueryBoolean(self.GetParams(), "restart", false) {
		self.SetStage("OnDeployStartGuestComplete", nil)
		guest.StartGueststartTask(ctx, self.GetUserCred(), nil, self.GetTaskId())
	} else {
		self.SetStage("OnDeployGuestSyncstatusComplete", nil)
		guest.StartSyncstatus(ctx, self.GetUserCred(), self.GetTaskId())
	}
}

func (self *GuestDeployTask) OnDeployGuestCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	action, _ := self.Params.GetString("deploy_action")
	keypair, _ := self.Params.GetString("keypair")
	if action == "deploy" && len(keypair) >= 32 {
		_, err := db.Update(guest, func() error {
			guest.KeypairId = ""
			return nil
		})
		if err != nil {
			log.Errorf("unset guest %s keypair failed %v", guest.Name, err)
		}
	}
	guest.SetStatus(ctx, self.UserCred, api.VM_DEPLOY_FAILED, data.String())
	self.SetStageFailed(ctx, data)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_DEPLOY, data, self.UserCred, false)
	db.OpsLog.LogEvent(guest, db.ACT_VM_DEPLOY_FAIL, data, self.UserCred)
}

func (self *GuestDeployTask) OnDeployStartGuestComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestDeployTask) OnDeployStartGuestCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}

func (self *GuestDeployTask) OnDeployGuestSyncstatusComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestDeployTask) OnDeployGuestSyncstatusCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}

func init() {
	taskman.RegisterTask(GuestDeployTask{})
}
