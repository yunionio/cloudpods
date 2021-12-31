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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type EipAssociateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(EipAssociateTask{})
}

func (self *EipAssociateTask) taskFail(ctx context.Context, eip *models.SElasticip, obj db.IStatusStandaloneModel, err error) {
	eip.SetStatus(self.UserCred, api.EIP_STATUS_ASSOCIATE_FAIL, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
	if obj != nil {
		db.StatusBaseSetStatus(obj, self.GetUserCred(), api.INSTANCE_ASSOCIATE_EIP_FAILED, err.Error())
		db.OpsLog.LogEvent(obj, db.ACT_EIP_ATTACH, err, self.GetUserCred())
		logclient.AddActionLogWithStartable(self, obj, logclient.ACT_EIP_ASSOCIATE, err, self.UserCred, false)
	}
	logclient.AddActionLogWithStartable(self, eip, logclient.ACT_VM_ASSOCIATE, err, self.UserCred, false)
}

func (self *EipAssociateTask) GetAssociateInput() (api.ElasticipAssociateInput, error) {
	input := api.ElasticipAssociateInput{}
	err := self.Params.Unmarshal(&input)
	if err != nil {
		return input, errors.Wrapf(err, "self.Params.Unmarshal")
	}
	return input, nil
}

func (self *EipAssociateTask) GetAssociateObj() (db.IStatusStandaloneModel, api.ElasticipAssociateInput, error) {
	input, err := self.GetAssociateInput()
	if err != nil {
		return nil, input, errors.Wrapf(err, "GetAssociateInput")
	}

	switch input.InstanceType {
	case api.EIP_ASSOCIATE_TYPE_SERVER:
		vmObj, err := models.GuestManager.FetchById(input.InstanceId)
		if err != nil {
			return nil, input, errors.Wrapf(err, "GuestManager.FetchById(%s)", input.InstanceId)
		}
		vm := vmObj.(*models.SGuest)
		input.InstanceExternalId = vm.ExternalId
		return vm, input, nil
	case api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY:
		natObj, err := models.NatGatewayManager.FetchById(input.InstanceId)
		if err != nil {
			return nil, input, errors.Wrapf(err, "NatGatewayManager.FetchById(%s)", input.InstanceId)
		}
		nat := natObj.(*models.SNatGateway)
		input.InstanceExternalId = nat.ExternalId
		return nat, input, nil
	case api.EIP_ASSOCIATE_TYPE_INSTANCE_GROUP:
		grpObj, err := models.GroupManager.FetchById(input.InstanceId)
		if err != nil {
			return nil, input, errors.Wrapf(err, "GroupManager.FetchById(%s)", input.InstanceId)
		}
		grp := grpObj.(*models.SGroup)
		return grp, input, nil
	default:
		return nil, input, fmt.Errorf("invalid instance type %s", input.InstanceType)
	}
}

func (self *EipAssociateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)

	region, err := eip.GetRegion()
	if err != nil {
		self.taskFail(ctx, eip, nil, errors.Wrapf(err, "eip.GetRegion"))
		return
	}

	ins, input, err := self.GetAssociateObj()
	if err != nil {
		self.taskFail(ctx, eip, nil, errors.Wrapf(err, "self.GetAssociateObj"))
		return
	}

	db.StatusBaseSetStatus(ins, self.GetUserCred(), api.INSTANCE_ASSOCIATE_EIP, "associate eip")

	self.SetStage("OnAssociateEipComplete", nil)
	err = region.GetDriver().RequestAssociatEip(ctx, self.UserCred, eip, input, ins, self)
	if err != nil {
		self.taskFail(ctx, eip, ins, errors.Wrapf(err, "RequestAssociatEip"))
		return
	}
}

func (self *EipAssociateTask) OnAssociateEipComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)

	ins, input, err := self.GetAssociateObj()
	if err == nil {
		switch input.InstanceType {
		case api.EIP_ASSOCIATE_TYPE_SERVER:
			server := ins.(*models.SGuest)
			server.StartSyncstatus(ctx, self.UserCred, "")
			logclient.AddActionLogWithStartable(self, eip, logclient.ACT_VM_ASSOCIATE, ins, self.UserCred, true)
		case api.EIP_ASSOCIATE_TYPE_NAT_GATEWAY:
			nat := ins.(*models.SNatGateway)
			nat.StartSyncstatus(ctx, self.UserCred, "")
			logclient.AddActionLogWithStartable(self, eip, logclient.ACT_NATGATEWAY_ASSOCIATE, ins, self.UserCred, true)
		case api.EIP_ASSOCIATE_TYPE_INSTANCE_GROUP:
			grp := ins.(*models.SGroup)
			grp.SetStatus(self.UserCred, "init", "success")
			logclient.AddActionLogWithStartable(self, eip, logclient.ACT_NATGATEWAY_ASSOCIATE, ins, self.UserCred, true)
		}
		logclient.AddActionLogWithStartable(self, ins, logclient.ACT_EIP_ASSOCIATE, nil, self.UserCred, true)
	}

	self.SetStageComplete(ctx, nil)
}

func (self *EipAssociateTask) OnAssociateEipCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)
	ins, _, _ := self.GetAssociateObj()
	self.taskFail(ctx, eip, ins, errors.Errorf(data.String()))
}
