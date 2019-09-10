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

type HostMaintainTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(HostMaintainTask{})
}

func (self *HostMaintainTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	host := obj.(*models.SHost)
	guests, _ := self.Params.Get("guests")
	preferHostId, _ := self.Params.Get("prefer_host_id")

	kwargs := jsonutils.NewDict()
	kwargs.Set("guests", guests)
	kwargs.Set("prefer_host_id", preferHostId)
	self.SetStage("OnGuestsMigrate", nil)
	err := models.GuestManager.StartHostGuestsMigrateTask(ctx, self.UserCred, host, self.Params, self.Id)
	if err != nil {
		self.TaskFailed(ctx, host, err.Error())
		return
	}
}

func (self *HostMaintainTask) OnGuestsMigrate(ctx context.Context, host *models.SHost, data jsonutils.JSONObject) {
	host.PerformDisable(ctx, self.UserCred, nil, nil)
	host.SetStatus(self.UserCred, api.HOST_MAINTAINING, "On host maintain task complete")
	logclient.AddSimpleActionLog(host, logclient.ACT_HOST_MAINTAINING, "host maintain", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *HostMaintainTask) OnGuestsMigrateFailed(ctx context.Context, host *models.SHost, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, host, data.String())
}

func (self *HostMaintainTask) TaskFailed(ctx context.Context, host *models.SHost, reason string) {
	host.PerformDisable(ctx, self.UserCred, nil, nil)
	host.SetStatus(self.UserCred, api.HOST_MAINTAIN_FAILE, "On host maintain task complete failed")
	logclient.AddSimpleActionLog(host, logclient.ACT_HOST_MAINTAINING, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}
