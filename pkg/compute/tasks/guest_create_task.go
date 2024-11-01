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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
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
	guest.SetStatus(ctx, self.UserCred, api.VM_CREATE_NETWORK, "")
	self.SetStage("OnWaitGuestNetworksReady", nil)
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
	guest.SetStatus(ctx, self.UserCred, api.VM_CREATE_DISK, "")
	self.SetStage("OnDiskPrepared", nil)
	drv, err := guest.GetDriver()
	if err != nil {
		self.OnDiskPreparedFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	err = drv.RequestGuestCreateAllDisks(ctx, guest, self)
	if err != nil {
		msg := fmt.Sprintf("unable to RequestGuestCreateAllDisks: %v", err)
		self.OnDiskPreparedFailed(ctx, guest, jsonutils.NewString(msg))
	}
}

func (self *GuestCreateTask) OnDiskPreparedFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetStatus(ctx, self.UserCred, api.VM_DISK_FAILED, "allocation failed")
	db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE_FAIL, data, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_ALLOCATE, data, self.UserCred, false)
	notifyclient.EventNotify(ctx, self.GetUserCred(), notifyclient.SEventNotifyParam{
		Obj:    guest,
		Action: notifyclient.ActionCreate,
		IsFail: true,
	})

	self.SetStageFailed(ctx, data)
}

func (self *GuestCreateTask) OnDiskPrepared(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	secgroups, err := guest.GetSecgroups()
	if err != nil {
		self.OnSecurityGroupPreparedFailed(ctx, guest, jsonutils.NewString(errors.Wrapf(err, "GetSecgroups").Error()))
		return
	}
	if len(secgroups) == 0 {
		self.OnSecurityGroupPrepared(ctx, guest, nil)
		return
	}

	vpc, err := guest.GetVpc()
	if err != nil {
		self.OnSecurityGroupPreparedFailed(ctx, guest, jsonutils.NewString(errors.Wrapf(err, "GetVpc").Error()))
		return
	}
	region, err := vpc.GetRegion()
	if err != nil {
		self.OnSecurityGroupPreparedFailed(ctx, guest, jsonutils.NewString(errors.Wrapf(err, "GetRegion").Error()))
		return
	}

	self.SetStage("OnSecurityGroupPrepared", nil)
	err = region.GetDriver().RequestPrepareSecurityGroups(ctx, self.UserCred, guest.GetOwnerId(), secgroups, vpc, func(ids []string) error {
		return guest.SaveSecgroups(ctx, self.UserCred, ids)
	}, self)
	if err != nil {
		self.OnSecurityGroupPreparedFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *GuestCreateTask) OnSecurityGroupPreparedFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetStatus(ctx, self.UserCred, api.VM_SECURITY_GROUP_FAILED, "prepare security group failed")
	db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE_FAIL, data, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_ALLOCATE, data, self.UserCred, false)
	notifyclient.EventNotify(ctx, self.GetUserCred(), notifyclient.SEventNotifyParam{
		Obj:    guest,
		Action: notifyclient.ActionCreate,
		IsFail: true,
	})

	self.SetStageFailed(ctx, data)
}

func (self *GuestCreateTask) OnSecurityGroupPrepared(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	cdrom, _ := self.Params.GetString("cdrom")
	var bootIndex *int8
	if self.Params.Contains("cdrom_boot_index") {
		bd, _ := self.Params.Int("cdrom_boot_index")
		bd8 := int8(bd)
		bootIndex = &bd8
	}

	if len(cdrom) > 0 {
		image, err := models.CachedimageManager.GetImageInfo(ctx, self.UserCred, cdrom, false)
		if err != nil {
			log.Errorf("failed get image %s info: %s", cdrom, err)
		} else {
			imagePros := map[string]interface{}{}
			for _, k := range []string{imageapi.IMAGE_OS_ARCH, imageapi.IMAGE_OS_DISTRO, imageapi.IMAGE_OS_VERSION, imageapi.IMAGE_OS_TYPE} {
				if v, ok := image.Properties[k]; ok {
					imagePros[k] = v
				}
			}
			guest.SetAllMetadata(ctx, imagePros, self.UserCred)
		}
		self.SetStage("OnCdromPrepared", nil)
		drv, err := guest.GetDriver()
		if err != nil {
			self.OnCdromPreparedFailed(ctx, guest, jsonutils.NewString(err.Error()))
			return
		}
		drv.RequestGuestCreateInsertIso(ctx, cdrom, bootIndex, self, guest)
	} else {
		self.OnCdromPrepared(ctx, guest, data)
	}
}

func (self *GuestCreateTask) OnCdromPrepared(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	log.Infof("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")
	log.Infof("DEPLOY GUEST %s", guest.Name)
	log.Infof("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")
	guest.SetStatus(ctx, self.UserCred, api.VM_DEPLOYING, "")
	self.StartDeployGuest(ctx, guest)
}

func (self *GuestCreateTask) OnCdromPreparedFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(ctx, self.UserCred, api.VM_DISK_FAILED, "")
	db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE_FAIL, data, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_ALLOCATE, data, self.UserCred, false)
	notifyclient.EventNotify(ctx, self.GetUserCred(), notifyclient.SEventNotifyParam{
		Obj:    guest,
		Action: notifyclient.ActionCreate,
		IsFail: true,
	})
	self.SetStageFailed(ctx, data)
}

func (self *GuestCreateTask) StartDeployGuest(ctx context.Context, guest *models.SGuest) {
	self.SetStage("OnDeployGuestDescComplete", nil)
	guest.StartGuestDeployTask(ctx, self.UserCred, self.Params, "create", self.GetId())
}

func (self *GuestCreateTask) OnDeployGuestDescComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	// bind eip
	{
		eipId, _ := self.Params.GetString("eip")
		if len(eipId) > 0 {
			var err error
			self.SetStage("OnDeployEipComplete", nil)
			eipObj, err := models.ElasticipManager.FetchById(eipId)
			if err != nil {
				msg := fmt.Sprintf("fail to get eip %s %s", eipId, err)
				self.OnDeployEipCompleteFailed(ctx, obj, jsonutils.NewString(msg))
				return
			}

			eip := eipObj.(*models.SElasticip)

			input := api.ElasticipAssociateInput{
				InstanceId:         guest.Id,
				InstanceExternalId: guest.ExternalId,
				InstanceType:       api.EIP_ASSOCIATE_TYPE_SERVER,
			}

			eipBw, _ := self.Params.Int("eip_bw")
			if eipBw > 0 {
				// newly allocated eip, need allocation and associate
				err = eip.AllocateAndAssociateInstance(ctx, self.UserCred, guest, input, self.GetId())
				err = errors.Wrapf(err, "eip.AllocateAndAssociateInstance")
			} else {
				// existing eip, association only
				err = eip.StartEipAssociateInstanceTask(ctx, self.UserCred, input, self.GetId())
				err = errors.Wrapf(err, "eip.StartEipAssociateInstanceTask")
			}
			if err != nil {
				self.OnDeployEipCompleteFailed(ctx, obj, jsonutils.NewString(err.Error()))
				return
			}

			return
		}
	}

	self.OnDeployEipComplete(ctx, guest, nil)
}

func (self *GuestCreateTask) notifyServerCreated(ctx context.Context, guest *models.SGuest) {
	guest.EventNotify(ctx, self.UserCred, notifyclient.ActionCreate)
}

func (self *GuestCreateTask) OnDeployGuestDescCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(ctx, self.UserCred, api.VM_DEPLOY_FAILED, "deploy_failed")
	db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE_FAIL, data, self.UserCred)
	notifyclient.EventNotify(ctx, self.GetUserCred(), notifyclient.SEventNotifyParam{
		Obj:    guest,
		Action: notifyclient.ActionCreate,
		IsFail: true,
	})
	self.SetStageFailed(ctx, data)
}

func (self *GuestCreateTask) OnDeployEipComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE, nil, self.UserCred)
	if !guest.IsSystem {
		self.notifyServerCreated(ctx, guest)
	}

	// Guest Create Complete
	duration, _ := self.GetParams().GetString("duration")
	if len(duration) > 0 {
		bc, err := billing.ParseBillingCycle(duration)
		if err == nil && guest.ExpiredAt.IsZero() {
			guest.SaveRenewInfo(ctx, self.GetUserCred(), &bc, nil, "")
		}
		if jsonutils.QueryBoolean(self.GetParams(), "auto_prepaid_recycle", false) {
			err := guest.CanPerformPrepaidRecycle()
			if err == nil {
				self.SetStageComplete(ctx, nil)
				guest.DoPerformPrepaidRecycle(ctx, self.GetUserCred(), true)
			}
		}
	}

	if jsonutils.QueryBoolean(self.GetParams(), "auto_start", false) {
		self.SetStage("OnAutoStartGuest", nil)
		params := jsonutils.NewDict()
		params.Set("start_from_create", jsonutils.JSONTrue)
		guest.StartGueststartTask(ctx, self.GetUserCred(), params, self.GetTaskId())
	} else {
		self.SetStage("OnSyncStatusComplete", nil)
		guest.StartSyncstatus(ctx, self.GetUserCred(), self.GetTaskId())
	}
}

func (self *GuestCreateTask) OnDeployEipCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(ctx, self.UserCred, api.INSTANCE_ASSOCIATE_EIP_FAILED, "deploy_failed")
	db.OpsLog.LogEvent(guest, db.ACT_EIP_ATTACH, data, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_EIP_ASSOCIATE, data, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, guest.Id, guest.Name, api.INSTANCE_ASSOCIATE_EIP_FAILED, data.String())
	self.SetStageFailed(ctx, data)
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
