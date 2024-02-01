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

type MountTargetDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(MountTargetDeleteTask{})
}

func (self *MountTargetDeleteTask) taskFailed(ctx context.Context, mt *models.SMountTarget, err error) {
	mt.SetStatus(ctx, self.UserCred, api.MOUNT_TARGET_STATUS_DELETE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, mt, logclient.ACT_DELOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *MountTargetDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	mt := obj.(*models.SMountTarget)

	if len(mt.ExternalId) == 0 {
		self.taskComplete(ctx, nil, mt)
		return
	}

	fs, err := mt.GetFileSystem()
	if err != nil {
		self.taskFailed(ctx, mt, errors.Wrapf(err, "GetFileSystem"))
		return
	}
	iFileSystem, err := fs.GetICloudFileSystem(ctx)
	if err != nil {
		self.taskFailed(ctx, mt, errors.Wrapf(err, "GetICloudFileSystem"))
		return
	}
	iMountTargets, err := iFileSystem.GetMountTargets()
	if err != nil {
		self.taskFailed(ctx, mt, errors.Wrapf(err, "GetMountTargets"))
		return
	}
	for i := range iMountTargets {
		if iMountTargets[i].GetGlobalId() == mt.ExternalId {
			err = iMountTargets[i].Delete()
			if err != nil {
				self.taskFailed(ctx, mt, errors.Wrapf(err, "iMountTarget.Delete"))
				return
			}
			self.taskComplete(ctx, fs, mt)
			return
		}
	}

	self.taskComplete(ctx, fs, mt)
}

func (self *MountTargetDeleteTask) taskComplete(ctx context.Context, fs *models.SFileSystem, mt *models.SMountTarget) {
	if fs != nil {
		logclient.AddActionLogWithStartable(self, fs, logclient.ACT_DELOCATE, mt, self.UserCred, true)
	}
	mt.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}
