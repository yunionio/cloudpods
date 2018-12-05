package tasks

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type GuestSuspendTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestSuspendTask{})
}

func (self *GuestSuspendTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_STOPPING, "", self.UserCred)
	guest.SetStatus(self.UserCred, models.VM_SUSPENDING, "GuestSusPendTask")
	self.SetStage("on_suspend_complete", nil)
	err := guest.GetDriver().RqeuestSuspendOnHost(ctx, guest, self)
	if err != nil {
		self.OnSuspendGuestFail(guest, err.Error())
	}
}

func (self *GuestSuspendTask) OnSuspendComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, models.VM_SUSPEND, "")
	db.OpsLog.LogEvent(guest, db.ACT_STOP, "", self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *GuestSuspendTask) OnSuspendCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, models.VM_RUNNING, "")
	db.OpsLog.LogEvent(guest, db.ACT_STOP_FAIL, err.String(), self.UserCred)
	self.SetStageFailed(ctx, err.String())
}

func (self *GuestSuspendTask) OnSuspendGuestFail(guest *models.SGuest, reason string) {
	guest.SetStatus(self.UserCred, models.VM_SUSPEND_FAILED, reason)
}
