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

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LoadbalancerCertificateCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerCertificateCreateTask{})
}

func (self *LoadbalancerCertificateCreateTask) taskFail(ctx context.Context, lbcert *models.SLoadbalancerCertificate, err error) {
	lbcert.SetStatus(ctx, self.GetUserCred(), api.LB_CREATE_FAILED, err.Error())
	db.OpsLog.LogEvent(lbcert, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
	logclient.AddActionLogWithStartable(self, lbcert, logclient.ACT_CREATE, err, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, lbcert.Id, lbcert.Name, api.LB_CREATE_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *LoadbalancerCertificateCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbcert := obj.(*models.SLoadbalancerCertificate)
	region, err := lbcert.GetRegion()
	if err != nil {
		self.taskFail(ctx, lbcert, errors.Wrapf(err, "GetRegion"))
		return
	}
	self.SetStage("OnLoadbalancerCertificateCreateComplete", nil)
	err = region.GetDriver().RequestCreateLoadbalancerCertificate(ctx, self.GetUserCred(), lbcert, self)
	if err != nil {
		self.taskFail(ctx, lbcert, errors.Wrapf(err, "RequestCreateLoadbalancerCertificate"))
	}
}

func (self *LoadbalancerCertificateCreateTask) OnLoadbalancerCertificateCreateComplete(ctx context.Context, lbcert *models.SLoadbalancerCertificate, data jsonutils.JSONObject) {
	lbcert.SetStatus(ctx, self.GetUserCred(), apis.STATUS_AVAILABLE, "")
	db.OpsLog.LogEvent(lbcert, db.ACT_ALLOCATE, lbcert.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lbcert, logclient.ACT_CREATE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerCertificateCreateTask) OnLoadbalancerCertificateCreateCompleteFailed(ctx context.Context, lbcert *models.SLoadbalancerCertificate, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbcert, errors.Errorf(reason.String()))
}
