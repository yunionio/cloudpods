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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestDissociateEipTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestDissociateEipTask{})
}

func (self *GuestDissociateEipTask) TaskFail(ctx context.Context, guest *models.SGuest, eip *models.SElasticip, err error) {
	guest.SetStatus(self.UserCred, api.VM_DISSOCIATE_EIP_FAILED, err.Error())
	if eip != nil {
		eip.SetStatus(self.UserCred, api.EIP_STATUS_READY, err.Error())
		logclient.AddActionLogWithStartable(self, eip, logclient.ACT_VM_DISSOCIATE, err, self.UserCred, false)
	}
	self.SetStageFailed(ctx, err.Error())
	db.OpsLog.LogEvent(guest, db.ACT_EIP_DETACH, err.Error(), self.GetUserCred())
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_EIP_DISSOCIATE, err, self.UserCred, false)
}

func (self *GuestDissociateEipTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	eip, err := guest.GetEip()
	if err != nil {
		self.TaskFail(ctx, guest, nil, errors.Wrap(err, "guest.GetEip"))
		return
	}

	extEip, err := eip.GetIEip()
	if err != nil && errors.Cause(err) != cloudprovider.ErrNotFound {
		self.TaskFail(ctx, guest, eip, errors.Wrap(err, "eip.GetIEip"))
		return
	}

	if err == nil && len(extEip.GetAssociationExternalId()) > 0 {
		err = extEip.Dissociate()
		if err != nil {
			self.TaskFail(ctx, guest, eip, errors.Wrap(err, "extEip.Dissociate"))
			return
		}
	}

	err = eip.Dissociate(ctx, self.UserCred)
	if err != nil {
		self.TaskFail(ctx, guest, eip, errors.Wrap(err, "eip.Dissociate"))
		return
	}

	eip.SetStatus(self.UserCred, api.EIP_STATUS_READY, "dissociate")
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_EIP_DISSOCIATE, nil, self.UserCred, true)
	logclient.AddActionLogWithStartable(self, eip, logclient.ACT_VM_DISSOCIATE, nil, self.UserCred, true)

	guest.StartSyncstatus(ctx, self.UserCred, "")
	self.SetStageComplete(ctx, nil)

	autoDelete := jsonutils.QueryBoolean(self.GetParams(), "auto_delete", false)

	if eip.AutoDellocate.IsTrue() || autoDelete {
		eip.StartEipDeallocateTask(ctx, self.UserCred, "")
	}

}
