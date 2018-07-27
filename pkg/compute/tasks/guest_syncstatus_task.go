package tasks

import (
	"context"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/cloudprovider"
	"github.com/yunionio/onecloud/pkg/compute/models"
)

type GuestSyncstatusTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestSyncstatusTask{})
}

func (self *GuestSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	host := guest.GetHost()
	if host == nil || host.HostStatus == models.HOST_OFFLINE {
		guest.SetStatus(self.UserCred, models.VM_UNKNOWN, "Host not responding")
		self.SetStageComplete(ctx, nil)
		return
	}
	body, err := guest.GetDriver().RequestSyncstatusOnHost(ctx, guest, host, self.UserCred)
	if err != nil {
		log.Errorf("request_syncstatus_on_host: %s", err)
		self.OnGetStatusFail(ctx, guest, err)
		return
	}
	self.OnGetStatusSucc(ctx, guest, body)
}

func (self *GuestSyncstatusTask) OnGetStatusSucc(ctx context.Context, guest *models.SGuest, body jsonutils.JSONObject) {
	statusStr, _ := body.GetString("status")
	switch statusStr {
	case cloudprovider.CloudVMStatusRunning:
		statusStr = models.VM_RUNNING
	case cloudprovider.CloudVMStatusSuspend:
		statusStr = models.VM_SUSPEND
	case cloudprovider.CloudVMStatusStopped:
		statusStr = models.VM_READY
	default:
		statusStr = models.VM_UNKNOWN
	}
	guest.SetStatus(self.UserCred, statusStr, "syncstatus")
	self.SetStageComplete(ctx, nil)
}

func (self *GuestSyncstatusTask) OnGetStatusFail(ctx context.Context, guest *models.SGuest, err error) {
	guest.SetStatus(self.UserCred, models.VM_UNKNOWN, err.Error())
	self.SetStageComplete(ctx, nil)
}
