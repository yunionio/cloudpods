package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestStartTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestStartTask{})
}

func (self *GuestStartTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.checkTemplate(ctx, guest)
}

func (self *GuestStartTask) checkTemplate(ctx context.Context, guest *models.SGuest) {
	diskCat := guest.CategorizeDisks()
	if diskCat.Root != nil && len(diskCat.Root.GetTemplateId()) > 0 {
		if len(guest.BackupHostId) > 0 {
			self.SetStage("OnMasterHostTemplateReady", nil)
		} else {
			self.SetStage("OnStartTemplateReady", nil)
		}
		guest.GetDriver().CheckDiskTemplateOnStorage(ctx, self.UserCred, diskCat.Root.GetTemplateId(), diskCat.Root.StorageId, self)
	} else {
		self.startStart(ctx, guest)
	}
}

func (self *GuestStartTask) OnMasterHostTemplateReady(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStage("OnStartTemplateReady", nil)
	diskCat := guest.CategorizeDisks()
	err := guest.GetDriver().CheckDiskTemplateOnStorage(ctx, self.UserCred, diskCat.Root.GetTemplateId(),
		diskCat.Root.BackupStorageId, self)
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
	}
}

func (self *GuestStartTask) OnStartTemplateReady(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.startStart(ctx, guest)
}

func (self *GuestStartTask) startStart(ctx context.Context, guest *models.SGuest) {
	db.OpsLog.LogEvent(guest, db.ACT_STARTING, nil, self.UserCred)
	if len(guest.BackupHostId) > 0 {
		self.RequestStartBacking(ctx, guest)
	} else {
		self.RequestStart(ctx, guest)
	}
}

func (self *GuestStartTask) RequestStart(ctx context.Context, guest *models.SGuest) {
	self.SetStage("OnStartComplete", nil)
	host := guest.GetHost()
	guest.SetStatus(self.UserCred, models.VM_STARTING, "")
	result, err := guest.GetDriver().RequestStartOnHost(ctx, guest, host, self.UserCred, self)
	if err != nil {
		self.onStartGuestFailed(ctx, guest, err)
	} else {
		if result != nil && jsonutils.QueryBoolean(result, "is_running", false) {
			// guest.SetStatus(self.UserCred, models.VM_RUNNING, "start")
			// self.taskComplete(ctx, guest)
			self.OnStartComplete(ctx, guest, nil)
		}
	}
}

func (self *GuestStartTask) RequestStartBacking(ctx context.Context, guest *models.SGuest) {
	self.SetStage("OnStartBackupGuestComplete", nil)
	host := models.HostManager.FetchHostById(guest.BackupHostId)
	guest.SetStatus(self.UserCred, models.VM_BACKUP_STARTING, "")
	result, err := guest.GetDriver().RequestStartOnHost(ctx, guest, host, self.UserCred, self)
	if err != nil {
		self.onStartGuestFailed(ctx, guest, err)
	} else {
		if result != nil && jsonutils.QueryBoolean(result, "is_running", false) {
			self.OnStartBackupGuestComplete(ctx, guest, nil)
		}
	}
}

func (self *GuestStartTask) OnStartBackupGuestComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if data != nil {
		nbdServerPort, err := data.Int("nbd_server_port")
		if err == nil {
			backupHost := models.HostManager.FetchHostById(guest.BackupHostId)
			nbdServerUri := fmt.Sprintf("nbd:%s:%d", backupHost.AccessIp, nbdServerPort)
			guest.SetMetadata(ctx, "backup_nbd_server_uri", nbdServerUri, self.UserCred)
		} else {
			self.onStartGuestFailed(ctx, guest, fmt.Errorf("Start backup guest result missing nbd_server_port"))
			return
		}
	}
	self.RequestStart(ctx, guest)
}

func (self *GuestStartTask) OnStartComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_START, guest.GetShortDesc(), self.UserCred)
	self.SetStage("OnGuestSyncstatusAfterStart", nil)
	guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
	logclient.AddActionLog(guest, logclient.ACT_VM_START, "", self.UserCred, true)
	// self.taskComplete(ctx, guest)
}

func (self *GuestStartTask) OnGuestSyncstatusAfterStart(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.taskComplete(ctx, guest)
}

func (self *GuestStartTask) OnStartCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_START_FAIL, err, self.UserCred)
}

func (self *GuestStartTask) onStartGuestFailed(ctx context.Context, guest *models.SGuest, err error) {
	guest.SetStatus(self.UserCred, models.VM_START_FAILED, err.Error())
	self.SetStageFailed(ctx, err.Error())
	self.OnStartCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
	logclient.AddActionLog(guest, logclient.ACT_VM_START, err, self.UserCred, false)
}

func (self *GuestStartTask) taskComplete(ctx context.Context, guest *models.SGuest) {
	models.HostManager.ClearSchedDescCache(guest.HostId)
	self.SetStageComplete(ctx, nil)
}
