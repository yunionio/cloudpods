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

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type NetworkRemoteUpdateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(NetworkRemoteUpdateTask{})
}

func (self *NetworkRemoteUpdateTask) taskFail(ctx context.Context, net *models.SNetwork, err error) {
	net.SetStatus(ctx, self.UserCred, apis.STATUS_UPDATE_TAGS_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *NetworkRemoteUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	net := obj.(*models.SNetwork)
	region, err := net.GetRegion()
	if err != nil {
		self.taskFail(ctx, net, errors.Wrapf(err, "GetRegion"))
		return
	}
	self.SetStage("OnRemoteUpdateComplete", nil)
	replaceTags := jsonutils.QueryBoolean(self.Params, "replace_tags", false)

	err = region.GetDriver().RequestRemoteUpdateNetwork(ctx, self.GetUserCred(), net, replaceTags, self)
	if err != nil {
		self.taskFail(ctx, net, errors.Wrapf(err, "RequestRemoteUpdateNetwork"))
	}
}

func (self *NetworkRemoteUpdateTask) OnRemoteUpdateComplete(ctx context.Context, net *models.SNetwork, data jsonutils.JSONObject) {
	self.SetStage("OnSyncStatusComplete", nil)
	net.StartSyncstatusTask(ctx, self.UserCred, self.GetTaskId())
}

func (self *NetworkRemoteUpdateTask) OnRemoteUpdateCompleteFailed(ctx context.Context, net *models.SNetwork, data jsonutils.JSONObject) {
	self.taskFail(ctx, net, errors.Errorf(data.String()))
}

func (self *NetworkRemoteUpdateTask) OnSyncStatusComplete(ctx context.Context, net *models.SNetwork, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *NetworkRemoteUpdateTask) OnSyncStatusCompleteFailed(ctx context.Context, net *models.SNetwork, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
