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

	"yunion.io/x/onecloud/pkg/cloudcommon/cmdline"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type BaremetalConvertHypervisorTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(BaremetalConvertHypervisorTask{})
}

func (self *BaremetalConvertHypervisorTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	baremetal := obj.(*models.SHost)

	baremetal.SetStatus(self.UserCred, models.BAREMETAL_CONVERTING, "")

	self.SetStage("on_guest_deploy_complete", nil)

	guestId, _ := self.Params.GetString("server_id")
	guestObj, _ := models.GuestManager.FetchById(guestId)
	guest := guestObj.(*models.SGuest)
	params, _ := self.Params.Get("server_params")
	paramsDict := params.(*jsonutils.JSONDict)
	pendingUsage := models.SQuota{}
	self.GetPendingUsage(&pendingUsage)
	input, err := cmdline.FetchServerCreateInputByJSON(paramsDict)
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
		return
	}
	guest.StartGuestCreateTask(ctx, self.UserCred, input, &pendingUsage, self.GetId())
}

func (self *BaremetalConvertHypervisorTask) OnGuestDeployComplete(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	db.OpsLog.LogEvent(baremetal, db.ACT_CONVERT_COMPLETE, "", self.UserCred)

	guestId, _ := self.Params.GetString("server_id")
	guestObj, _ := models.GuestManager.FetchById(guestId)
	guest := guestObj.(*models.SGuest)
	data, _ := self.Params.Get("server_params")
	hypervisor, _ := data.GetString("__convert_host_type__")
	driver := models.GetHostDriver(hypervisor)
	if driver == nil {
		self.SetStageFailed(ctx, fmt.Sprintf("Get Host Driver error %s", hypervisor))
	}
	err := driver.FinishConvert(self.UserCred, baremetal, guest, driver.GetHostType())
	if err != nil {
		log.Errorln(err)
		logclient.AddActionLogWithStartable(self, baremetal, logclient.ACT_BM_CONVERT_HYPER, fmt.Sprintf("convert deploy falied %s", err.Error()), self.UserCred, false)
	}
	logclient.AddActionLogWithStartable(self, baremetal, logclient.ACT_BM_CONVERT_HYPER, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *BaremetalConvertHypervisorTask) OnGuestDeployCompleteFailed(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	db.OpsLog.LogEvent(baremetal, db.ACT_CONVERT_FAIL, body, self.UserCred)
	guestId, _ := self.Params.GetString("server_id")
	guestObj, _ := models.GuestManager.FetchById(guestId)
	guest := guestObj.(*models.SGuest)
	guest.SetDisableDelete(self.UserCred, false)
	self.SetStage("OnGuestDeleteComplete", nil)
	guest.StartDeleteGuestTask(ctx, self.UserCred, self.GetTaskId(), false, true)
	logclient.AddActionLogWithStartable(self, baremetal, logclient.ACT_BM_CONVERT_HYPER, "convert deploy failed", self.UserCred, false)
}

func (self *BaremetalConvertHypervisorTask) OnGuestDeleteComplete(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	data, _ := self.Params.Get("server_params")
	hypervisor, _ := data.GetString("__convert_host_type__")
	driver := models.GetHostDriver(hypervisor)
	if driver == nil {
		self.SetStageFailed(ctx, fmt.Sprintf("Get Host Driver error %s", hypervisor))
	}
	err := driver.ConvertFailed(baremetal)
	if err != nil {
		logclient.AddActionLogWithStartable(self, baremetal, logclient.ACT_BM_CONVERT_HYPER, fmt.Sprintf("convert failed: %s", err), self.UserCred, false)
	}
	self.SetStage("OnFailedSyncstatusComplete", nil)
	baremetal.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
}

func (self *BaremetalConvertHypervisorTask) OnFailedSyncstatusComplete(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	self.SetStageFailed(ctx, "convert failed")
}
