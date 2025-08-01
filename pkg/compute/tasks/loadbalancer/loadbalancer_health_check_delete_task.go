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

package loadbalancer

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LoadbalancerHealthCheckDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerHealthCheckDeleteTask{})
}

func (self *LoadbalancerHealthCheckDeleteTask) taskFail(ctx context.Context, hc *models.SLoadbalancerHealthCheck, err error) {
	hc.SetStatus(ctx, self.GetUserCred(), api.LB_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(hc, db.ACT_DELOCATE_FAIL, err.Error(), self.UserCred)
	logclient.AddActionLogWithStartable(self, hc, logclient.ACT_DELOCATE, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *LoadbalancerHealthCheckDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	hc := obj.(*models.SLoadbalancerHealthCheck)
	if len(hc.ExternalId) == 0 {
		self.taskComplete(ctx, hc)
		return
	}

	iRegion, err := hc.GetIRegion(ctx)
	if err != nil {
		self.taskFail(ctx, hc, errors.Wrapf(err, "GetIRegion"))
		return
	}
	hcs, err := iRegion.GetILoadBalancerHealthChecks()
	if err != nil {
		self.taskFail(ctx, hc, errors.Wrapf(err, "GetLoadbalancerHealthChecks"))
		return
	}
	for i := range hcs {
		if hcs[i].GetGlobalId() == hc.ExternalId {
			err = hcs[i].Delete()
			if err != nil {
				self.taskFail(ctx, hc, errors.Wrapf(err, "Delete"))
				return
			}
			self.taskComplete(ctx, hc)
			return
		}
	}

	self.taskComplete(ctx, hc)
}

func (self *LoadbalancerHealthCheckDeleteTask) taskComplete(ctx context.Context, hc *models.SLoadbalancerHealthCheck) {
	hc.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}
