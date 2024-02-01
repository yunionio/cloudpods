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

type FileSystemRemoteUpdateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(FileSystemRemoteUpdateTask{})
}

func (self *FileSystemRemoteUpdateTask) taskFail(ctx context.Context, fs *models.SFileSystem, err error) {
	fs.SetStatus(ctx, self.UserCred, api.NAS_UPDATE_TAGS_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *FileSystemRemoteUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	fs := obj.(*models.SFileSystem)
	self.SetStage("OnRemoteUpdateComplete", nil)
	replaceTags := jsonutils.QueryBoolean(self.Params, "replace_tags", false)

	iFs, err := fs.GetICloudFileSystem(ctx)
	if err != nil {
		self.taskFail(ctx, fs, errors.Wrapf(err, "GetICloudFileSystem"))
		return
	}

	taskman.LocalTaskRun(self, func() (jsonutils.JSONObject, error) {
		oldTags, err := iFs.GetTags()
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
				return nil, nil
			}
			return nil, errors.Wrap(err, "iFs.GetTags()")
		}
		tags, err := fs.GetAllUserMetadata()
		if err != nil {
			return nil, errors.Wrapf(err, "fs.GetAllUserMetadata")
		}
		tagsUpdateInfo := cloudprovider.TagsUpdateInfo{OldTags: oldTags, NewTags: tags}
		err = cloudprovider.SetTags(ctx, iFs, fs.ManagerId, tags, replaceTags)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
				return nil, nil
			}
			logclient.AddActionLogWithStartable(self, fs, logclient.ACT_UPDATE_TAGS, err, self.GetUserCred(), false)
			return nil, errors.Wrap(err, "iFs.SetMetadata")
		}
		logclient.AddActionLogWithStartable(self, fs, logclient.ACT_UPDATE_TAGS, tagsUpdateInfo, self.GetUserCred(), true)
		return nil, nil
	})
}

func (self *FileSystemRemoteUpdateTask) OnRemoteUpdateComplete(ctx context.Context, fs *models.SFileSystem, data jsonutils.JSONObject) {
	self.SetStage("OnSyncStatusComplete", nil)
	models.StartResourceSyncStatusTask(ctx, self.UserCred, fs, "FileSystemSyncstatusTask", self.GetTaskId())
}

func (self *FileSystemRemoteUpdateTask) OnRemoteUpdateCompleteFailed(ctx context.Context, fs *models.SFileSystem, data jsonutils.JSONObject) {
	self.taskFail(ctx, fs, errors.Errorf(data.String()))
}

func (self *FileSystemRemoteUpdateTask) OnSyncStatusComplete(ctx context.Context, fs *models.SFileSystem, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *FileSystemRemoteUpdateTask) OnSyncStatusCompleteFailed(ctx context.Context, fs *models.SFileSystem, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
