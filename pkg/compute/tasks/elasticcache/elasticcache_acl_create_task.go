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

package elasticcache

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ElasticcacheAclCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ElasticcacheAclCreateTask{})
}

func (self *ElasticcacheAclCreateTask) taskFail(ctx context.Context, ea *models.SElasticcacheAcl, reason jsonutils.JSONObject) {
	ea.SetStatus(ctx, self.GetUserCred(), api.ELASTIC_CACHE_ACL_STATUS_CREATE_FAILED, reason.String())
	db.OpsLog.LogEvent(ea, db.ACT_ALLOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, ea, logclient.ACT_CREATE, reason, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, ea.Id, ea.Name, api.ELASTIC_CACHE_ACL_STATUS_CREATE_FAILED, reason.String())
	self.SetStageFailed(ctx, reason)
}

func (self *ElasticcacheAclCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	ea := obj.(*models.SElasticcacheAcl)
	region := ea.GetRegion()
	if region == nil {
		self.taskFail(ctx, ea, jsonutils.NewString(fmt.Sprintf("failed to find region for elastic cache backup %s", ea.GetName())))
		return
	}

	self.SetStage("OnElasticcacheAclCreateComplete", nil)
	if err := region.GetDriver().RequestCreateElasticcacheAcl(ctx, self.GetUserCred(), ea, self); err != nil {
		self.taskFail(ctx, ea, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *ElasticcacheAclCreateTask) OnElasticcacheAclCreateComplete(ctx context.Context, ea *models.SElasticcacheAcl, data jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(self, ea, logclient.ACT_CREATE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *ElasticcacheAclCreateTask) OnElasticcacheAclCreateCompleteFailed(ctx context.Context, ea *models.SElasticcacheAcl, data jsonutils.JSONObject) {
	self.taskFail(ctx, ea, data)
}
