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

type VpcCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(VpcCreateTask{})
}

func (self *VpcCreateTask) taskFailed(ctx context.Context, vpc *models.SVpc, err error) {
	vpc.SetStatus(ctx, self.UserCred, api.VPC_STATUS_FAILED, err.Error())
	db.OpsLog.LogEvent(vpc, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, vpc, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *VpcCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	vpc := obj.(*models.SVpc)
	vpc.SetStatus(ctx, self.UserCred, api.VPC_STATUS_PENDING, "")

	region, err := vpc.GetRegion()
	if err != nil {
		self.taskFailed(ctx, vpc, errors.Wrapf(err, "GetRegion"))
		return
	}
	self.SetStage("OnCreateVpcComplete", nil)
	err = region.GetDriver().RequestCreateVpc(ctx, self.UserCred, region, vpc, self)
	if err != nil {
		self.taskFailed(ctx, vpc, errors.Wrapf(err, "RequestCreateVpc"))
		return
	}
}

func (self *VpcCreateTask) OnCreateVpcComplete(ctx context.Context, vpc *models.SVpc, data jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(self, vpc, logclient.ACT_ALLOCATE, nil, self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    vpc,
		Action: notifyclient.ActionCreate,
	})
	self.SetStageComplete(ctx, nil)
}

func (self *VpcCreateTask) OnCreateVpcCompleteFailed(ctx context.Context, vpc *models.SVpc, data jsonutils.JSONObject) {
	self.taskFailed(ctx, vpc, errors.Errorf(data.String()))
}
