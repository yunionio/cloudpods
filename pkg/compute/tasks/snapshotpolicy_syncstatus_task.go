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

type SnapshotpolicySyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SnapshotpolicySyncstatusTask{})
}

func (self *SnapshotpolicySyncstatusTask) taskFailed(ctx context.Context, sp *models.SSnapshotPolicy, err error) {
	sp.SetStatus(ctx, self.UserCred, apis.STATUS_UNKNOWN, err.Error())
	db.OpsLog.LogEvent(sp, db.ACT_SYNC_STATUS, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, sp, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *SnapshotpolicySyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	sp := obj.(*models.SSnapshotPolicy)

	iSp, err := sp.GetISnapshotPolicy(ctx)
	if err != nil {
		self.taskFailed(ctx, sp, errors.Wrapf(err, "GetISnapshotPolicy"))
		return
	}

	provider, err := sp.GetCloudprovider()
	if err != nil {
		self.taskFailed(ctx, sp, errors.Wrapf(err, "GetCloudprovider"))
		return
	}

	sp.SyncWithCloudPolicy(ctx, self.UserCred, provider, iSp)
	self.SetStageComplete(ctx, nil)
}
