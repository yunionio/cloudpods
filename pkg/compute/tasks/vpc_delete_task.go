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
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type VpcDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(VpcDeleteTask{})
}

func (self *VpcDeleteTask) taskFailed(ctx context.Context, vpc *models.SVpc, err error) {
	vpc.SetStatus(ctx, self.UserCred, api.VPC_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(vpc, db.ACT_DELOCATE_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, vpc, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *VpcDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	vpc := obj.(*models.SVpc)
	vpc.SetStatus(ctx, self.UserCred, api.VPC_STATUS_DELETING, "")
	region, err := vpc.GetRegion()
	if err != nil {
		self.taskFailed(ctx, vpc, errors.Wrap(err, "vpc.GetRegion"))
		return
	}

	self.SetStage("OnDeleteVpcComplete", nil)
	err = region.GetDriver().RequestDeleteVpc(ctx, self.UserCred, region, vpc, self)
	if err != nil {
		self.taskFailed(ctx, vpc, errors.Wrapf(err, "RequestDeleteVpc"))
		return
	}
}

func (self *VpcDeleteTask) OnDeleteVpcComplete(ctx context.Context, vpc *models.SVpc, body jsonutils.JSONObject) {
	err := vpc.RealDelete(ctx, self.UserCred)
	if err != nil {
		self.taskFailed(ctx, vpc, errors.Wrap(err, "RealDelete"))
		return
	}
	db.OpsLog.LogEvent(vpc, db.ACT_DELOCATING, vpc.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, vpc, logclient.ACT_DELETE, nil, self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    vpc,
		Action: notifyclient.ActionDelete,
	})
	self.SetStageComplete(ctx, nil)
}

func (self *VpcDeleteTask) OnDeleteVpcCompleteFailed(ctx context.Context, vpc *models.SVpc, reason jsonutils.JSONObject) {
	self.taskFailed(ctx, vpc, errors.Errorf(reason.String()))
}
