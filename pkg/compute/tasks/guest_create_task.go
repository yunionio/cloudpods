package tasks

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
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
	guest.SetStatus(self.UserCred, models.VM_CREATE_NETWORK, "")
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
	guest.SetStatus(self.UserCred, models.VM_CREATE_DISK, "")
	self.SetStage("on_disk_prepared", nil)
	guest.GetDriver().RequestGuestCreateAllDisks(ctx, guest, self)
}

func (self *GuestCreateTask) OnDiskPreparedFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, models.VM_DISK_FAILED, "allocation failed")
	db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE_FAIL, data, self.UserCred)
	logclient.AddActionLog(guest, logclient.ACT_ALLOCATE, data, self.UserCred, false)
	notifyclient.NotifySystemError(guest.Id, guest.Name, models.VM_DISK_FAILED, data.String())
}

func (self *GuestCreateTask) OnDiskPrepared(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	cdrom, _ := self.Params.GetString("cdrom")
	if len(cdrom) > 0 {
		self.SetStage("on_cdrom_prepared", nil)
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
	guest.SetStatus(self.UserCred, models.VM_DEPLOYING, "")
	self.StartDeployGuest(ctx, guest)
}

func (self *GuestCreateTask) OnCdromPreparedFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, models.VM_DISK_FAILED, "")
	db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE_FAIL, data, self.UserCred)
	logclient.AddActionLog(guest, logclient.ACT_ALLOCATE, data, self.UserCred, false)
	notifyclient.NotifySystemError(guest.Id, guest.Name, models.VM_DISK_FAILED, fmt.Sprintf("cdrom_failed %s", data))
}

func (self *GuestCreateTask) StartDeployGuest(ctx context.Context, guest *models.SGuest) {
	self.SetStage("on_deploy_guest_desc_complete", nil)
	guest.StartGuestDeployTask(ctx, self.UserCred, self.Params, "create", self.GetId())
}

func (self *GuestCreateTask) OnDeployGuestDescComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE, nil, self.UserCred)
	if !guest.IsSystem {
		self.notifyServerCreated(ctx, guest)
	}
	guest.GetDriver().OnGuestCreateTaskComplete(ctx, guest, self)
}

func (self *GuestCreateTask) notifyServerCreated(ctx context.Context, guest *models.SGuest) {
	guest.NotifyServerEvent(notifyclient.SERVER_CREATED, notifyclient.PRIORITY_IMPORTANT, true)
	guest.NotifyAdminServerEvent(ctx, notifyclient.SERVER_CREATED_ADMIN, notifyclient.PRIORITY_IMPORTANT)
}

func (self *GuestCreateTask) OnDeployGuestDescCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, models.VM_DEPLOY_FAILED, "deploy_failed")
	db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE_FAIL, data, self.UserCred)
	logclient.AddActionLog(guest, logclient.ACT_ALLOCATE, data, self.UserCred, false)
	notifyclient.NotifySystemError(guest.Id, guest.Name, models.VM_DEPLOY_FAILED, data.String())
	self.SetStageFailed(ctx, data.String())
}

func (self *GuestCreateTask) OnAutoStartGuest(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.SetStageComplete(ctx, guest.GetShortDesc())
}

func (self *GuestCreateTask) OnSyncStatusComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.SetStageComplete(ctx, guest.GetShortDesc())
}
