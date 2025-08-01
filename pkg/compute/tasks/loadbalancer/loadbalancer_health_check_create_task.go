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
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LoadbalancerHealthCheckCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerHealthCheckCreateTask{})
}

func (self *LoadbalancerHealthCheckCreateTask) taskFail(ctx context.Context, hc *models.SLoadbalancerHealthCheck, err error) {
	hc.SetStatus(ctx, self.GetUserCred(), api.LB_CREATE_FAILED, err.Error())
	db.OpsLog.LogEvent(hc, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, hc, logclient.ACT_CREATE, err, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, hc.Id, hc.Name, api.LB_CREATE_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *LoadbalancerHealthCheckCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	hc := obj.(*models.SLoadbalancerHealthCheck)

	iRegion, err := hc.GetIRegion(ctx)
	if err != nil {
		self.taskFail(ctx, hc, errors.Wrapf(err, "GetIRegion"))
		return
	}
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

	iHc, err := iRegion.CreateILoadBalancerHealthCheck(opts)
	if err != nil {
		self.taskFail(ctx, hc, errors.Wrapf(err, "CreateILoadBalancerHealthCheck"))
		return
	}
	_, err = db.Update(hc, func() error {
		hc.ExternalId = iHc.GetGlobalId()
		hc.Status = apis.STATUS_AVAILABLE
		return nil
	})
	if err != nil {
		self.taskFail(ctx, hc, errors.Wrapf(err, "Update"))
	}
	self.SetStageComplete(ctx, nil)

}
