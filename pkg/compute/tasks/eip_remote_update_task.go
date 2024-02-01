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

type EipRemoteUpdateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(EipRemoteUpdateTask{})
}

func (self *EipRemoteUpdateTask) taskFail(ctx context.Context, eip *models.SElasticip, err error) {
	eip.SetStatus(ctx, self.UserCred, apis.STATUS_UPDATE_TAGS_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *EipRemoteUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)
	replaceTags := jsonutils.QueryBoolean(self.Params, "replace_tags", false)

	iEip, err := eip.GetIEip(ctx)
	if err != nil {
		self.taskFail(ctx, eip, errors.Wrapf(err, "GetIEip"))
		return
	}

	oldTags, err := iEip.GetTags()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
			self.OnRemoteUpdateComplete(ctx, eip, nil)
			return
		}
		self.taskFail(ctx, eip, errors.Wrapf(err, "GetTags"))
		return
	}
	tags, err := eip.GetAllUserMetadata()
	if err != nil {
		self.taskFail(ctx, eip, errors.Wrapf(err, "GetAllUserMetadata"))
		return
	}
	tagsUpdateInfo := cloudprovider.TagsUpdateInfo{OldTags: oldTags, NewTags: tags}
	err = cloudprovider.SetTags(ctx, iEip, eip.ManagerId, tags, replaceTags)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
			self.OnRemoteUpdateComplete(ctx, eip, nil)
			return
		}
		logclient.AddActionLogWithStartable(self, eip, logclient.ACT_UPDATE_TAGS, err, self.GetUserCred(), false)
		self.taskFail(ctx, eip, err)
		return
	}
	logclient.AddActionLogWithStartable(self, eip, logclient.ACT_UPDATE_TAGS, tagsUpdateInfo, self.GetUserCred(), true)
	self.OnRemoteUpdateComplete(ctx, eip, nil)
}

func (self *EipRemoteUpdateTask) OnRemoteUpdateComplete(ctx context.Context, eip *models.SElasticip, data jsonutils.JSONObject) {
	self.SetStage("OnSyncStatusComplete", nil)
	models.StartResourceSyncStatusTask(ctx, self.UserCred, eip, "EipSyncstatusTask", self.GetTaskId())
}

func (self *EipRemoteUpdateTask) OnSyncStatusComplete(ctx context.Context, eip *models.SElasticip, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *EipRemoteUpdateTask) OnSyncStatusCompleteFailed(ctx context.Context, eip *models.SElasticip, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
