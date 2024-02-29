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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SNatSEntryDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SNatSEntryDeleteTask{})
}

func (self *SNatSEntryDeleteTask) taskFailed(ctx context.Context, snat *models.SNatSEntry, err error) {
	snat.SetStatus(ctx, self.UserCred, api.NAT_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(snat, db.ACT_DELOCATE_FAIL, err, self.UserCred)
	nat, _ := snat.GetNatgateway()
	if nat != nil {
		logclient.AddActionLogWithStartable(self, nat, logclient.ACT_NAT_DELETE_SNAT, err, self.UserCred, false)
	}
	logclient.AddActionLogWithStartable(self, snat, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *SNatSEntryDeleteTask) taskComplete(ctx context.Context, snat *models.SNatSEntry) {
	nat, _ := snat.GetNatgateway()
	if nat != nil {
		logclient.AddActionLogWithStartable(self, nat, logclient.ACT_NAT_DELETE_SNAT, snat, self.UserCred, true)
	}
	snat.RealDelete(ctx, self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *SNatSEntryDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	snat := obj.(*models.SNatSEntry)

	if len(snat.ExternalId) == 0 {
		self.taskComplete(ctx, snat)
		return
	}

	nat, err := snat.GetNatgateway()
	if err != nil {
		self.taskFailed(ctx, snat, errors.Wrapf(err, "snat.GetNatgateway"))
		return
	}

	iNat, err := nat.GetINatGateway(ctx)
	if err != nil {
		self.taskFailed(ctx, snat, errors.Wrapf(err, "nat.GetINatGateway"))
		return
	}

	iSnat, err := iNat.GetINatSEntryById(snat.ExternalId)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, snat)
			return
		}
		self.taskFailed(ctx, snat, errors.Wrapf(err, "iNat.GetINatSEntryByID(%s)", snat.ExternalId))
		return
	}

	err = iSnat.Delete()
	if err != nil {
		self.taskFailed(ctx, snat, errors.Wrapf(err, "iSnat.Delete"))
		return
	}

	err = cloudprovider.WaitDeleted(iSnat, 10*time.Second, 5*time.Minute)
	if err != nil {
		self.taskFailed(ctx, snat, errors.Wrapf(err, "cloudprovider.WaitDeleted"))
		return
	}

	eip, _ := snat.GetEip()
	if eip != nil {
		region, _ := nat.GetRegion()
		region.GetDriver().OnNatEntryDeleteComplete(ctx, self.UserCred, eip)
	}

	self.taskComplete(ctx, snat)
}
