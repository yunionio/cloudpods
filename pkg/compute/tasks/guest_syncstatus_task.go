package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
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
	if host == nil || host.HostStatus == api.HOST_OFFLINE {
		log.Errorf("host is not reachable")
		guest.SetStatus(self.UserCred, api.VM_UNKNOWN, "Host not responding")
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
		statusStr = api.VM_RUNNING
	case cloudprovider.CloudVMStatusSuspend:
		statusStr = api.VM_SUSPEND
	case cloudprovider.CloudVMStatusStopped:
		statusStr = api.VM_READY
	case api.VM_BLOCK_STREAM: /// XXX ???
		break
	default:
		statusStr = api.VM_UNKNOWN
	}
	statusData := jsonutils.NewDict()
	statusData.Add(jsonutils.NewString(statusStr), "status")
	guest.PerformStatus(ctx, self.UserCred, nil, statusData)
	self.SetStageComplete(ctx, nil)
	// logclient.AddActionLog(guest, logclient.ACT_VM_SYNC_STATUS, "", self.UserCred, true)
}

func (self *GuestSyncstatusTask) OnGetStatusFail(ctx context.Context, guest *models.SGuest, err error) {
	guest.SetStatus(self.UserCred, api.VM_UNKNOWN, err.Error())
	self.SetStageComplete(ctx, nil)
	// logclient.AddActionLog(guest, logclient.ACT_VM_SYNC_STATUS, err, self.UserCred, false)
}
