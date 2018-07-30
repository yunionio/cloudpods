package tasks

import (
	"context"
	"fmt"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/compute/models"
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
)

type GuestDeployTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestDeployTask{})
}

func (self *GuestDeployTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if !guest.IsNetworkAllocated() {
		self.SetStageFailed(ctx, fmt.Sprintf("Guest %s network not ready!!", guest.Name))
	} else {
		self.OnGuestNetworkReady(ctx, guest)
	}
}

func (self *GuestDeployTask) OnGuestNetworkReady(ctx context.Context, guest *models.SGuest) {
	if jsonutils.QueryBoolean(self.Params, "restart", false) {
		self.SetStage("on_deploy_wait_server_stop", nil)
		guest.StartGuestStopTask(ctx, self.UserCred, false, self.GetTaskId())
	} else {
		self.OnDeployWaitServerStop(ctx, guest, nil)
	}
}

func (self *GuestDeployTask) OnDeployWaitServerStop(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	targetHostId, _ := self.Params.GetString("target_host_id")
	if len(targetHostId) == 0 {
		targetHostId = guest.HostId
	}
	host := models.HostManager.FetchHostById(targetHostId)
	self.StartDeployGuestOnHost(ctx, guest, host)
}

func (self *GuestDeployTask) StartDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost) {
	self.SetStage("on_deploy_guest_complete", nil)
	err := guest.GetDriver().RequestDeployGuestOnHost(ctx, guest, host, self)
	if err != nil {
		log.Errorf("request_deploy_guest_on_host %s", err)
		self.OnDeployGuestFail(ctx, guest, err)
	} else {
		self.OnDeployGuestSucc(guest)
	}
}

func (self *GuestDeployTask) OnDeployGuestSucc(guest *models.SGuest) {
	guest.SetStatus(self.UserCred, models.VM_DEPLOYING, "")
}

func (self *GuestDeployTask) OnDeployGuestFail(ctx context.Context, guest *models.SGuest, err error) {
	guest.SetStatus(self.UserCred, models.VM_DEPLOY_FAILED, err.Error())
	self.SetStageFailed(ctx, err.Error())
}

func (self *GuestDeployTask) OnDeployGuestComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	log.Infof("on_guest_deploy_task_data_received %s", data)
	guest := obj.(*models.SGuest)
	guest.GetDriver().OnGuestDeployTaskDataReceived(ctx, guest, self, data)
	guest.GetDriver().OnGuestDeployTaskComplete(ctx, guest, self)
}

func (self *GuestDeployTask) OnDeployGuestCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, models.VM_DEPLOY_FAILED, data.String())
}

func (self *GuestDeployTask) OnDeployStartGuestComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestDeployTask) OnDeployGuestSyncstatusComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}
