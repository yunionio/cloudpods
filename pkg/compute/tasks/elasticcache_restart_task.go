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
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ElasticcacheRestartTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ElasticcacheRestartTask{})
}

func (self *ElasticcacheRestartTask) taskFail(ctx context.Context, elasticcache *models.SElasticcache, reason string) {
	elasticcache.SetStatus(self.GetUserCred(), api.ELASTIC_CACHE_STATUS_RESTART_FAILED, reason)
	db.OpsLog.LogEvent(elasticcache, db.ACT_RESTART_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, elasticcache, logclient.ACT_VM_RESTART, reason, self.UserCred, false)
	notifyclient.NotifySystemError(elasticcache.Id, elasticcache.Name, api.ELASTIC_CACHE_STATUS_RESTART_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *ElasticcacheRestartTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	ec := obj.(*models.SElasticcache)
	region := ec.GetRegion()
	if region == nil {
		self.taskFail(ctx, ec, fmt.Sprintf("failed to find region for elastic cache %s", ec.GetName()))
		return
	}

	self.SetStage("OnElasticcacheRestartComplete", nil)
	if err := region.GetDriver().RequestRestartElasticcache(ctx, self.GetUserCred(), ec, self); err != nil {
		self.taskFail(ctx, ec, err.Error())
		return
	} else {
		logclient.AddActionLogWithStartable(self, ec, logclient.ACT_VM_RESTART, nil, self.UserCred, true)
		self.SetStageComplete(ctx, nil)
	}
}
