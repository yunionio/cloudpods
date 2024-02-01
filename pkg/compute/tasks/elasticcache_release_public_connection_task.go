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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ElasticcacheReleasePublicConnectionTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ElasticcacheReleasePublicConnectionTask{})
}

func (self *ElasticcacheReleasePublicConnectionTask) taskFail(ctx context.Context, elasticcache *models.SElasticcache, err error) {
	elasticcache.SetStatus(ctx, self.GetUserCred(), api.ELASTIC_CACHE_STATUS_CHANGING, err.Error())
	db.OpsLog.LogEvent(elasticcache, db.ACT_DELOCATE_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, elasticcache, logclient.ACT_DELOCATE, err, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, elasticcache.Id, elasticcache.Name, api.ELASTIC_CACHE_STATUS_CHANGE_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *ElasticcacheReleasePublicConnectionTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	elasticcache := obj.(*models.SElasticcache)
	region, err := elasticcache.GetRegion()
	if err != nil {
		self.taskFail(ctx, elasticcache, errors.Wrapf(err, "GetRegion"))
		return
	}

	self.SetStage("OnElasticcacheReleasePublicConnectionComplete", nil)
	err = region.GetDriver().RequestElasticcacheReleasePublicConnection(ctx, self.GetUserCred(), elasticcache, self)
	if err != nil {
		self.OnElasticcacheReleasePublicConnectionCompleteFailed(ctx, elasticcache, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *ElasticcacheReleasePublicConnectionTask) OnElasticcacheReleasePublicConnectionComplete(ctx context.Context, elasticcache *models.SElasticcache, data jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(self, elasticcache, logclient.ACT_DELOCATE, "release public connection", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *ElasticcacheReleasePublicConnectionTask) OnElasticcacheReleasePublicConnectionCompleteFailed(ctx context.Context, elasticcache *models.SElasticcache, reason jsonutils.JSONObject) {
	self.taskFail(ctx, elasticcache, fmt.Errorf(reason.String()))
}
