package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestDeployTask struct {
	SGuestBaseTask
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
		self.SetStage("OnDeployWaitServerStop", nil)
		guest.StartGuestStopTask(ctx, self.UserCred, false, self.GetTaskId())
	} else {
		self.OnDeployWaitServerStop(ctx, guest, nil)
	}
}

func (self *GuestDeployTask) OnDeployWaitServerStop(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStage("OnDeployGuestComplete", nil)
	targetHostId, _ := self.Params.GetString("target_host_id")
	if len(targetHostId) == 0 {
		if len(guest.BackupHostId) > 0 {
			self.SetStage("OnSlaveHostDeployComplete", nil)
			self.DeployBackup(ctx, guest, nil)
			return
		} else {
			targetHostId = guest.HostId
		}
	}
	host := models.HostManager.FetchHostById(targetHostId)
	self.DeployOnHost(ctx, guest, host)
}

func (self *GuestDeployTask) DeployBackup(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	host := models.HostManager.FetchHostById(guest.BackupHostId)
	err := guest.GetDriver().RequestDeployGuestOnHost(ctx, guest, host, self)
	if err != nil {
		log.Errorf("request_deploy_guest_on_host %s", err)
		self.OnDeployGuestFail(ctx, guest, err)
	} else {
		guest.SetStatus(self.UserCred, models.VM_DEPLOYING_BACKUP, "")
	}
}

func (self *GuestDeployTask) DeployOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost) {
	err := guest.GetDriver().RequestDeployGuestOnHost(ctx, guest, host, self)
	if err != nil {
		log.Errorf("request_deploy_guest_on_host %s", err)
		self.OnDeployGuestFail(ctx, guest, err)
	} else {
		guest.SetStatus(self.UserCred, models.VM_DEPLOYING, "")
	}
}

func (self *GuestDeployTask) OnSlaveHostDeployComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	host := guest.GetHost()
	self.SetStage("OnDeployGuestComplete", nil)
	self.DeployOnHost(ctx, guest, host)
}

func (self *GuestDeployTask) OnSlaveHostDeployCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetStatus(self.UserCred, models.VM_DEPLOYING_BACKUP_FAILED, "")
	self.SetStage("OnUndeployBackupGuest", nil)
	guest.StartUndeployGuestTask(ctx, self.UserCred, self.GetId(), guest.BackupHostId)
}

func (self *GuestDeployTask) OnUndeployBackupGuest(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, "deploy backup failed")
}

func (self *GuestDeployTask) OnDeployGuestFail(ctx context.Context, guest *models.SGuest, err error) {
	guest.SetStatus(self.UserCred, models.VM_DEPLOY_FAILED, err.Error())
	self.SetStageFailed(ctx, err.Error())
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_DEPLOY, err, self.UserCred, false)
}

func (self *GuestDeployTask) OnDeployGuestComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	log.Infof("on_guest_deploy_task_data_received %s", data)
	guest := obj.(*models.SGuest)
	guest.GetDriver().OnGuestDeployTaskDataReceived(ctx, guest, self, data)
	guest.GetDriver().OnGuestDeployTaskComplete(ctx, guest, self)
	action, _ := self.Params.GetString("deploy_action")
	keypair, _ := self.Params.GetString("keypair")
	reset_password := jsonutils.QueryBoolean(self.Params, "reset_password", false)
	unbind_kp := jsonutils.QueryBoolean(self.Params, "__delete_keypair__", false)
	_log := false
	if action == "deploy" {
		if len(keypair) >= 32 {
			if unbind_kp {
				logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_UNBIND_KEYPAIR, nil, self.UserCred, true)
				_log = true
			} else {
				logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_BIND_KEYPAIR, nil, self.UserCred, true)
				_log = true
			}

		} else if reset_password {
			logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_RESET_PSWD, "", self.UserCred, true)
			_log = true
		}
	}
	if !_log {
		// 如果 deploy 有其他事件，统一记在这里。
		logclient.AddActionLogWithStartable(self, guest, "misc部署", "", self.UserCred, true)
	}
}

func (self *GuestDeployTask) OnDeployGuestCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	action, _ := self.Params.GetString("deploy_action")
	keypair, _ := self.Params.GetString("keypair")
	if action == "deploy" && len(keypair) >= 32 {
		_, err := db.Update(guest, func() error {
			guest.KeypairId = ""
			return nil
		})
		if err != nil {
			log.Errorf("unset guest %s keypair failed %v", guest.Name, err)
		}
	}
	guest.SetStatus(self.UserCred, models.VM_DEPLOY_FAILED, data.String())
	self.SetStageFailed(ctx, data.String())
}

func (self *GuestDeployTask) OnDeployStartGuestComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestDeployTask) OnDeployStartGuestCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data.String())
}

func (self *GuestDeployTask) OnDeployGuestSyncstatusComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestDeployTask) OnDeployGuestSyncstatusCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data.String())
}

type GuestDeployBackupTask struct {
	GuestDeployTask
}

func (self *GuestDeployBackupTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if len(guest.BackupHostId) == 0 {
		self.SetStageFailed(ctx, "Guest dosen't have backup host")
	}
	self.SetStage("OnDeployGuestComplete", nil)
	self.DeployBackup(ctx, guest, nil)
}

func (self *GuestDeployBackupTask) OnDeployGuestComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func init() {
	taskman.RegisterTask(GuestDeployTask{})
	taskman.RegisterTask(GuestDeployBackupTask{})
}
