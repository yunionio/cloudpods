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

type BaremetalServerStartTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(BaremetalServerStartTask{})
}

func (self *BaremetalServerStartTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, models.VM_START_START, "")
	db.OpsLog.LogEvent(guest, db.ACT_STARTING, "", self.UserCred)
	baremetal := guest.GetHost()
	if baremetal == nil {
		self.OnStartCompleteFailed(ctx, guest, jsonutils.NewString("Baremetal is None"))
		return
	}
	desc := guest.GetJsonDescAtBaremetal(ctx, baremetal)
	config := jsonutils.NewDict()
	config.Set("desc", desc)
	url := fmt.Sprintf("/baremetals/%s/servers/%s/start", baremetal.Id, guest.Id)
	headers := self.GetTaskRequestHeader()
	self.SetStage("OnStartComplete", nil)
	_, err := baremetal.BaremetalSyncRequest(ctx, "POST", url, headers, config)
	if err != nil {
		log.Errorln(err)
		self.OnStartCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (self *BaremetalServerStartTask) OnStartComplete(ctx context.Context, guest *models.SGuest, body jsonutils.JSONObject) {
	guest.SetStatus(self.UserCred, models.VM_RUNNING, "")
	baremetal := guest.GetHost()
	baremetal.SetStatus(self.UserCred, models.BAREMETAL_RUNNING, "")
	db.OpsLog.LogEvent(guest, db.ACT_START, guest.GetShortDesc(ctx), self.UserCred)
}

func (self *BaremetalServerStartTask) OnStartCompleteFailed(ctx context.Context, guest *models.SGuest, body jsonutils.JSONObject) {
	reason := body.String()
	guest.SetStatus(self.UserCred, models.VM_START_FAILED, reason)
	db.OpsLog.LogEvent(guest, db.ACT_START_FAIL, reason, self.UserCred)
	baremetal := guest.GetHost()
	baremetal.SetStatus(self.UserCred, models.BAREMETAL_START_FAIL, reason)
	self.SetStageFailed(ctx, reason)
}
