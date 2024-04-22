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

type BaremetalMaintenanceTask struct {
	SBaremetalBaseTask
}

func init() {
	taskman.RegisterTask(BaremetalMaintenanceTask{})
}

func (self *BaremetalMaintenanceTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	baremetal := obj.(*models.SHost)
	self.SetStage("OnEnterMaintenantModeSucc", nil)
	drv, err := baremetal.GetHostDriver()
	if err != nil {
		self.OnEnterMaintenantModeSuccFailed(ctx, baremetal, jsonutils.NewString(err.Error()))
		return
	}
	err = drv.RequestBaremetalMaintence(ctx, self.GetUserCred(), baremetal, self)
	if err != nil {
		self.OnEnterMaintenantModeSuccFailed(ctx, baremetal, jsonutils.NewString(err.Error()))
		return
	}
	baremetal.SetStatus(ctx, self.UserCred, api.BAREMETAL_MAINTAINING, "")
}

func (self *BaremetalMaintenanceTask) OnEnterMaintenantModeSucc(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	action := self.Action()
	if len(action) > 0 {
		logclient.AddActionLogWithStartable(self, baremetal, action, "", self.UserCred, true)
	}
	if action != logclient.ACT_VM_START {
		db.Update(baremetal, func() error {
			baremetal.IsMaintenance = true
			return nil
		})
	}
	username, _ := body.Get("username")
	password, _ := body.Get("password")
	ip, _ := body.Get("ip")
	metadatas := map[string]interface{}{
		"__maint_username": username,
		"__maint_password": password,
		"__maint_ip":       ip,
	}
	guestRunning, err := body.Get("guest_running")
	if err != nil {
		metadatas["__maint_guest_running"] = guestRunning
	}
	baremetal.SetAllMetadata(ctx, metadatas, self.UserCred)
	baremetal.StartSyncConfig(ctx, self.GetUserCred(), "")
	self.SetStageComplete(ctx, nil)
}

func (self *BaremetalMaintenanceTask) OnEnterMaintenantModeSuccFailed(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	self.SetStageFailed(ctx, body)
	baremetal.StartSyncstatus(ctx, self.GetUserCred(), "")
	guest := baremetal.GetBaremetalServer()
	if guest != nil {
		guest.StartSyncstatus(ctx, self.UserCred, "")
	}
	action := self.Action()
	if len(action) > 0 {
		logclient.AddActionLogWithStartable(self, baremetal, action, body, self.UserCred, false)
	}
}
