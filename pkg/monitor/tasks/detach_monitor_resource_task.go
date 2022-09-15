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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/monitor/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DetachMonitorResourceJointTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(&DetachMonitorResourceJointTask{})
}

func (self *DetachMonitorResourceJointTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	alert := obj.(models.ICommonAlert)
	err := alert.DetachMonitorResourceJoint(ctx, self.GetUserCred())
	if err != nil {
		msg := jsonutils.NewString(fmt.Sprintf("alert:%s DetachMonitorResourceJoint err:%v", alert.GetName(), err))
		self.taskFail(ctx, alert, msg)
		return
	}
	logclient.AddActionLogWithStartable(self, alert, logclient.ACT_DETACH_MONITOR_RESOURCE_JOINT, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *DetachMonitorResourceJointTask) taskFail(ctx context.Context, alert models.ICommonAlert, msg jsonutils.JSONObject) {
	db.OpsLog.LogEvent(alert, db.ACT_DETACH_MONITOR_RESOURCE_JOINT, msg, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, alert, logclient.ACT_DETACH_MONITOR_RESOURCE_JOINT, msg, self.UserCred, false)
	self.SetStageFailed(ctx, msg)
	return
}
