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

type SNatDEntryCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SNatDEntryCreateTask{})
}

func (self *SNatDEntryCreateTask) taskFailed(ctx context.Context, dnat *models.SNatDEntry, err error) {
	dnat.SetStatus(ctx, self.UserCred, api.NAT_STATUS_CREATE_FAILED, err.Error())
	db.OpsLog.LogEvent(dnat, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
	nat, _ := dnat.GetNatgateway()
	if nat != nil {
		logclient.AddActionLogWithStartable(self, nat, logclient.ACT_NAT_CREATE_DNAT, err, self.UserCred, false)
	}
	logclient.AddActionLogWithStartable(self, dnat, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *SNatDEntryCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	dnat := obj.(*models.SNatDEntry)

	eip, err := dnat.GetEip()
	if err != nil {
		self.taskFailed(ctx, dnat, errors.Wrapf(err, "snat.GetEip"))
		return
	}

	if len(eip.AssociateId) > 0 {
		self.OnAssociateEipComplete(ctx, dnat, body)
		return
	}

	nat, err := dnat.GetNatgateway()
	if err != nil {
		self.taskFailed(ctx, dnat, errors.Wrapf(err, "snat.GetNatgateway"))
		return
	}

	self.SetStage("OnAssociateEipComplete", nil)
	region, _ := nat.GetRegion()
	err = region.GetDriver().RequestAssociateEipForNAT(ctx, self.GetUserCred(), nat, eip, self)
	if err != nil {
		self.taskFailed(ctx, dnat, errors.Wrapf(err, "RequestBindIPToNatgateway"))
		return
	}
}

func (self *SNatDEntryCreateTask) OnAssociateEipCompleteFailed(ctx context.Context, dnat *models.SNatDEntry, reason jsonutils.JSONObject) {
	self.taskFailed(ctx, dnat, errors.Errorf(reason.String()))
}

func (self *SNatDEntryCreateTask) OnAssociateEipComplete(ctx context.Context, dnat *models.SNatDEntry, body jsonutils.JSONObject) {
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

	eip, err := dnat.GetEip()
	if err != nil {
		self.taskFailed(ctx, dnat, errors.Wrapf(err, "dnat.GetEip"))
		return
	}

	// construct a DNat RUle
	rule := cloudprovider.SNatDRule{
		Protocol:     dnat.IpProtocol,
		InternalIP:   dnat.InternalIP,
		InternalPort: dnat.InternalPort,
		ExternalIP:   dnat.ExternalIP,
		ExternalPort: dnat.ExternalPort,
		ExternalIPID: eip.ExternalId,
	}
	iDnat, err := iNat.CreateINatDEntry(rule)
	if err != nil {
		self.taskFailed(ctx, dnat, errors.Wrapf(err, "iNat.CreateINatDEntry"))
		return
	}

	err = db.SetExternalId(dnat, self.UserCred, iDnat.GetGlobalId())
	if err != nil {
		self.taskFailed(ctx, dnat, errors.Wrapf(err, "db.SetExternalId(%s)", iDnat.GetGlobalId()))
		return
	}

	err = cloudprovider.WaitStatus(iDnat, api.NAT_STAUTS_AVAILABLE, 10*time.Second, 5*time.Minute)
	if err != nil {
		self.taskFailed(ctx, dnat, errors.Wrapf(err, "cloudprovider.WaitStatus"))
		return
	}

	dnat.SetStatus(ctx, self.UserCred, api.NAT_STAUTS_AVAILABLE, "")
	logclient.AddActionLogWithStartable(self, nat, logclient.ACT_NAT_CREATE_DNAT, nil, self.UserCred, true)
	logclient.AddActionLogWithStartable(self, dnat, logclient.ACT_ALLOCATE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
