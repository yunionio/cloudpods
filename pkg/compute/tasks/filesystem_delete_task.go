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

// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or fsreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific langufse governing permissions and
// limitations under the License.

package tasks

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type FileSystemDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(FileSystemDeleteTask{})
}

func (self *FileSystemDeleteTask) taskFailed(ctx context.Context, fs *models.SFileSystem, err error) {
	fs.SetStatus(self.UserCred, api.NAS_STATUS_DELETE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, fs, logclient.ACT_DELOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *FileSystemDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	fs := obj.(*models.SFileSystem)

	if len(fs.ExternalId) == 0 {
		self.taskComplete(ctx, fs)
		return
	}

	iFs, err := fs.GetICloudFileSystem()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, fs)
			return
		}
		self.taskFailed(ctx, fs, errors.Wrapf(err, "fs.GetICloudFileSystem"))
		return
	}
	err = func() error {
		mts, err := iFs.GetMountTargets()
		if err != nil {
			return errors.Wrapf(err, "iFs.GetMountTargets")
		}
		for i := range mts {
			err = mts[i].Delete()
			if err != nil {
				return errors.Wrapf(err, "Delete MountTarget")
			}
		}
		return nil
	}()
	if err != nil {
		self.taskFailed(ctx, fs, errors.Wrapf(err, "Delete MountTarget"))
		return
	}
	err = iFs.Delete()
	if err != nil {
		self.taskFailed(ctx, fs, errors.Wrapf(err, "iFs.Delete"))
		return
	}
	cloudprovider.WaitDeleted(iFs, time.Second*10, time.Minute*5)
	self.taskComplete(ctx, fs)
}

func (self *FileSystemDeleteTask) taskComplete(ctx context.Context, fs *models.SFileSystem) {
	fs.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}
