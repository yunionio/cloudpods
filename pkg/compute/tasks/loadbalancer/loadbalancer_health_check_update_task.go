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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LoadbalancerHealthCheckUpdateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerHealthCheckUpdateTask{})
}

func (self *LoadbalancerHealthCheckUpdateTask) taskFail(ctx context.Context, hc *models.SLoadbalancerHealthCheck, err error) {
	hc.SetStatus(ctx, self.GetUserCred(), api.LB_SYNC_CONF_FAILED, err.Error())
	db.OpsLog.LogEvent(hc, db.ACT_SYNC_CONF, err.Error(), self.UserCred)
	logclient.AddActionLogWithStartable(self, hc, logclient.ACT_SYNC_CONF, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *LoadbalancerHealthCheckUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
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
			opts := &cloudprovider.SLoadbalancerHealthCheck{
				Name: hc.Name,

				HealthCheckType:     hc.HealthCheckType,
				HealthCheckDomain:   hc.HealthCheckDomain,
				HealthCheckURI:      hc.HealthCheckURI,
				HealthCheckHttpCode: hc.HealthCheckHttpCode,
				HealthCheckMethod:   hc.HealthCheckMethod,
				HealthCheckPort:     hc.HealthCheckPort,
				HealthCheckTimeout:  hc.HealthCheckTimeout,
				HealthCheckInterval: hc.HealthCheckInterval,
				HealthCheckRise:     hc.HealthCheckRise,
				HealthCheckReq:      hc.HealthCheckReq,
				HealthCheckExp:      hc.HealthCheckExp,
			}
			err := hcs[i].Update(ctx, opts)
			if err != nil {
				self.taskFail(ctx, hc, errors.Wrapf(err, "Update"))
				return
			}
			self.taskComplete(ctx, hc)
			return
		}
	}

	self.taskFail(ctx, hc, errors.Wrapf(errors.ErrNotFound, "LoadbalancerHealthCheck %s not found", hc.ExternalId))
}

func (self *LoadbalancerHealthCheckUpdateTask) taskComplete(ctx context.Context, hc *models.SLoadbalancerHealthCheck) {
	hc.SetStatus(ctx, self.GetUserCred(), apis.STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}
