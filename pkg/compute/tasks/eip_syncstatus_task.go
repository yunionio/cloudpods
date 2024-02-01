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

type EipSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(EipSyncstatusTask{})
}

func (self *EipSyncstatusTask) taskFail(ctx context.Context, eip *models.SElasticip, err error) {
	eip.SetStatus(ctx, self.UserCred, api.EIP_STATUS_UNKNOWN, err.Error())
	db.OpsLog.LogEvent(eip, db.ACT_SYNC_STATUS, err, self.GetUserCred())
	logclient.AddActionLogWithStartable(self, eip, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *EipSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)

	extEip, err := eip.GetIEip(ctx)
	if err != nil {
		self.taskFail(ctx, eip, errors.Wrapf(err, "eip.GetIEip"))
		return
	}

	err = extEip.Refresh()
	if err != nil {
		self.taskFail(ctx, eip, errors.Wrapf(err, "extEip.Refresh"))
		return
	}

	err = eip.SyncWithCloudEip(ctx, self.UserCred, eip.GetCloudprovider(), extEip, nil)
	if err != nil {
		self.taskFail(ctx, eip, errors.Wrapf(err, "SyncWithCloudEip"))
		return
	}

	err = eip.SyncInstanceWithCloudEip(ctx, self.UserCred, extEip)
	if err != nil {
		self.taskFail(ctx, eip, errors.Wrapf(err, "SyncInstanceWithCloudEip"))
		return
	}

	logclient.AddActionLogWithStartable(self, eip, logclient.ACT_SYNC_STATUS, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
