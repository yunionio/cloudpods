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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type EipDissociateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(EipDissociateTask{})
}

func (self *EipDissociateTask) TaskFail(ctx context.Context, eip *models.SElasticip, msg string, model db.IModel) {
	eip.SetStatus(self.UserCred, api.EIP_STATUS_READY, msg)
	self.SetStageFailed(ctx, msg)
	var logOp string
	if model != nil {
		switch srv := model.(type) {
		case *models.SGuest:
			srv.SetStatus(self.UserCred, api.VM_DISSOCIATE_EIP_FAILED, msg)
			logOp = logclient.ACT_VM_DISSOCIATE
		case *models.SNatGateway:
			srv.SetStatus(self.UserCred, api.VM_DISSOCIATE_EIP_FAILED, msg)
			logOp = logclient.ACT_NATGATEWAY_DISSOCIATE
		case *models.SLoadbalancer:
			srv.SetStatus(self.UserCred, api.VM_DISSOCIATE_EIP_FAILED, msg)
			logOp = logclient.ACT_LOADBALANCER_DISSOCIATE
		}
		db.OpsLog.LogEvent(model, db.ACT_EIP_DETACH, msg, self.GetUserCred())
		logclient.AddActionLogWithStartable(self, model, logclient.ACT_EIP_DISSOCIATE, msg, self.UserCred, false)
		logclient.AddActionLogWithStartable(self, model, logOp, msg, self.UserCred, false)
	}
}

func (self *EipDissociateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)

	if eip.IsAssociated() {

		var (
			model db.IModel
			logOp string
		)

		if server := eip.GetAssociateVM(); server != nil {
			if server.Status != api.VM_DISSOCIATE_EIP {
				server.SetStatus(self.UserCred, api.VM_DISSOCIATE_EIP, "dissociate eip")
			}
			model = server
			logOp = logclient.ACT_VM_DISSOCIATE
		} else if lb := eip.GetAssociateLoadbalancer(); lb != nil {
			model = lb
			logOp = logclient.ACT_LOADBALANCER_DISSOCIATE
		} else if nat := eip.GetAssociateNatGateway(); nat != nil {
			model = nat
			logOp = logclient.ACT_NATGATEWAY_DISSOCIATE
		} else {
			self.TaskFail(ctx, eip, "unsupported associate type", nil)
			return
		}

		extEip, err := eip.GetIEip()
		if err != nil && err != cloudprovider.ErrNotFound {
			msg := fmt.Sprintf("fail to find iEIP for eip %s", err)
			self.TaskFail(ctx, eip, msg, model)
			return
		}

		if err == nil && len(extEip.GetAssociationExternalId()) > 0 {
			err = extEip.Dissociate()
			if err != nil {
				msg := fmt.Sprintf("fail to remote dissociate eip %s", err)
				self.TaskFail(ctx, eip, msg, model)
				return
			}
		}

		err = eip.Dissociate(ctx, self.UserCred)
		if err != nil {
			msg := fmt.Sprintf("fail to local dissociate eip %s", err)
			self.TaskFail(ctx, eip, msg, model)
			return
		}

		eip.SetStatus(self.UserCred, api.EIP_STATUS_READY, "dissociate")

		logclient.AddActionLogWithStartable(self, model, logclient.ACT_EIP_DISSOCIATE, nil, self.UserCred, true)
		logclient.AddActionLogWithStartable(self, eip, logOp, nil, self.UserCred, true)

		switch srv := model.(type) {
		case *models.SGuest:
			srv.StartSyncstatus(ctx, self.UserCred, "")
		}
	}

	self.SetStageComplete(ctx, nil)

	autoDelete := jsonutils.QueryBoolean(self.GetParams(), "auto_delete", false)

	if eip.AutoDellocate.IsTrue() || autoDelete {
		eip.StartEipDeallocateTask(ctx, self.UserCred, "")
	}
}
