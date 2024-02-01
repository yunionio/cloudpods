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
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SecurityGroupRemoteUpdateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SecurityGroupRemoteUpdateTask{})
}

func (self *SecurityGroupRemoteUpdateTask) taskFailed(ctx context.Context, group *models.SSecurityGroup, err error) {
	group.SetStatus(ctx, self.UserCred, apis.STATUS_UPDATE_TAGS_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *SecurityGroupRemoteUpdateTask) taskComplete(ctx context.Context, group *models.SSecurityGroup) {
	group.SetStatus(ctx, self.UserCred, api.SECGROUP_STATUS_READY, "")
	self.SetStageComplete(ctx, nil)
}

func (self *SecurityGroupRemoteUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	group := obj.(*models.SSecurityGroup)

	provider, err := group.GetCloudprovider()
	if err != nil {
		self.taskFailed(ctx, group, errors.Wrapf(err, "GetCloudprovider"))
		return
	}

	iGroup, err := group.GetISecurityGroup(ctx)
	if err != nil {
		self.taskFailed(ctx, group, errors.Wrapf(err, "GetISecurityGroup"))
		return
	}

	oldTags, err := iGroup.GetTags()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
			self.taskComplete(ctx, group)
			return
		}
		self.taskFailed(ctx, group, errors.Wrapf(err, "GetTags"))
		return
	}
	tags, err := group.GetAllUserMetadata()
	if err != nil {
		self.taskFailed(ctx, group, errors.Wrapf(err, "GetAllUserMetadata"))
		return
	}
	tagsUpdateInfo := cloudprovider.TagsUpdateInfo{OldTags: oldTags, NewTags: tags}

	replaceTags := jsonutils.QueryBoolean(self.Params, "replace_tags", false)

	err = cloudprovider.SetTags(ctx, iGroup, group.ManagerId, tags, replaceTags)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
			self.taskComplete(ctx, group)
			return
		}
		logclient.AddSimpleActionLog(group, logclient.ACT_UPDATE_TAGS, err, self.UserCred, false)
		self.taskFailed(ctx, group, errors.Wrapf(err, "SetTags"))
		return
	}
	logclient.AddSimpleActionLog(group, logclient.ACT_UPDATE_TAGS, tagsUpdateInfo, self.UserCred, true)
	// sync back cloud metadata
	iGroup.Refresh()

	group.SyncWithCloudSecurityGroup(ctx, self.UserCred, iGroup, provider.GetOwnerId(), false)
	self.taskComplete(ctx, group)
}
