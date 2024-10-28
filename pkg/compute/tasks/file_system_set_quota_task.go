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
)

type FileSystemSetQuotaTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(FileSystemSetQuotaTask{})
}

func (self *FileSystemSetQuotaTask) taskFail(ctx context.Context, fs *models.SFileSystem, err error) {
	fs.SetStatus(ctx, self.UserCred, api.NAS_STATUS_AVAILABLE, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *FileSystemSetQuotaTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	fs := obj.(*models.SFileSystem)

	iFs, err := fs.GetICloudFileSystem(ctx)
	if err != nil {
		self.taskFail(ctx, fs, errors.Wrapf(err, "GetICloudFileSystem"))
		return
	}

	input := &cloudprovider.SFileSystemSetQuotaInput{}
	err = self.GetParams().Unmarshal(input)
	if err != nil {
		self.taskFail(ctx, fs, errors.Wrapf(err, "Params.Unmarshal"))
		return
	}

	err = iFs.SetQuota(input)
	if err != nil {
		self.taskFail(ctx, fs, errors.Wrapf(err, "SetQuota"))
		return
	}

	err = iFs.Refresh()
	if err != nil {
		self.taskFail(ctx, fs, errors.Wrapf(err, "Refresh"))
		return
	}

	err = fs.SyncWithCloudFileSystem(ctx, self.GetUserCred(), iFs)
	if err != nil {
		self.taskFail(ctx, fs, errors.Wrapf(err, "SyncWithCloudFileSystem"))
		return
	}

	self.SetStageComplete(ctx, nil)
}
