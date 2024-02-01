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

type BaremetalServerStartTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(BaremetalServerStartTask{})
}

func (self *BaremetalServerStartTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(ctx, self.UserCred, api.VM_START_START, "")
	db.OpsLog.LogEvent(guest, db.ACT_STARTING, "", self.UserCred)
	baremetal, _ := guest.GetHost()
	if baremetal == nil {
		self.OnStartCompleteFailed(ctx, guest, jsonutils.NewString("Baremetal is None"))
		return
	}
	desc := guest.GetJsonDescAtBaremetal(ctx, baremetal)
	config := jsonutils.NewDict()
	config.Set("desc", jsonutils.Marshal(desc))
	url := fmt.Sprintf("/baremetals/%s/servers/%s/start", baremetal.Id, guest.Id)
	headers := self.GetTaskRequestHeader()
	self.SetStage("OnStartComplete", nil)
	_, err := baremetal.BaremetalSyncRequest(ctx, "POST", url, headers, config)
	if err != nil {
		log.Errorln(err)
		self.OnStartCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (self *BaremetalServerStartTask) OnStartComplete(ctx context.Context, guest *models.SGuest, body jsonutils.JSONObject) {
	guest.SetStatus(ctx, self.UserCred, api.VM_RUNNING, "")
	baremetal, _ := guest.GetHost()
	baremetal.SetStatus(ctx, self.UserCred, api.BAREMETAL_RUNNING, "")
	db.OpsLog.LogEvent(guest, db.ACT_START, guest.GetShortDesc(ctx), self.UserCred)
}

func (self *BaremetalServerStartTask) OnStartCompleteFailed(ctx context.Context, guest *models.SGuest, body jsonutils.JSONObject) {
	guest.SetStatus(ctx, self.UserCred, api.VM_START_FAILED, body.String())
	db.OpsLog.LogEvent(guest, db.ACT_START_FAIL, body, self.UserCred)
	baremetal, _ := guest.GetHost()
	baremetal.SetStatus(ctx, self.UserCred, api.BAREMETAL_START_FAIL, body.String())
	self.SetStageFailed(ctx, body)
}
