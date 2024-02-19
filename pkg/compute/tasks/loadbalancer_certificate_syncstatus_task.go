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

type LoadbalancerCertificateSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerCertificateSyncstatusTask{})
}

func (self *LoadbalancerCertificateSyncstatusTask) taskFail(ctx context.Context, lbcert *models.SLoadbalancerCertificate, err error) {
	lbcert.SetStatus(ctx, self.GetUserCred(), apis.STATUS_UNKNOWN, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *LoadbalancerCertificateSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbcert := obj.(*models.SLoadbalancerCertificate)
	region, err := lbcert.GetRegion()
	if err != nil {
		self.taskFail(ctx, lbcert, errors.Wrapf(err, "GetRegion"))
		return
	}
	self.SetStage("OnLoadbalancerCertificateSyncstatusComplete", nil)
	err = region.GetDriver().RequestLoadbalancerCertificateSyncstatus(ctx, self.GetUserCred(), lbcert, self)
	if err != nil {
		self.taskFail(ctx, lbcert, errors.Wrapf(err, "RequestLoadbalancerCertificateSyncstatus"))
	}
}

func (self *LoadbalancerCertificateSyncstatusTask) OnLoadbalancerCertificateSyncstatusComplete(ctx context.Context, lbcert *models.SLoadbalancerCertificate, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerCertificateSyncstatusTask) OnLoadbalancerCertificateSyncstatusCompleteFailed(ctx context.Context, lbcert *models.SLoadbalancerCertificate, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbcert, errors.Errorf(reason.String()))
}
