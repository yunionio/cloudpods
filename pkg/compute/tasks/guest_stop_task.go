package tasks

import (
	"context"
	"fmt"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/compute/models"
)

type GuestStopTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestStopTask{})
}

func (self *GuestStopTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_STOPPING, nil, self.UserCred)
	self.stopGuest(ctx, guest)
}

func (self *GuestStopTask) isSubtask() bool {
	return jsonutils.QueryBoolean(self.Params, "subtask", false)
}

func (self *GuestStopTask) stopGuest(ctx context.Context, guest *models.SGuest) {
	host := guest.GetHost()
	if host == nil {
		self.OnStopGuestFail(ctx, guest, fmt.Errorf("no associated host"))
		return
	}
	if !self.isSubtask() {
		guest.SetStatus(self.UserCred, models.VM_STOPPING, "")
	}
	self.SetStage("on_guest_stop_task_complete", nil)
	err := guest.GetDriver().RequestStopOnHost(ctx, guest, host, self)
	if err != nil {
		log.Errorf("RequestStopOnHost fail %s", err)
		self.OnStopGuestFail(ctx, guest, err)
	}
}

func (self *GuestStopTask) OnGuestStopTaskComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if !self.isSubtask() {
		guest.SetStatus(self.UserCred, models.VM_READY, "")
	}
	db.OpsLog.LogEvent(guest, db.ACT_STOP, nil, self.UserCred)
	models.HostManager.ClearSchedDescCache(guest.HostId)
	self.SetStageComplete(ctx, nil)
	if guest.Status == models.VM_READY && guest.DisableDelete.IsFalse() && guest.ShutdownBehavior == models.SHUTDOWN_TERMINATE {
		guest.StartAutoDeleteGuestTask(ctx, self.UserCred, "")
	}
}

func (self *GuestStopTask) OnStopGuestFail(ctx context.Context, guest *models.SGuest, err error) {
	guest.SetStatus(self.UserCred, models.VM_STOP_FAILED, err.Error())
	db.OpsLog.LogEvent(guest, db.ACT_STOP_FAIL, err.Error(), self.UserCred)
	self.SetStageFailed(ctx, err.Error())
}
