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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type VpcCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(VpcCreateTask{})
}

func (self *VpcCreateTask) TaskFailed(ctx context.Context, vpc *models.SVpc, err error) {
	log.Errorf("vpc create task fail: %s", err)
	vpc.SetStatus(self.UserCred, api.VPC_STATUS_FAILED, err.Error())
	db.OpsLog.LogEvent(vpc, db.ACT_ALLOCATE_FAIL, err.Error(), self.UserCred)
	logclient.AddActionLogWithStartable(self, vpc, logclient.ACT_ALLOCATE, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, err.Error())
}

func (self *VpcCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	vpc := obj.(*models.SVpc)
	vpc.SetStatus(self.UserCred, api.VPC_STATUS_PENDING, "")

	region, err := vpc.GetRegion()
	if err != nil {
		self.TaskFailed(ctx, vpc, errors.Wrap(err, "vpc.GetRegion"))
		return
	}
	self.SetStage("OnCreateVpcComplete", nil)
	err = region.GetDriver().RequestCreateVpc(ctx, self.UserCred, region, vpc, self)
	if err != nil {
		self.TaskFailed(ctx, vpc, err)
		return
	}
}

func (self *VpcCreateTask) OnCreateVpcComplete(ctx context.Context, vpc *models.SVpc, data jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(self, vpc, logclient.ACT_ALLOCATE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *VpcCreateTask) OnCreateVpcCompleteFailed(ctx context.Context, vpc *models.SVpc, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, vpc, fmt.Errorf("%s", data.String()))
}
