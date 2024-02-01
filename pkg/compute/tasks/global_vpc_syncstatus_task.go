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

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GlobalVpcSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(GlobalVpcSyncstatusTask{})
}

func (self *GlobalVpcSyncstatusTask) taskFail(ctx context.Context, gvpc *models.SGlobalVpc, err error) {
	gvpc.SetStatus(ctx, self.GetUserCred(), apis.STATUS_UNKNOWN, err.Error())
	db.OpsLog.LogEvent(gvpc, db.ACT_SYNC_STATUS, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, gvpc, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *GlobalVpcSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	gvpc := obj.(*models.SGlobalVpc)

	iVpc, err := gvpc.GetICloudGlobalVpc(ctx)
	if err != nil {
		self.taskFail(ctx, gvpc, errors.Wrapf(err, "gvpc.GetICloudGlobalVpc"))
		return
	}

	err = gvpc.SyncWithCloudGlobalVpc(ctx, self.GetUserCred(), iVpc)
	if err != nil {
		self.taskFail(ctx, gvpc, errors.Wrapf(err, "gvpc.SyncWithCloudGlobalVpc"))
		return
	}

	self.SetStageComplete(ctx, nil)
}
