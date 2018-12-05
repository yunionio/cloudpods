package tasks

import (
	"context"

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
		self.SetStage("on_start_template_ready", nil)
		guest.GetDriver().CheckDiskTemplateOnStorage(ctx, self.UserCred, diskCat.Root.GetTemplateId(), diskCat.Root.StorageId, self)
	} else {
		self.startStart(ctx, guest)
	}
}

func (self *GuestStartTask) OnStartTemplateReady(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.startStart(ctx, guest)
}

func (self *GuestStartTask) startStart(ctx context.Context, guest *models.SGuest) {
	db.OpsLog.LogEvent(guest, db.ACT_STARTING, nil, self.UserCred)
	self.SetStage("on_start_complete", nil)
	host := guest.GetHost()
	guest.SetStatus(self.UserCred, models.VM_STARTING, "")
	result, err := guest.GetDriver().RequestStartOnHost(ctx, guest, host, self.UserCred, self)
	if err != nil {
		self.OnStartCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
	} else {
		if result != nil && jsonutils.QueryBoolean(result, "is_running", false) {
			self.OnStartComplete(ctx, guest, nil)
			// guest.SetStatus(self.UserCred, models.VM_RUNNING, "start")
			// self.taskComplete(ctx, guest)
		}
	}
}

func (self *GuestStartTask) OnStartComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_START, guest.GetShortDesc(), self.UserCred)
	self.SetStage("on_guest_syncstatus_after_start", nil)
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
	guest.SetStatus(self.UserCred, models.VM_START_FAILED, err.String())
	db.OpsLog.LogEvent(guest, db.ACT_START_FAIL, err, self.UserCred)
	logclient.AddActionLog(guest, logclient.ACT_VM_START, err, self.UserCred, false)
	self.SetStageFailed(ctx, err.String())
}

func (self *GuestStartTask) taskComplete(ctx context.Context, guest *models.SGuest) {
	models.HostManager.ClearSchedDescCache(guest.HostId)
	self.SetStageComplete(ctx, nil)
}
