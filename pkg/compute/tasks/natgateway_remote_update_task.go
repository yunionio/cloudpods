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

type NatGatewayRemoteUpdateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(NatGatewayRemoteUpdateTask{})
}

func (self *NatGatewayRemoteUpdateTask) taskFail(ctx context.Context, nat *models.SNatGateway, err error) {
	nat.SetStatus(ctx, self.UserCred, apis.STATUS_UPDATE_TAGS_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *NatGatewayRemoteUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	nat := obj.(*models.SNatGateway)
	replaceTags := jsonutils.QueryBoolean(self.Params, "replace_tags", false)

	vpc, err := nat.GetVpc()
	if err != nil {
		self.taskFail(ctx, nat, errors.Wrapf(err, "GetVpc"))
		return
	}

	iNatGateway, err := nat.GetINatGateway(ctx)
	if err != nil {
		self.taskFail(ctx, nat, errors.Wrapf(err, "GetINatGateway"))
		return
	}

	oldTags, err := iNatGateway.GetTags()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
			self.OnRemoteUpdateComplete(ctx, nat, nil)
			return
		}
		self.taskFail(ctx, nat, errors.Wrapf(err, "GetTags"))
		return
	}
	tags, err := nat.GetAllUserMetadata()
	if err != nil {
		self.taskFail(ctx, nat, errors.Wrapf(err, "GetAllUserMetadata"))
		return
	}
	tagsUpdateInfo := cloudprovider.TagsUpdateInfo{OldTags: oldTags, NewTags: tags}
	err = cloudprovider.SetTags(ctx, iNatGateway, vpc.ManagerId, tags, replaceTags)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotSupported || errors.Cause(err) == cloudprovider.ErrNotImplemented {
			self.OnRemoteUpdateComplete(ctx, nat, nil)
			return
		}
		logclient.AddActionLogWithStartable(self, nat, logclient.ACT_UPDATE_TAGS, err, self.GetUserCred(), false)
		self.taskFail(ctx, nat, err)
		return
	}
	logclient.AddActionLogWithStartable(self, nat, logclient.ACT_UPDATE_TAGS, tagsUpdateInfo, self.GetUserCred(), true)
	self.OnRemoteUpdateComplete(ctx, nat, nil)
}

func (self *NatGatewayRemoteUpdateTask) OnRemoteUpdateComplete(ctx context.Context, nat *models.SNatGateway, data jsonutils.JSONObject) {
	self.SetStage("OnSyncStatusComplete", nil)
	models.StartResourceSyncStatusTask(ctx, self.UserCred, nat, "NatGatewaySyncstatusTask", self.GetTaskId())
}

func (self *NatGatewayRemoteUpdateTask) OnSyncStatusComplete(ctx context.Context, nat *models.SNatGateway, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *NatGatewayRemoteUpdateTask) OnSyncStatusCompleteFailed(ctx context.Context, nat *models.SNatGateway, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
