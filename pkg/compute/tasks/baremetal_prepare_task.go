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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type BaremetalPrepareTask struct {
	SBaremetalBaseTask
}

func init() {
	taskman.RegisterTask(BaremetalPrepareTask{})
}

func (self *BaremetalPrepareTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	baremetal := obj.(*models.SHost)
	url := fmt.Sprintf("/baremetals/%s/prepare", baremetal.Id)
	headers := self.GetTaskRequestHeader()
	self.SetStage("OnSyncConfigComplete", nil)
	_, err := baremetal.BaremetalSyncRequest(ctx, "POST", url, headers, self.Params)
	if err != nil {
		self.OnFailure(ctx, baremetal, jsonutils.NewString(err.Error()))
	}
}

func (self *BaremetalPrepareTask) OnFailure(ctx context.Context, baremetal *models.SHost, reason jsonutils.JSONObject) {
	baremetal.SetStatus(ctx, self.UserCred, api.BAREMETAL_PREPARE_FAIL, reason.String())
	self.SetStageFailed(ctx, reason)
}

func (self *BaremetalPrepareTask) OnSyncConfigComplete(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	baremetal.ClearSchedDescCache()
	self.SetStageComplete(ctx, nil)
}

func (self *BaremetalPrepareTask) OnSyncConfigCompleteFailed(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	self.OnFailure(ctx, baremetal, body)
}
