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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type MountTargetSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(MountTargetSyncstatusTask{})
}

func (self *MountTargetSyncstatusTask) taskFail(ctx context.Context, mt *models.SMountTarget, err error) {
	mt.SetStatus(ctx, self.GetUserCred(), api.MOUNT_TARGET_STATUS_UNKNOWN, err.Error())
	db.OpsLog.LogEvent(mt, db.ACT_SYNC_STATUS, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, mt, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *MountTargetSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	mt := obj.(*models.SMountTarget)

	fs, err := mt.GetFileSystem()
	if err != nil {
		self.taskFail(ctx, mt, errors.Wrapf(err, "GetFileSystem"))
		return
	}
	iFs, err := fs.GetICloudFileSystem(ctx)
	if err != nil {
		self.taskFail(ctx, mt, errors.Wrapf(err, "fs.GetICloudFileSystem"))
		return
	}
	mts, err := iFs.GetMountTargets()
	if err != nil {
		self.taskFail(ctx, mt, errors.Wrapf(err, "iFs.GetMountTargets"))
		return
	}
	for i := range mts {
		if mts[i].GetGlobalId() == mt.ExternalId {
			mt.SyncWithMountTarget(ctx, self.GetUserCred(), fs.ManagerId, mts[i])
			self.SetStageComplete(ctx, nil)
			return
		}
	}

	self.taskFail(ctx, mt, errors.Wrapf(cloudprovider.ErrNotFound, mt.ExternalId))
}
