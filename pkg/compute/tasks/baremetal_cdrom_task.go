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

type BaremetalCdromTask struct {
	SBaremetalBaseTask
}

func init() {
	taskman.RegisterTask(BaremetalCdromTask{})
}

func (self *BaremetalCdromTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	baremetal := obj.(*models.SHost)
	action, _ := self.Params.GetString("action")
	if action == api.BAREMETAL_CDROM_ACTION_INSERT {
		baremetal.SetStatus(self.UserCred, api.BAREMETAL_INSERTING_ISO, "")
	} else {
		baremetal.SetStatus(self.UserCred, api.BAREMETAL_EJECTING_ISO, "")
	}
	url := fmt.Sprintf("/baremetals/%s/cdrom", baremetal.Id)
	headers := self.GetTaskRequestHeader()
	self.SetStage("OnSyncConfigComplete", nil)
	_, err := baremetal.BaremetalSyncRequest(ctx, "POST", url, headers, self.Params)
	if err != nil {
		self.OnFailure(ctx, baremetal, err.Error())
	}
}

func (self *BaremetalCdromTask) OnFailure(ctx context.Context, baremetal *models.SHost, reason string) {
	action, _ := self.Params.GetString("action")
	if action == api.BAREMETAL_CDROM_ACTION_INSERT {
		baremetal.SetStatus(self.UserCred, api.BAREMETAL_INSERT_FAIL, reason)
	} else {
		baremetal.SetStatus(self.UserCred, api.BAREMETAL_EJECT_FAIL, reason)
	}
	self.SetStageFailed(ctx, reason)
}

func (self *BaremetalCdromTask) OnSyncConfigComplete(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *BaremetalCdromTask) OnSyncConfigCompleteFailed(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	reason, _ := body.GetString("__reason__")
	self.OnFailure(ctx, baremetal, reason)
}
