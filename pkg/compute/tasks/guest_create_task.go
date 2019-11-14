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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestCreateTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestCreateTask{})
}

func (self *GuestCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, api.VM_CREATE_NETWORK, "")
	self.SetStage("on_wait_guest_networks_ready", nil)
	self.OnWaitGuestNetworksReady(ctx, obj, nil)
}

func (self *GuestCreateTask) OnWaitGuestNetworksReady(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if !guest.IsNetworkAllocated() {
		log.Infof("Guest %s network not ready!!", guest.Name)
		time.Sleep(time.Second * 2)
		self.ScheduleRun(nil)
	} else {
		self.OnGuestNetworkReady(ctx, guest)
	}
}

func (self *GuestCreateTask) OnGuestNetworkReady(ctx context.Context, guest *models.SGuest) {
	guest.SetStatus(self.UserCred, api.VM_CREATE_DISK, "")
	self.SetStage("OnDiskPrepared", nil)
	guest.GetDriver().RequestGuestCreateAllDisks(ctx, guest, self)
}

func (self *GuestCreateTask) OnDiskPreparedFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, api.VM_DISK_FAILED, "allocation failed")
	db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE_FAIL, data, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_ALLOCATE, data, self.UserCred, false)
	notifyclient.NotifySystemError(guest.Id, guest.Name, api.VM_DISK_FAILED, data.String())
	self.SetStageFailed(ctx, data.String())
}

func (self *GuestCreateTask) OnDiskPrepared(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	cdrom, _ := self.Params.GetString("cdrom")
	if len(cdrom) > 0 {
		self.SetStage("OnCdromPrepared", nil)
		guest.GetDriver().RequestGuestCreateInsertIso(ctx, cdrom, guest, self)
	} else {
		self.OnCdromPrepared(ctx, obj, data)
	}
}

func (self *GuestCreateTask) OnCdromPrepared(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	log.Infof("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")
	log.Infof("DEPLOY GUEST %s", guest.Name)
	log.Infof("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")
	guest.SetStatus(self.UserCred, api.VM_DEPLOYING, "")
	self.StartDeployGuest(ctx, guest)
}

func (self *GuestCreateTask) OnCdromPreparedFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, api.VM_DISK_FAILED, "")
	db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE_FAIL, data, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_ALLOCATE, data, self.UserCred, false)
	notifyclient.NotifySystemError(guest.Id, guest.Name, api.VM_DISK_FAILED, fmt.Sprintf("cdrom_failed %s", data))
	self.SetStageFailed(ctx, fmt.Sprintf("cdrom_failed %s", data))
}

func (self *GuestCreateTask) StartDeployGuest(ctx context.Context, guest *models.SGuest) {
	self.SetStage("OnDeployGuestDescComplete", nil)
	guest.StartGuestDeployTask(ctx, self.UserCred, self.Params, "create", self.GetId())
}

func (self *GuestCreateTask) OnDeployGuestDescComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.SetStage("OnDeployEipComplete", nil)
	self.StartEipSubTask(ctx, guest)
}

func (self *GuestCreateTask) notifyServerCreated(ctx context.Context, guest *models.SGuest) {
	guest.NotifyServerEvent(
		self.UserCred, notifyclient.SERVER_CREATED,
		notify.NotifyPriorityImportant, true, nil, false,
	)
	guest.NotifyAdminServerEvent(ctx, notifyclient.SERVER_CREATED_ADMIN, notify.NotifyPriorityImportant)
}

func (self *GuestCreateTask) OnDeployGuestDescCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, api.VM_DEPLOY_FAILED, "deploy_failed")
	db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE_FAIL, data, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_ALLOCATE, data, self.UserCred, false)
	notifyclient.NotifySystemError(guest.Id, guest.Name, api.VM_DEPLOY_FAILED, data.String())
	self.SetStageFailed(ctx, data.String())
}

func (self *GuestCreateTask) OnDeployEipComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE, nil, self.UserCred)
	if !guest.IsSystem {
		self.notifyServerCreated(ctx, guest)
	}

	guest.GetDriver().OnGuestCreateTaskComplete(ctx, guest, self)
}

func (self *GuestCreateTask) OnDeployEipCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, api.VM_ASSOCIATE_EIP_FAILED, "deploy_failed")
	db.OpsLog.LogEvent(guest, db.ACT_EIP_ATTACH, data, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_EIP_ASSOCIATE, data, self.UserCred, false)
	notifyclient.NotifySystemError(guest.Id, guest.Name, api.VM_ASSOCIATE_EIP_FAILED, data.String())
	self.SetStageFailed(ctx, data.String())
}

func (self *GuestCreateTask) OnAutoStartGuest(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.TaskComplete(ctx, guest)
}

func (self *GuestCreateTask) OnSyncStatusComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.TaskComplete(ctx, guest)
}

func (self *GuestCreateTask) TaskComplete(ctx context.Context, guest *models.SGuest) {
	db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE, "", self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_ALLOCATE, "", self.UserCred, true)
	self.SetStageComplete(ctx, guest.GetShortDesc(ctx))

}

func (self *GuestCreateTask) StartEipSubTask(ctx context.Context, guest *models.SGuest) {
	eipId, _ := self.Params.GetString("eip")
	if len(eipId) > 0 {
		eipObj, err := models.ElasticipManager.FetchById(eipId)
		if err != nil {
			log.Errorf("fail to get eip %s %s", eipId, err)
			return
		}

		eip := eipObj.(*models.SElasticip)

		eipBw, _ := self.Params.Int("eip_bw")
		if eipBw > 0 {
			// newly allocated eip, need allocation and associate
			eip.AllocateAndAssociateVM(ctx, self.UserCred, guest)
		} else {
			// existing eip, association only
			eip.StartEipAssociateInstanceTask(ctx, self.UserCred, guest, "")
		}
		return
	}

	self.OnDeployEipComplete(ctx, guest, nil)
}
