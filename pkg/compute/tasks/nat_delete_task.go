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
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type NatGatewayDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(NatGatewayDeleteTask{})
}

func (self *NatGatewayDeleteTask) taskFailed(ctx context.Context, nat *models.SNatGateway, err error) {
	nat.SetStatus(ctx, self.UserCred, api.NAT_STATUS_DELETE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, nat, logclient.ACT_DELOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *NatGatewayDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	nat := obj.(*models.SNatGateway)

	iNat, err := nat.GetINatGateway(ctx)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.OnEipDissociateComplete(ctx, nat, nil)
			return
		}
		self.taskFailed(ctx, nat, errors.Wrapf(err, "nat.GetINatGateway"))
		return
	}

	dnat, err := iNat.GetINatDTable()
	if err != nil {
		self.taskFailed(ctx, nat, errors.Wrapf(err, "iNat.GetINatDTable"))
		return
	}
	for i := range dnat {
		err = dnat[i].Delete()
		if err != nil {
			self.taskFailed(ctx, nat, errors.Wrapf(err, "delete d entry %v", dnat[i]))
			return
		}
		cloudprovider.WaitDeleted(dnat[i], time.Second*5, time.Minute)
	}
	snat, err := iNat.GetINatSTable()
	if err != nil {
		self.taskFailed(ctx, nat, errors.Wrapf(err, "GetINatSTable"))
		return
	}
	for i := range snat {
		err = snat[i].Delete()
		if err != nil {
			self.taskFailed(ctx, nat, errors.Wrapf(err, "delete s entry %v", snat[i]))
			return
		}
		cloudprovider.WaitDeleted(snat[i], time.Second*5, time.Minute)
	}

	self.SetStage("OnEipDissociateComplete", nil)
	self.OnEipDissociateComplete(ctx, nat, nil)
}

func (self *NatGatewayDeleteTask) OnEipDissociateComplete(ctx context.Context, nat *models.SNatGateway, data jsonutils.JSONObject) {
	eips, err := nat.GetEips()
	if err != nil {
		self.taskFailed(ctx, nat, errors.Wrapf(err, "nat.GetEips"))
		return
	}
	if len(eips) > 0 {
		eips[0].StartEipDissociateTask(ctx, self.GetUserCred(), false, self.GetTaskId())
		return
	}
	self.doDeleteNatGateway(ctx, nat)
}

func (self *NatGatewayDeleteTask) OnEipDissociateCompleteFailed(ctx context.Context, nat *models.SNatGateway, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, nil)
}

func (self *NatGatewayDeleteTask) doDeleteNatGateway(ctx context.Context, nat *models.SNatGateway) {
	iNat, err := nat.GetINatGateway(ctx)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, nat)
			return
		}
		self.taskFailed(ctx, nat, errors.Wrapf(err, "nat.GetINatGateway"))
		return
	}

	err = iNat.Delete()
	if err != nil {
		self.taskFailed(ctx, nat, errors.Wrapf(err, "iNat.Delete"))
		return
	}

	err = cloudprovider.WaitDeleted(iNat, time.Second*5, time.Minute*3)
	if err != nil {
		self.taskFailed(ctx, nat, errors.Wrapf(err, "cloudprovider.WaitDeleted"))
		return
	}

	self.taskComplete(ctx, nat)
}

func (self *NatGatewayDeleteTask) taskComplete(ctx context.Context, nat *models.SNatGateway) {
	nat.RealDelete(ctx, self.GetUserCred())
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    nat,
		Action: notifyclient.ActionDelete,
	})
	self.SetStageComplete(ctx, nil)
}
