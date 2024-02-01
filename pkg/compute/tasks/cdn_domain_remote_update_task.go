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

type CDNDomainRemoteUpdateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CDNDomainRemoteUpdateTask{})
}

func (self *CDNDomainRemoteUpdateTask) taskFail(ctx context.Context, cdn *models.SCDNDomain, err error) {
	cdn.SetStatus(ctx, self.UserCred, apis.STATUS_UPDATE_TAGS_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *CDNDomainRemoteUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cdn := obj.(*models.SCDNDomain)
	replaceTags := jsonutils.QueryBoolean(self.Params, "replace_tags", false)

	iCDNDomain, err := cdn.GetICloudCDNDomain(ctx)
	if err != nil {
		self.taskFail(ctx, cdn, errors.Wrapf(err, "GetICloudCDNDomain"))
		return
	}

	oldTags, err := iCDNDomain.GetTags()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
			self.OnRemoteUpdateComplete(ctx, cdn, nil)
			return
		}
		self.taskFail(ctx, cdn, errors.Wrapf(err, "GetTags"))
		return
	}
	tags, err := cdn.GetAllUserMetadata()
	if err != nil {
		self.taskFail(ctx, cdn, errors.Wrapf(err, "GetAllUserMetadata"))
		return
	}
	tagsUpdateInfo := cloudprovider.TagsUpdateInfo{OldTags: oldTags, NewTags: tags}
	err = cloudprovider.SetTags(ctx, iCDNDomain, cdn.ManagerId, tags, replaceTags)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
			self.OnRemoteUpdateComplete(ctx, cdn, nil)
			return
		}
		logclient.AddActionLogWithStartable(self, cdn, logclient.ACT_UPDATE_TAGS, err, self.GetUserCred(), false)
		self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
	logclient.AddActionLogWithStartable(self, cdn, logclient.ACT_UPDATE_TAGS, tagsUpdateInfo, self.GetUserCred(), true)
	self.OnRemoteUpdateComplete(ctx, cdn, nil)
}

func (self *CDNDomainRemoteUpdateTask) OnRemoteUpdateComplete(ctx context.Context, cdn *models.SCDNDomain, data jsonutils.JSONObject) {
	self.SetStage("OnSyncStatusComplete", nil)
	models.StartResourceSyncStatusTask(ctx, self.UserCred, cdn, "CDNDomainSyncstatusTask", self.GetTaskId())
}

func (self *CDNDomainRemoteUpdateTask) OnSyncStatusComplete(ctx context.Context, cdn *models.SCDNDomain, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *CDNDomainRemoteUpdateTask) OnSyncStatusCompleteFailed(ctx context.Context, cdn *models.SCDNDomain, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
