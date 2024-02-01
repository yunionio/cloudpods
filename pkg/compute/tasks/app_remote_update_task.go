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

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type AppRemoteUpdateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(AppRemoteUpdateTask{})
}

func (self *AppRemoteUpdateTask) taskFail(ctx context.Context, app *models.SApp, err error) {
	app.SetStatus(ctx, self.UserCred, apis.STATUS_UPDATE_TAGS_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *AppRemoteUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	app := obj.(*models.SApp)
	replaceTags := jsonutils.QueryBoolean(self.Params, "replace_tags", false)

	iApp, err := app.GetIApp(ctx)
	if err != nil {
		self.taskFail(ctx, app, errors.Wrapf(err, "GetIApp"))
		return
	}

	oldTags, err := iApp.GetTags()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
			self.OnRemoteUpdateComplete(ctx, app, nil)
			return
		}
		self.taskFail(ctx, app, errors.Wrapf(err, "GetTags"))
		return
	}
	tags, err := app.GetAllUserMetadata()
	if err != nil {
		self.taskFail(ctx, app, errors.Wrapf(err, "GetAllUserMetadata"))
		return
	}
	tagsUpdateInfo := cloudprovider.TagsUpdateInfo{OldTags: oldTags, NewTags: tags}
	err = cloudprovider.SetTags(ctx, iApp, app.ManagerId, tags, replaceTags)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
			self.OnRemoteUpdateComplete(ctx, app, nil)
			return
		}
		logclient.AddActionLogWithStartable(self, app, logclient.ACT_UPDATE_TAGS, err, self.GetUserCred(), false)
		self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
	logclient.AddActionLogWithStartable(self, app, logclient.ACT_UPDATE_TAGS, tagsUpdateInfo, self.GetUserCred(), true)
	self.OnRemoteUpdateComplete(ctx, app, nil)
}

func (self *AppRemoteUpdateTask) OnRemoteUpdateComplete(ctx context.Context, app *models.SApp, data jsonutils.JSONObject) {
	self.SetStage("OnSyncStatusComplete", nil)
	models.StartResourceSyncStatusTask(ctx, self.UserCred, app, "AppSyncstatusTask", self.GetTaskId())
}

func (self *AppRemoteUpdateTask) OnSyncStatusComplete(ctx context.Context, app *models.SApp, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *AppRemoteUpdateTask) OnSyncStatusCompleteFailed(ctx context.Context, app *models.SApp, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
