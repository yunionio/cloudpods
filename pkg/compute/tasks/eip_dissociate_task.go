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

func (self *EipDissociateTask) TaskFail(ctx context.Context, eip *models.SElasticip, msg string, vm *models.SGuest) {
	eip.SetStatus(self.UserCred, api.EIP_STATUS_READY, msg)
	self.SetStageFailed(ctx, msg)
	if vm != nil {
		vm.SetStatus(self.UserCred, api.VM_DISSOCIATE_EIP_FAILED, msg)
		db.OpsLog.LogEvent(vm, db.ACT_EIP_DETACH, msg, self.GetUserCred())
		logclient.AddActionLogWithStartable(self, vm, logclient.ACT_EIP_DISSOCIATE, msg, self.UserCred, false)
	}
}

func (self *EipDissociateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)

	server := eip.GetAssociateVM()
	if server != nil {

		if server.Status != api.VM_DISSOCIATE_EIP {
			server.SetStatus(self.UserCred, api.VM_DISSOCIATE_EIP, "dissociate eip")
		}

		extEip, err := eip.GetIEip()
		if err != nil && err != cloudprovider.ErrNotFound {
			msg := fmt.Sprintf("fail to find iEIP for eip %s", err)
			self.TaskFail(ctx, eip, msg, server)
			return
		}

		if err == nil && len(extEip.GetAssociationExternalId()) > 0 {
			err = extEip.Dissociate()
			if err != nil {
				msg := fmt.Sprintf("fail to remote dissociate eip %s", err)
				self.TaskFail(ctx, eip, msg, server)
				return
			}
		}

		err = eip.Dissociate(ctx, self.UserCred)
		if err != nil {
			msg := fmt.Sprintf("fail to local dissociate eip %s", err)
			self.TaskFail(ctx, eip, msg, server)
			return
		}

		eip.SetStatus(self.UserCred, api.EIP_STATUS_READY, "dissociate")

		server.StartSyncstatus(ctx, self.UserCred, "")
	}

	self.SetStageComplete(ctx, nil)

	autoDelete := jsonutils.QueryBoolean(self.GetParams(), "auto_delete", false)

	if eip.AutoDellocate.IsTrue() || autoDelete {
		eip.StartEipDeallocateTask(ctx, self.UserCred, "")
	}
}
