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

	"yunion.io/x/cloudmux/pkg/apis"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type LoadbalancerHealthCheckSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerHealthCheckSyncstatusTask{})
}

func (self *LoadbalancerHealthCheckSyncstatusTask) taskFail(ctx context.Context, hc *models.SLoadbalancerHealthCheck, err error) {
	hc.SetStatus(ctx, self.GetUserCred(), apis.STATUS_UNKNOWN, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *LoadbalancerHealthCheckSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	hc := obj.(*models.SLoadbalancerHealthCheck)
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
			err := hc.SyncWithCloudLoadbalancerHealthCheck(ctx, self.GetUserCred(), hcs[i], hc.GetCloudprovider())
			if err != nil {
				self.taskFail(ctx, hc, errors.Wrapf(err, "SyncWithCloudLoadbalancerHealthCheck"))
				return
			}
			self.taskComplete(ctx, hc)
			return
		}
	}

	self.taskFail(ctx, hc, errors.Wrapf(cloudprovider.ErrNotFound, "LoadbalancerHealthCheck not found"))
}

func (self *LoadbalancerHealthCheckSyncstatusTask) taskComplete(ctx context.Context, hc *models.SLoadbalancerHealthCheck) {
	hc.SetStatus(ctx, self.GetUserCred(), apis.STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}
