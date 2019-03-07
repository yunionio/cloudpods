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

type BaremetalServerSyncStatusTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(BaremetalServerSyncStatusTask{})
}

func (self *BaremetalServerSyncStatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	baremetal := guest.GetHost()
	if baremetal == nil {
		guest.SetStatus(self.UserCred, models.VM_INIT, "BaremetalServerSyncStatusTask")
		self.SetStageComplete(ctx, nil)
		return
	}
	url := fmt.Sprintf("/baremetals/%s/servers/%s/status", baremetal.Id, guest.Id)
	headers := self.GetTaskRequestHeader()
	self.SetStage("OnGuestStatusTaskComplete", nil)
	_, err := baremetal.BaremetalSyncRequest(ctx, "POST", url, headers, nil)
	if err != nil {
		log.Errorln(err)
		self.OnGetStatusFail(ctx, guest)
	}
}

func (self *BaremetalServerSyncStatusTask) OnGuestStatusTaskComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	var status string
	var hostStatus string
	if data.Contains("status") {
		statusStr, _ := data.GetString("status")
		switch statusStr {
		case "running":
			status = models.VM_RUNNING
			hostStatus = models.HOST_STATUS_RUNNING
		case "stopped", "ready":
			status = models.VM_READY
			hostStatus = models.HOST_STATUS_READY
		case "admin":
			status = models.VM_ADMIN
			hostStatus = models.HOST_STATUS_RUNNING
		default:
			status = models.VM_INIT
			hostStatus = models.HOST_STATUS_UNKNOWN
		}
	} else {
		status = models.VM_UNKNOWN
		hostStatus = models.HOST_STATUS_UNKNOWN
	}
	guest.SetStatus(self.UserCred, status, "BaremetalServerSyncStatusTask")
	host := guest.GetHost()
	host.SetStatus(self.UserCred, hostStatus, "BaremetalServerSyncStatusTask")

	self.SetStageComplete(ctx, nil)
}

func (self *BaremetalServerSyncStatusTask) OnGuestStatusTaskCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	kwargs := jsonutils.NewDict()
	kwargs.Set("status", jsonutils.NewString(models.VM_UNKNOWN))
	guest.PerformStatus(ctx, self.UserCred, nil, kwargs)
}

func (self *BaremetalServerSyncStatusTask) OnGetStatusFail(ctx context.Context, guest *models.SGuest) {
	kwargs := jsonutils.NewDict()
	kwargs.Set("status", jsonutils.NewString(models.VM_UNKNOWN))
	guest.PerformStatus(ctx, self.UserCred, nil, kwargs)
	self.SetStageComplete(ctx, nil)
}
