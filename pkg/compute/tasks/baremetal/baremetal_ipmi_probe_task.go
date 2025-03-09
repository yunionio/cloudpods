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

package baremetal

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type BaremetalIpmiProbeTask struct {
	SBaremetalBaseTask
}

func init() {
	taskman.RegisterTask(BaremetalIpmiProbeTask{})
}

func (self *BaremetalIpmiProbeTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	baremetal := obj.(*models.SHost)
	baremetal.SetStatus(ctx, self.UserCred, api.BAREMETAL_PROBING, "")
	url := fmt.Sprintf("/baremetals/%s/ipmi-probe", baremetal.Id)
	headers := self.GetTaskRequestHeader()
	self.SetStage("OnSyncConfigComplete", nil)
	_, err := baremetal.BaremetalSyncRequest(ctx, "POST", url, headers, self.Params)
	if err != nil {
		self.OnFailure(ctx, baremetal, jsonutils.NewString(err.Error()))
	}
}

func (self *BaremetalIpmiProbeTask) OnFailure(ctx context.Context, baremetal *models.SHost, reason jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(self, baremetal, logclient.ACT_PROBE, reason, self.UserCred, false)
	baremetal.SetStatus(ctx, self.UserCred, api.BAREMETAL_PROBE_FAIL, reason.String())
	self.SetStageFailed(ctx, reason)
}

func (self *BaremetalIpmiProbeTask) OnSyncConfigComplete(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(self, baremetal, logclient.ACT_PROBE, baremetal.GetShortDesc(ctx), self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *BaremetalIpmiProbeTask) OnSyncConfigCompleteFailed(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	self.OnFailure(ctx, baremetal, body)
}
