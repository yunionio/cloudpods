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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type BaremetalServerResetTask struct {
	SGuestBaseTask
}

func (self *BaremetalServerResetTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	baremetal, _ := guest.GetHost()
	if baremetal == nil {
		self.SetStageFailed(ctx, jsonutils.NewString("Baremetal is not found"))
		return
	}
	url := fmt.Sprintf("/baremetals/%s/servers/%s/reset", baremetal.Id, guest.Id)
	headers := self.GetTaskRequestHeader()
	_, err := baremetal.BaremetalSyncRequest(ctx, "POST", url, headers, nil)
	if err != nil {
		log.Errorln(err)
		self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
	} else {
		self.SetStageComplete(ctx, nil)
	}
}
