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

	api "yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/monitor/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ResolveUnusedTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ResolveUnusedTask{})
}

func (self *ResolveUnusedTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	suggestSysAlert := obj.(*models.SSuggestSysAlert)
	err := suggestSysAlert.GetDriver().Resolve(suggestSysAlert)
	if err != nil {
		msg := fmt.Sprintf("fail to delete %s", err)
		self.taskFail(ctx, suggestSysAlert, jsonutils.NewString(msg))
		return
	}
	suggestSysAlert.SetStatus(self.UserCred, api.SUGGEST_ALERT_DELETING, "")
	err = suggestSysAlert.RealDelete(ctx, self.UserCred)
	if err != nil {
		msg := fmt.Sprintf("fail to delete SSuggestSysAlert %s", err)
		self.taskFail(ctx, suggestSysAlert, jsonutils.NewString(msg))
		return
	}
	db.OpsLog.LogEvent(suggestSysAlert, db.ACT_DELETE, nil, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, suggestSysAlert, logclient.ACT_DELETE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *ResolveUnusedTask) taskFail(ctx context.Context, alert *models.SSuggestSysAlert, msg jsonutils.JSONObject) {
	alert.SetStatus(self.UserCred, api.SUGGEST_ALERT_DELETE_FAIL, msg.String())
	db.OpsLog.LogEvent(alert, db.ACT_DELETE, msg, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, alert, logclient.ACT_DELETE, msg, self.UserCred, false)
	self.SetStageFailed(ctx, msg)
	return
}
