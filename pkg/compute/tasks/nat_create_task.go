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
	"yunion.io/x/pkg/util/billing"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type NatGatewayCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(NatGatewayCreateTask{})
}

func (self *NatGatewayCreateTask) taskFailed(ctx context.Context, nat *models.SNatGateway, err error) {
	nat.SetStatus(ctx, self.UserCred, api.NAT_STATUS_CREATE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, nat, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *NatGatewayCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	nat := obj.(*models.SNatGateway)

	opts := cloudprovider.NatGatewayCreateOptions{
		Name:    nat.Name,
		Desc:    nat.Description,
		NatSpec: nat.NatSpec,
	}

	vpc, err := nat.GetVpc()
	if err != nil {
		self.taskFailed(ctx, nat, errors.Wrapf(err, "nat.GetVpc"))
		return
	}

	opts.VpcId = vpc.ExternalId

	if len(nat.NetworkId) > 0 {
		_network, err := models.NetworkManager.FetchById(nat.NetworkId)
		if err != nil {
			self.taskFailed(ctx, nat, errors.Wrapf(err, "NetworkManager.FetchById(%s)", nat.NetworkId))
			return
		}
		network := _network.(*models.SNetwork)
		opts.NetworkId = network.ExternalId
	}

	if nat.BillingType == billing_api.BILLING_TYPE_PREPAID {
		bc, err := billing.ParseBillingCycle(nat.BillingCycle)
		if err != nil {
			self.taskFailed(ctx, nat, errors.Wrapf(err, "ParseBillingCycle(%s)", nat.BillingCycle))
			return
		}
		bc.AutoRenew = nat.AutoRenew
		opts.BillingCycle = &bc
	}

	self.SetStage("OnCreateNatGatewayCreateComplete", nil)
	taskman.LocalTaskRun(self, func() (jsonutils.JSONObject, error) {
		iVpc, err := vpc.GetIVpc(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "vpc.GetIVpc")
		}

		iNat, err := iVpc.CreateINatGateway(&opts)
		if err != nil {
			return nil, errors.Wrapf(err, "iVpc.CreateINatGateway")
		}
		err = db.SetExternalId(nat, self.GetUserCred(), iNat.GetGlobalId())
		if err != nil {
			return nil, errors.Wrapf(err, "db.SetExternalId")
		}

		err = cloudprovider.WaitStatus(iNat, api.NAT_STAUTS_AVAILABLE, time.Second*5, time.Minute*10)
		if err != nil {
			return nil, errors.Wrapf(err, "cloudprovider.WaitStatus")
		}

		nat.SyncWithCloudNatGateway(ctx, self.GetUserCred(), nat.GetCloudprovider(), iNat)
		return nil, nil
	})
}

func (self *NatGatewayCreateTask) OnCreateNatGatewayCreateComplete(ctx context.Context, nat *models.SNatGateway, body jsonutils.JSONObject) {
	input := api.NatgatewayCreateInput{}
	self.GetParams().Unmarshal(&input)
	if len(input.Eip) > 0 || input.EipBw > 0 {
		self.SetStage("OnDeployEipComplete", nil)

		var eip *models.SElasticip = nil
		var err error
		if len(input.Eip) > 0 {
			eipObj, err := models.ElasticipManager.FetchById(input.Eip)
			if err != nil {
				self.OnDeployEipCompleteFailed(ctx, nat, jsonutils.NewString(errors.Wrapf(err, "ElasticipManager.FetchById(%s)", input.Eip).Error()))
				return
			}
			eip = eipObj.(*models.SElasticip)
		} else {
			pendingRegionUsage := models.SRegionQuota{}
			self.GetPendingUsage(&pendingRegionUsage, 1)
			self.SetPendingUsage(&pendingRegionUsage, 1)

			eip, err = models.ElasticipManager.NewEipForVMOnHost(ctx, self.UserCred, &models.NewEipForVMOnHostArgs{
				Bandwidth:     input.EipBw,
				BgpType:       input.EipBgpType,
				ChargeType:    input.EipChargeType,
				AutoDellocate: input.EipAutoDellocate,

				Natgateway:   nat,
				PendingUsage: &pendingRegionUsage,
			})
			self.SetPendingUsage(&pendingRegionUsage, 1)
			if err != nil {
				self.OnDeployEipCompleteFailed(ctx, nat, jsonutils.NewString(errors.Wrapf(err, "ElasticipManager.NewEipForVMOnHost").Error()))
				return
			}
		}

		opts := api.ElasticipAssociateInput{
			InstanceId:         nat.Id,
			InstanceExternalId: nat.ExternalId,
			InstanceType:       api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY,
		}
		if input.EipBw > 0 {
			// newly allocated eip, need allocation and associate
			err = eip.AllocateAndAssociateInstance(ctx, self.UserCred, nat, opts, self.GetId())
			err = errors.Wrap(err, "AllocateAndAssociateVM")
		} else {
			err = eip.StartEipAssociateInstanceTask(ctx, self.UserCred, opts, self.GetId())
			err = errors.Wrap(err, "StartEipAssociateInstanceTask")
		}
		if err != nil {
			self.OnDeployEipCompleteFailed(ctx, nat, jsonutils.NewString(err.Error()))
			return
		}

		return
	}
	self.OnDeployEipComplete(ctx, nat, nil)
}

func (self *NatGatewayCreateTask) OnCreateNatGatewayCreateCompleteFailed(ctx context.Context, nat *models.SNatGateway, body jsonutils.JSONObject) {
	self.taskFailed(ctx, nat, errors.Errorf(body.String()))
}

func (self *NatGatewayCreateTask) OnDeployEipCompleteFailed(ctx context.Context, nat *models.SNatGateway, data jsonutils.JSONObject) {
	nat.SetStatus(ctx, self.UserCred, api.INSTANCE_ASSOCIATE_EIP_FAILED, data.String())
	db.OpsLog.LogEvent(nat, db.ACT_EIP_ATTACH, data, self.UserCred)
	logclient.AddActionLogWithStartable(self, nat, logclient.ACT_EIP_ASSOCIATE, data, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, nat.Id, nat.Name, api.INSTANCE_ASSOCIATE_EIP_FAILED, data.String())
	self.SetStageFailed(ctx, data)
}

func (self *NatGatewayCreateTask) OnDeployEipComplete(ctx context.Context, nat *models.SNatGateway, data jsonutils.JSONObject) {
	self.SetStage("OnSyncstatusComplete", nil)
	nat.StartSyncstatus(ctx, self.GetUserCred(), self.GetTaskId())
}

func (self *NatGatewayCreateTask) OnSyncstatusComplete(ctx context.Context, nat *models.SNatGateway, data jsonutils.JSONObject) {
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    nat,
		Action: notifyclient.ActionCreate,
	})
	self.SetStageComplete(ctx, nil)
}

func (self *NatGatewayCreateTask) OnSyncstatusCompleteFailed(ctx context.Context, nat *models.SNatGateway, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
