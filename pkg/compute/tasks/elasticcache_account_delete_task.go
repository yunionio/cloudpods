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

type ElasticcacheAccountDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ElasticcacheAccountDeleteTask{})
}

func (self *ElasticcacheAccountDeleteTask) taskFail(ctx context.Context, ea *models.SElasticcacheAccount, reason string) {
	ea.SetStatus(self.GetUserCred(), api.ELASTIC_CACHE_ACCOUNT_STATUS_DELETE_FAILED, reason)
	db.OpsLog.LogEvent(ea, db.ACT_ALLOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, ea, logclient.ACT_CREATE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(ea.Id, ea.Name, api.ELASTIC_CACHE_ACCOUNT_STATUS_DELETE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *ElasticcacheAccountDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	ea := obj.(*models.SElasticcacheAccount)
	region := ea.GetRegion()
	if region == nil {
		self.taskFail(ctx, ea, fmt.Sprintf("failed to find region for elastic cache account %s", ea.GetName()))
		return
	}

	self.SetStage("OnElasticcacheAccountDeleteComplete", nil)
	if err := region.GetDriver().RequestDeleteElasticcacheAccount(ctx, self.GetUserCred(), ea, self); err != nil {
		self.taskFail(ctx, ea, err.Error())
		return
	} else {
		err = db.DeleteModel(ctx, self.GetUserCred(), ea)
		if err != nil {
			self.taskFail(ctx, ea, err.Error())
			return
		}
		self.SetStageComplete(ctx, nil)
	}
}
