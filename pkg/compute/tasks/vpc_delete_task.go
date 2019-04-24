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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type VpcDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(VpcDeleteTask{})
}

func (self *VpcDeleteTask) taskFailed(ctx context.Context, vpc *models.SVpc, err error) {
	log.Errorf("vpc delete task fail: %s", err)
	vpc.SetStatus(self.UserCred, api.VPC_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(vpc, db.ACT_DELOCATE_FAIL, err.Error(), self.UserCred)
	self.SetStageFailed(ctx, err.Error())
}

func (self *VpcDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	vpc := obj.(*models.SVpc)

	vpc.SetStatus(self.UserCred, api.VPC_STATUS_DELETING, "")
	db.OpsLog.LogEvent(vpc, db.ACT_DELOCATING, vpc.GetShortDesc(ctx), self.UserCred)

	region, err := vpc.GetIRegion()
	if err != nil {
		self.taskFailed(ctx, vpc, err)
		return
	}
	ivpc, err := region.GetIVpcById(vpc.GetExternalId())
	if ivpc != nil {
		err = ivpc.Delete()
		if err != nil {
			self.taskFailed(ctx, vpc, err)
			return
		}
		err = cloudprovider.WaitDeleted(ivpc, 10*time.Second, 300*time.Second)
		if err != nil {
			self.taskFailed(ctx, vpc, err)
			return
		}
	} else if err == cloudprovider.ErrNotFound {
		// already deleted, do nothing
	} else {
		self.taskFailed(ctx, vpc, err)
		return
	}

	err = vpc.Purge(ctx, self.UserCred)
	if err != nil {
		self.taskFailed(ctx, vpc, err)
		return
	}

	self.SetStageComplete(ctx, nil)
}
