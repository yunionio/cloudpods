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
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type EipAllocateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(EipAllocateTask{})
}

func (self *EipAllocateTask) onFailed(ctx context.Context, eip *models.SElasticip, reason string) {
	self.finalReleasePendingUsage(ctx)
	self.setGuestAllocateEipFailed(eip, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *EipAllocateTask) setGuestAllocateEipFailed(eip *models.SElasticip, reason string) {
	if eip != nil {
		db.OpsLog.LogEvent(eip, db.ACT_ALLOCATE_FAIL, reason, self.GetUserCred())
		logclient.AddActionLogWithStartable(self, eip, logclient.ACT_ALLOCATE, reason, self.UserCred, false)
	}
	if self.Params != nil && self.Params.Contains("instance_id") {
		instanceId, _ := self.Params.GetString("instance_id")
		instance, err := models.GuestManager.FetchById(instanceId)
		if err != nil {
			log.Errorf("failed to find guest by id: %s error: %v", instanceId, err)
			return
		}
		guest := instance.(*models.SGuest)
		guest.SetStatus(self.UserCred, api.VM_ASSOCIATE_EIP_FAILED, reason)
	}
}

func (self *EipAllocateTask) finalReleasePendingUsage(ctx context.Context) {
	pendingUsage := models.SQuota{}
	if err := self.GetPendingUsage(&pendingUsage); err == nil && !pendingUsage.IsEmpty() {
		if err := models.QuotaManager.CancelPendingUsage(ctx, self.UserCred, self.UserCred.GetProjectId(), nil, &pendingUsage); err != nil {
			log.Errorf("CancelPendingUsage error: %v", err)
		}
	}
}

func (self *EipAllocateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)

	iregion, err := eip.GetIRegion()
	if err != nil {
		msg := fmt.Sprintf("fail to find iregion for eip %s", err)
		eip.SetStatus(self.UserCred, api.EIP_STATUS_ALLOCATE_FAIL, msg)
		self.onFailed(ctx, eip, msg)
		return
	}

	extEip, err := iregion.CreateEIP(eip.Name, eip.Bandwidth, eip.ChargeType, eip.BgpType)
	if err != nil {
		msg := fmt.Sprintf("create eip fail %s", err)
		eip.SetStatus(self.UserCred, api.EIP_STATUS_ALLOCATE_FAIL, msg)
		self.onFailed(ctx, eip, msg)
		return
	}

	err = eip.SyncWithCloudEip(ctx, self.UserCred, eip.GetCloudprovider(), extEip, "")

	if err != nil {
		msg := fmt.Sprintf("sync eip fail %s", err)
		eip.SetStatus(self.UserCred, api.EIP_STATUS_ALLOCATE_FAIL, msg)
		self.onFailed(ctx, eip, msg)
		return
	}

	self.finalReleasePendingUsage(ctx)

	if self.Params != nil && self.Params.Contains("instance_id") {
		self.SetStage("on_eip_associate_complete", nil)
		err = eip.StartEipAssociateTask(ctx, self.UserCred, self.Params, "")
		if err != nil {
			msg := fmt.Sprintf("start associate task fail %s", err)
			self.SetStageFailed(ctx, msg)
		}
	} else {
		logclient.AddActionLogWithStartable(self, eip, logclient.ACT_ALLOCATE, nil, self.UserCred, true)
		self.SetStageComplete(ctx, nil)
	}
}

func (self *EipAllocateTask) OnEipAssociateComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)
	logclient.AddActionLogWithStartable(self, eip, logclient.ACT_ALLOCATE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *EipAllocateTask) OnEipAssociateCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)
	self.setGuestAllocateEipFailed(eip, data.String())
	self.SetStageFailed(ctx, data.String())
}
