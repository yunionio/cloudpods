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

func (self *BaremetalConvertHypervisorTask) getGuest() *models.SGuest {
	guestId, _ := self.Params.GetString("server_id")
	guestObj, _ := models.GuestManager.FetchById(guestId)
	return guestObj.(*models.SGuest)
}

func (self *BaremetalConvertHypervisorTask) getHypervisor() string {
	hostType, _ := self.Params.GetString("convert_host_type")
	return hostType
}

func (self *BaremetalConvertHypervisorTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	baremetal := obj.(*models.SHost)

	baremetal.SetStatus(ctx, self.UserCred, api.BAREMETAL_CONVERTING, "")

	self.SetStage("OnGuestDeployComplete", nil)

	guest := self.getGuest()
	params, _ := self.Params.Get("server_params")
	paramsDict := params.(*jsonutils.JSONDict)
	input, err := cmdline.FetchServerCreateInputByJSON(paramsDict)
	if err != nil {
		self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
	input.ParentTaskId = self.GetTaskId()
	models.GuestManager.OnCreateComplete(ctx, []db.IModel{guest}, self.UserCred, self.UserCred, nil, []jsonutils.JSONObject{jsonutils.Marshal(input)})
}

func (self *BaremetalConvertHypervisorTask) OnGuestDeployComplete(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	db.OpsLog.LogEvent(baremetal, db.ACT_CONVERT_COMPLETE, "", self.UserCred)

	guest := self.getGuest()
	hypervisor := self.getHypervisor()
	region, err := baremetal.GetRegion()
	if err != nil {
		self.SetStageFailed(ctx, jsonutils.NewString(fmt.Sprintf("Get Host region error %v", err)))
		return
	}
	driver, err := models.GetHostDriver(hypervisor, region.Provider)
	if err != nil {
		self.SetStageFailed(ctx, jsonutils.NewString(fmt.Sprintf("Get Host Driver error %v", err)))
		return
	}
	err = driver.FinishConvert(ctx, self.UserCred, baremetal, guest, driver.GetHostType())
	if err != nil {
		log.Errorln(err)
		logclient.AddActionLogWithStartable(self, baremetal, logclient.ACT_BM_CONVERT_HYPER, fmt.Sprintf("convert deploy falied %s", err.Error()), self.UserCred, false)
	}
	logclient.AddActionLogWithStartable(self, baremetal, logclient.ACT_BM_CONVERT_HYPER, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *BaremetalConvertHypervisorTask) OnGuestDeployCompleteFailed(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	db.OpsLog.LogEvent(baremetal, db.ACT_CONVERT_FAIL, body, self.UserCred)
	guest := self.getGuest()
	guest.SetDisableDelete(self.UserCred, false)
	self.SetStage("OnGuestDeleteComplete", nil)
	opts := api.ServerDeleteInput{OverridePendingDelete: true}
	guest.StartDeleteGuestTask(ctx, self.UserCred, self.GetTaskId(), opts)
	logclient.AddActionLogWithStartable(self, baremetal, logclient.ACT_BM_CONVERT_HYPER, fmt.Sprintf("convert deploy failed: %s", body.String()), self.UserCred, false)
}

func (self *BaremetalConvertHypervisorTask) OnGuestDeleteComplete(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	hypervisor := self.getHypervisor()
	region, err := baremetal.GetRegion()
	if err != nil {
		self.SetStageFailed(ctx, jsonutils.NewString(fmt.Sprintf("Get Host region error %v", err)))
		return
	}
	driver, _ := models.GetHostDriver(hypervisor, region.Provider)
	if driver != nil {
		driver.ConvertFailed(baremetal)
	}
	self.SetStage("OnFailedSyncstatusComplete", nil)
	baremetal.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
}

func (self *BaremetalConvertHypervisorTask) OnFailedSyncstatusComplete(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	self.SetStageFailed(ctx, body)
}
