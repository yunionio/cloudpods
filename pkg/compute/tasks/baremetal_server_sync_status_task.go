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

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type BaremetalServerSyncStatusTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(BaremetalServerSyncStatusTask{})
}

func (self *BaremetalServerSyncStatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	baremetal, _ := guest.GetHost()
	if baremetal == nil {
		guest.SetStatus(ctx, self.UserCred, api.VM_INIT, "BaremetalServerSyncStatusTask")
		self.SetStageComplete(ctx, nil)
		return
	}
	url := fmt.Sprintf("/baremetals/%s/servers/%s/status", baremetal.Id, guest.Id)
	headers := self.GetTaskRequestHeader()
	self.SetStage("OnGuestStatusTaskComplete", nil)
	_, err := baremetal.BaremetalSyncRequest(ctx, "POST", url, headers, nil)
	if err != nil {
		log.Errorln(err)
		self.OnGetStatusFail(ctx, guest)
	}
}

func (self *BaremetalServerSyncStatusTask) OnGuestStatusTaskComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	var status string
	var hostStatus string
	host, _ := guest.GetHost()
	if data.Contains("status") {
		statusStr, _ := data.GetString("status")
		switch statusStr {
		case "running":
			status = api.VM_RUNNING
			hostStatus = api.HOST_STATUS_RUNNING
		case "stopped", "ready":
			status = api.VM_READY
			hostStatus = api.HOST_STATUS_READY
		case "admin":
			status = api.VM_ADMIN
			hostStatus = api.HOST_STATUS_RUNNING
			if !host.IsMaintenance && !host.HasBMC() {
				status = api.VM_READY
			}
		default:
			status = api.VM_INIT
			hostStatus = api.HOST_STATUS_UNKNOWN
		}
	} else {
		status = api.VM_UNKNOWN
		hostStatus = api.HOST_STATUS_UNKNOWN
	}
	guest.SetStatus(ctx, self.UserCred, status, "BaremetalServerSyncStatusTask")
	host.SetStatus(ctx, self.UserCred, hostStatus, "BaremetalServerSyncStatusTask")

	self.SetStageComplete(ctx, nil)
}

func (self *BaremetalServerSyncStatusTask) OnGuestStatusTaskCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	input := apis.PerformStatusInput{
		Status: api.VM_UNKNOWN,
	}
	guest.PerformStatus(ctx, self.UserCred, nil, input)
}

func (self *BaremetalServerSyncStatusTask) OnGetStatusFail(ctx context.Context, guest *models.SGuest) {
	kwargs := jsonutils.NewDict()
	kwargs.Set("status", jsonutils.NewString(api.VM_UNKNOWN))
	input := apis.PerformStatusInput{
		Status: api.VM_UNKNOWN,
	}
	guest.PerformStatus(ctx, self.UserCred, nil, input)
	self.SetStageComplete(ctx, nil)
}
