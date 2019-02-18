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

type BaremetalMaintenanceTask struct {
	SBaremetalBaseTask
}

func init() {
	taskman.RegisterTask(BaremetalMaintenanceTask{})
}

func (self *BaremetalMaintenanceTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	baremetal := obj.(*models.SHost)
	url := fmt.Sprintf("/baremetals/%s/maintenance", baremetal.Id)
	headers := self.GetTaskRequestHeader()
	self.SetStage("OnEnterMaintenantModeSucc", nil)
	_, err := baremetal.BaremetalSyncRequest(ctx, "POST", url, headers, self.Params)
	if err != nil {
		self.OnEnterMaintenantModeSuccFailed(ctx, baremetal, jsonutils.NewString(err.Error()))
	}
	baremetal.SetStatus(self.UserCred, models.BAREMETAL_MAINTAINING, "")
}

func (self *BaremetalMaintenanceTask) OnEnterMaintenantModeSucc(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	action := self.Action()
	if len(action) > 0 {
		logclient.AddActionLogWithStartable(self, baremetal, action, "", self.UserCred, true)
	}
	baremetal.GetModelManager().TableSpec().Update(baremetal, func() error {
		baremetal.IsMaintenance = true
		return nil
	})
	username, _ := body.Get("username")
	password, _ := body.Get("password")
	ip, _ := body.Get("ip")
	metadatas := map[string]interface{}{
		"__maint_username": username,
		"__maint_password": password,
		"__maint_ip":       ip,
	}
	guestRunning, err := body.Get("guest_running")
	if err != nil {
		metadatas["__maint_guest_running"] = guestRunning
	}
	baremetal.SetAllMetadata(ctx, metadatas, self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *BaremetalMaintenanceTask) OnEnterMaintenantModeSuccFailed(ctx context.Context, baremetal *models.SHost, body jsonutils.JSONObject) {
	self.SetStageFailed(ctx, body.String())
	baremetal.StartSyncstatus(ctx, self.UserCred, "")
	guest := baremetal.GetBaremetalServer()
	if guest != nil {
		guest.StartSyncstatus(ctx, self.UserCred, "")
	}
	action := self.Action()
	if len(action) > 0 {
		logclient.AddActionLogWithStartable(self, baremetal, action, body.String(), self.UserCred, false)
	}
}
