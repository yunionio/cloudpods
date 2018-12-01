package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type BaremetalServerStopTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(BaremetalServerStopTask{})
}

func (self *BaremetalServerStopTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_STOPPING, "", self.UserCred)
	guest.SetStatus(self.UserCred, models.VM_START_STOP, "")
	baremetal := guest.GetHost()
	if baremetal != nil {
		self.OnStopGuestFail(ctx, guest, "Baremetal is None")
		return
	}
	params := jsonutils.NewDict()
	timeout, err := self.Params.Int("timeout")
	if err != nil {
		timeout = 30
	}
	if jsonutils.QueryBoolean(self.Params, "is_force", false) || jsonutils.QueryBoolean(self.Params, "reset", false) {
		timeout = 0
	}
	params.Set("timeout", jsonutils.NewInt(timeout))
	url := fmt.Sprintf("/baremetals/%s/servers/%s/stop", baremetal.Id, guest.Id)
	headers := self.GetTaskRequestHeader()
	self.SetStage("OnGuestStopTaskComplete", nil)
	_, err = baremetal.BaremetalSyncRequest(ctx, "POST", url, headers, params)
	if err != nil {
		log.Errorln(err)
		self.OnStopGuestFail(ctx, guest, err.Error())
	}
}

func (self *BaremetalServerStopTask) OnGuestStopTaskComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if guest.Status == models.VM_STOPPING {
		guest.SetStatus(self.UserCred, models.VM_READY, "")
		db.OpsLog.LogEvent(guest, db.ACT_STOP, "", self.UserCred)
	}
	baremetal := guest.GetHost()
	baremetal.SetStatus(self.UserCred, models.BAREMETAL_READY, "")
	self.SetStageComplete(ctx, nil)
	if guest.Status == models.VM_READY {
		if !jsonutils.QueryBoolean(self.Params, "reset", false) && guest.DisableDelete.IsFalse() && guest.ShutdownBehavior == models.SHUTDOWN_TERMINATE {
			guest.StartAutoDeleteGuestTask(ctx, self.UserCred, "")
		}
	}
}

func (self *BaremetalServerStopTask) OnGuestStopTaskCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetStatus(self.UserCred, db.ACT_STOP_FAIL, data.String())
	db.OpsLog.LogEvent(guest, db.ACT_STOP_FAIL, data.String(), self.UserCred)
	baremetal := guest.GetHost()
	baremetal.SetStatus(self.UserCred, models.BAREMETAL_READY, "")
	self.SetStageFailed(ctx, data.String())
}

func (self *BaremetalServerStopTask) OnStopGuestFail(ctx context.Context, guest *models.SGuest, reason string) {
	self.OnGuestStopTaskCompleteFailed(ctx, guest, jsonutils.NewString(reason))
}
