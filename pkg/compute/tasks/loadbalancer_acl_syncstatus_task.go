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

	"yunion.io/x/cloudmux/pkg/apis"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type LoadbalancerAclSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerAclSyncstatusTask{})
}

func (self *LoadbalancerAclSyncstatusTask) taskFail(ctx context.Context, lbacl *models.SLoadbalancerAcl, err error) {
	lbacl.SetStatus(ctx, self.GetUserCred(), apis.STATUS_UNKNOWN, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *LoadbalancerAclSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbacl := obj.(*models.SLoadbalancerAcl)
	region, err := lbacl.GetRegion()
	if err != nil {
		self.taskFail(ctx, lbacl, errors.Wrapf(err, "GetRegion"))
		return
	}
	self.SetStage("OnLoadbalancerAclSyncstatusComplete", nil)
	err = region.GetDriver().RequestLoadbalancerAclSyncstatus(ctx, self.GetUserCred(), lbacl, self)
	if err != nil {
		self.taskFail(ctx, lbacl, errors.Wrapf(err, "RequestLoadbalancerAclSyncstatus"))
	}
}

func (self *LoadbalancerAclSyncstatusTask) OnLoadbalancerAclSyncstatusComplete(ctx context.Context, lbacl *models.SLoadbalancerAcl, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerAclSyncstatusTask) OnLoadbalancerAclSyncstatusCompleteFailed(ctx context.Context, lbacl *models.SLoadbalancerAcl, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbacl, errors.Errorf(reason.String()))
}
