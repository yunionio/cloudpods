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

type SNatSEntryCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SNatSEntryCreateTask{})
}

func (self *SNatSEntryCreateTask) taskFailed(ctx context.Context, snat *models.SNatSEntry, err error) {
	snat.SetStatus(ctx, self.UserCred, api.NAT_STATUS_CREATE_FAILED, err.Error())
	db.OpsLog.LogEvent(snat, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
	nat, _ := snat.GetNatgateway()
	if nat != nil {
		logclient.AddActionLogWithStartable(self, nat, logclient.ACT_NAT_CREATE_SNAT, err, self.UserCred, false)
	}
	logclient.AddActionLogWithStartable(self, snat, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *SNatSEntryCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	snat := obj.(*models.SNatSEntry)

	eip, err := snat.GetEip()
	if err != nil {
		self.taskFailed(ctx, snat, errors.Wrapf(err, "snat.GetEip"))
		return
	}

	if len(eip.AssociateId) > 0 {
		self.OnAssociateEipComplete(ctx, snat, body)
		return
	}

	nat, err := snat.GetNatgateway()
	if err != nil {
		self.taskFailed(ctx, snat, errors.Wrapf(err, "snat.GetNatgateway"))
		return
	}

	self.SetStage("OnAssociateEipComplete", nil)
	region, _ := nat.GetRegion()
	err = region.GetDriver().RequestAssociateEipForNAT(ctx, self.GetUserCred(), nat, eip, self)
	if err != nil {
		self.taskFailed(ctx, snat, errors.Wrapf(err, "RequestAssociateEipForNAT"))
		return
	}
}

func (self *SNatSEntryCreateTask) OnAssociateEipCompleteFailed(ctx context.Context, snatEntry *models.SNatSEntry, reason jsonutils.JSONObject) {
	self.taskFailed(ctx, snatEntry, errors.Errorf(reason.String()))
}

func (self *SNatSEntryCreateTask) OnAssociateEipComplete(ctx context.Context, snat *models.SNatSEntry, body jsonutils.JSONObject) {
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

	eip, err := snat.GetEip()
	if err != nil {
		self.taskFailed(ctx, snat, errors.Wrapf(err, "snat.GetEip"))
		return
	}

	// construct a DNat RUle
	rule := cloudprovider.SNatSRule{
		ExternalIP:   snat.IP,
		ExternalIPID: eip.ExternalId,
	}
	if len(snat.SourceCIDR) > 0 {
		rule.SourceCIDR = snat.SourceCIDR
	} else {
		network, err := snat.GetNetwork()
		if err != nil {
			self.taskFailed(ctx, snat, errors.Wrapf(err, "snat.GetNetwork"))
			return
		}
		rule.NetworkID = network.ExternalId
	}
	iSnat, err := iNat.CreateINatSEntry(rule)
	if err != nil {
		self.taskFailed(ctx, snat, errors.Wrapf(err, "CreateINatSEntry"))
		return
	}

	err = db.SetExternalId(snat, self.UserCred, iSnat.GetGlobalId())
	if err != nil {
		self.taskFailed(ctx, snat, errors.Wrapf(err, "db.SetExternalId(%s)", iSnat.GetGlobalId()))
		return
	}

	err = cloudprovider.WaitStatus(iSnat, api.NAT_STAUTS_AVAILABLE, 10*time.Second, 5*time.Minute)
	if err != nil {
		self.taskFailed(ctx, snat, errors.Wrapf(err, "cloudprovider.WaitStatus(iSnat)"))
		return
	}

	snat.SetStatus(ctx, self.UserCred, api.NAT_STAUTS_AVAILABLE, "")
	logclient.AddActionLogWithStartable(self, nat, logclient.ACT_NAT_CREATE_SNAT, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
