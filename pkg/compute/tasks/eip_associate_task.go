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
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type EipAssociateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(EipAssociateTask{})
}

func (self *EipAssociateTask) TaskFail(ctx context.Context, eip *models.SElasticip, msg string, vm *models.SGuest) {
	eip.SetStatus(self.UserCred, api.EIP_STATUS_READY, msg)
	self.SetStageFailed(ctx, msg)
	if vm != nil {
		vm.SetStatus(self.UserCred, api.VM_ASSOCIATE_EIP_FAILED, msg)
		db.OpsLog.LogEvent(vm, db.ACT_EIP_ATTACH, msg, self.GetUserCred())
		logclient.AddActionLogWithStartable(self, vm, logclient.ACT_EIP_ASSOCIATE, msg, self.UserCred, false)
	}
	logclient.AddActionLogWithStartable(self, eip, logclient.ACT_VM_ASSOCIATE, msg, self.UserCred, false)
}

func (self *EipAssociateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)

	instanceId, _ := self.Params.GetString("instance_id")
	server := models.GuestManager.FetchGuestById(instanceId)

	if server == nil {
		msg := fmt.Sprintf("fail to find server for instanceId %s", instanceId)
		self.TaskFail(ctx, eip, msg, nil)
		return
	}

	if server.Status != api.VM_ASSOCIATE_EIP {
		server.SetStatus(self.UserCred, api.VM_ASSOCIATE_EIP, "associate eip")
	}

	extEip, err := eip.GetIEip()
	if err != nil {
		msg := fmt.Sprintf("fail to find iEIP for eip %s", err)
		self.TaskFail(ctx, eip, msg, server)
		return
	}

	err = extEip.Associate(server.ExternalId)
	if err != nil {
		msg := fmt.Sprintf("fail to remote associate EIP %s", err)
		self.TaskFail(ctx, eip, msg, server)
		return
	}

	err = eip.AssociateVM(ctx, self.UserCred, server)
	if err != nil {
		msg := fmt.Sprintf("fail to local associate EIP %s", err)
		self.TaskFail(ctx, eip, msg, server)
		return
	}

	eip.SetStatus(self.UserCred, api.EIP_STATUS_READY, "associate")

	server.StartSyncstatus(ctx, self.UserCred, "")
	logclient.AddActionLogWithStartable(self, server, logclient.ACT_EIP_ASSOCIATE, nil, self.UserCred, true)
	logclient.AddActionLogWithStartable(self, eip, logclient.ACT_VM_ASSOCIATE, nil, self.UserCred, true)

	self.SetStageComplete(ctx, nil)
}
