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
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ModelartsPoolSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ModelartsPoolSyncstatusTask{})
}

func (self *ModelartsPoolSyncstatusTask) taskFailed(ctx context.Context, modelarts *models.SModelartsPool, err error) {
	modelarts.SetStatus(ctx, self.GetUserCred(), api.MODELARTS_POOL_STATUS_UNKNOWN, err.Error())
	db.OpsLog.LogEvent(modelarts, db.ACT_SYNC_STATUS, err, self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, modelarts, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *ModelartsPoolSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	modelarts := obj.(*models.SModelartsPool)
	pool, err := modelarts.GetIModelartsPool()
	if err != nil {
		self.taskFailed(ctx, modelarts, errors.Wrapf(err, "modelarts.GetIModelartsPool()"))
		return
	}

	err = modelarts.SyncWithCloudModelartsPool(ctx, self.UserCred, pool)
	if err != nil {
		self.taskFailed(ctx, modelarts, errors.Wrap(err, "snetwork.SyncWithCloudModelartsPool()"))
		return
	}

	logclient.AddActionLogWithStartable(self, modelarts, logclient.ACT_SYNC_STATUS, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
