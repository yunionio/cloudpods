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

type SNatDEntryDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SNatDEntryDeleteTask{})
}

func (self *SNatDEntryDeleteTask) taskFailed(ctx context.Context, dnat *models.SNatDEntry, err error) {
	dnat.SetStatus(ctx, self.UserCred, api.NAT_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(dnat, db.ACT_DELOCATE_FAIL, err, self.UserCred)
	nat, _ := dnat.GetNatgateway()
	if nat != nil {
		logclient.AddActionLogWithStartable(self, nat, logclient.ACT_NAT_DELETE_DNAT, err, self.UserCred, false)
	}
	logclient.AddActionLogWithStartable(self, dnat, logclient.ACT_NAT_DELETE_DNAT, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *SNatDEntryDeleteTask) taskComplete(ctx context.Context, dnat *models.SNatDEntry) {
	nat, _ := dnat.GetNatgateway()
	if nat != nil {
		logclient.AddActionLogWithStartable(self, nat, logclient.ACT_NAT_DELETE_SNAT, dnat, self.UserCred, true)
	}
	dnat.RealDelete(ctx, self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *SNatDEntryDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	dnat := obj.(*models.SNatDEntry)

	if len(dnat.ExternalId) == 0 {
		self.taskComplete(ctx, dnat)
		return
	}

	nat, err := dnat.GetNatgateway()
	if err != nil {
		self.taskFailed(ctx, dnat, errors.Wrapf(err, "dnat.GetNatgateway"))
		return
	}
	iNat, err := nat.GetINatGateway(ctx)
	if err != nil {
		self.taskFailed(ctx, dnat, errors.Wrapf(err, "nat.GetINatGateway"))
		return
	}

	iDnat, err := iNat.GetINatDEntryById(dnat.ExternalId)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, dnat)
			return
		}
		self.taskFailed(ctx, dnat, errors.Wrapf(err, "iNat.GetINatDEntryByID(%s)", dnat.ExternalId))
		return
	}
	err = iDnat.Delete()
	if err != nil {
		self.taskFailed(ctx, dnat, errors.Wrapf(err, "iDnat.Delete"))
		return
	}

	err = cloudprovider.WaitDeleted(iDnat, 10*time.Second, 5*time.Minute)
	if err != nil {
		self.taskFailed(ctx, dnat, errors.Wrapf(err, "cloudprovider.WaitDeleted"))
		return
	}

	eip, _ := dnat.GetEip()
	if eip != nil {
		region, _ := nat.GetRegion()
		region.GetDriver().OnNatEntryDeleteComplete(ctx, self.UserCred, eip)
	}

	self.taskComplete(ctx, dnat)
}
