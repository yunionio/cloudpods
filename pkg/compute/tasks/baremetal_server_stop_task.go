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
	"yunion.io/x/onecloud/pkg/compute/models"
)

type BaremetalServerStopTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(BaremetalServerStopTask{})
}

func (self *BaremetalServerStopTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_STOPPING, "", self.UserCred)
	guest.SetStatus(ctx, self.UserCred, api.VM_START_STOP, "")
	baremetal, _ := guest.GetHost()
	if baremetal != nil {
		self.OnStopGuestFail(ctx, guest, "Baremetal is None")
		return
	}
	params := jsonutils.NewDict()
	timeout, err := self.Params.Int("timeout")
	if err != nil {
		timeout = 30
	}
	if jsonutils.QueryBoolean(self.Params, "is_force", false) || jsonutils.QueryBoolean(self.Params, "reset", false) {
		timeout = 0
	}
	params.Set("timeout", jsonutils.NewInt(timeout))
	url := fmt.Sprintf("/baremetals/%s/servers/%s/stop", baremetal.Id, guest.Id)
	headers := self.GetTaskRequestHeader()
	self.SetStage("OnGuestStopTaskComplete", nil)
	_, err = baremetal.BaremetalSyncRequest(ctx, "POST", url, headers, params)
	if err != nil {
		log.Errorln(err)
		self.OnStopGuestFail(ctx, guest, err.Error())
	}
}

func (self *BaremetalServerStopTask) OnGuestStopTaskComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if guest.Status == api.VM_STOPPING {
		guest.SetStatus(ctx, self.UserCred, api.VM_READY, "")
		db.OpsLog.LogEvent(guest, db.ACT_STOP, "", self.UserCred)
	}
	baremetal, _ := guest.GetHost()
	baremetal.SetStatus(ctx, self.UserCred, api.BAREMETAL_READY, "")
	self.SetStageComplete(ctx, nil)
	if guest.Status == api.VM_READY {
		if !jsonutils.QueryBoolean(self.Params, "reset", false) && guest.DisableDelete.IsFalse() && guest.ShutdownBehavior == api.SHUTDOWN_TERMINATE {
			guest.StartAutoDeleteGuestTask(ctx, self.UserCred, "")
		}
	}
}

func (self *BaremetalServerStopTask) OnGuestStopTaskCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetStatus(ctx, self.UserCred, db.ACT_STOP_FAIL, data.String())
	db.OpsLog.LogEvent(guest, db.ACT_STOP_FAIL, data, self.UserCred)
	baremetal, _ := guest.GetHost()
	baremetal.SetStatus(ctx, self.UserCred, api.BAREMETAL_READY, data.String())
	self.SetStageFailed(ctx, data)
}

func (self *BaremetalServerStopTask) OnStopGuestFail(ctx context.Context, guest *models.SGuest, reason string) {
	self.OnGuestStopTaskCompleteFailed(ctx, guest, jsonutils.NewString(reason))
}
