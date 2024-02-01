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

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestSaveTemplateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(GuestSaveTemplateTask{})
}

func (self *GuestSaveTemplateTask) taskFailed(ctx context.Context, g *models.SGuest, reason jsonutils.JSONObject) {
	g.SetStatus(ctx, self.UserCred, api.VM_TEMPLATE_SAVE_FAILED, reason.String())
	logclient.AddActionLogWithStartable(self, g, logclient.ACT_SAVE_TO_TEMPLATE, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *GuestSaveTemplateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	g := obj.(*models.SGuest)
	ci := g.ToCreateInput(ctx, self.UserCred)
	// Information not to be saved：
	// 1. Expiry release information
	if ci.BillingType == billing_api.BILLING_TYPE_POSTPAID {
		ci.Duration = ""
	}

	gtName, _ := self.Params.GetString("name")
	genGtName, _ := self.Params.GetString("generate_name")
	dict := jsonutils.NewDict()
	if len(genGtName) > 0 {
		dict.Set("generate_name", jsonutils.NewString(genGtName))
	} else {
		dict.Set("name", jsonutils.NewString(gtName))
	}
	dict.Set("description", jsonutils.NewString(fmt.Sprintf("Save from Guest '%s'", g.Name)))
	dict.Set("content", jsonutils.Marshal(ci))
	session := auth.GetSession(ctx, self.UserCred, "")
	_, err := compute.GuestTemplate.Create(session, dict)
	if err != nil {
		self.taskFailed(ctx, g, jsonutils.NewString(err.Error()))
		return
	}
	logclient.AddActionLogWithStartable(self, g, logclient.ACT_SAVE_TO_TEMPLATE, "", self.UserCred, true)

	// sync status
	self.SetStageComplete(ctx, nil)
	g.StartSyncstatus(ctx, self.UserCred, "")
}
