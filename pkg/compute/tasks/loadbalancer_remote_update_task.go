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
)

type LoadbalancerRemoteUpdateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerRemoteUpdateTask{})
}

func (self *LoadbalancerRemoteUpdateTask) taskFail(ctx context.Context, lb *models.SLoadbalancer, err error) {
	lb.SetStatus(ctx, self.UserCred, api.LB_UPDATE_TAGS_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *LoadbalancerRemoteUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lb := obj.(*models.SLoadbalancer)
	region, err := lb.GetRegion()
	if err != nil {
		self.taskFail(ctx, lb, errors.Wrapf(err, "GetRegion"))
		return
	}
	self.SetStage("OnRemoteUpdateComplete", nil)
	replaceTags := jsonutils.QueryBoolean(self.Params, "replace_tags", false)
	if err := region.GetDriver().RequestRemoteUpdateLoadbalancer(ctx, self.GetUserCred(), lb, replaceTags, self); err != nil {
		self.taskFail(ctx, lb, errors.Wrapf(err, "RequestRemoteUpdateLoadbalancer"))
		return
	}
}

func (self *LoadbalancerRemoteUpdateTask) OnRemoteUpdateComplete(ctx context.Context, lb *models.SLoadbalancer, data jsonutils.JSONObject) {
	self.SetStage("OnSyncStatusComplete", nil)
	models.StartResourceSyncStatusTask(ctx, self.UserCred, lb, "LoadbalancerSyncstatusTask", self.GetTaskId())
}

func (self *LoadbalancerRemoteUpdateTask) OnRemoteUpdateCompleteFailed(ctx context.Context, lb *models.SLoadbalancer, data jsonutils.JSONObject) {
	self.taskFail(ctx, lb, errors.Errorf(data.String()))
}

func (self *LoadbalancerRemoteUpdateTask) OnSyncStatusComplete(ctx context.Context, lb *models.SLoadbalancer, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerRemoteUpdateTask) OnSyncStatusCompleteFailed(ctx context.Context, lb *models.SLoadbalancer, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
