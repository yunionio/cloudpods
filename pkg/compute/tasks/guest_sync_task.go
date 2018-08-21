package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type GuestSyncConfTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestSyncConfTask{})
}

func (self *GuestSyncConfTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_SYNC_CONF, nil, self.UserCred)
	if host := guest.GetHost(); host == nil {
		self.SetStageFailed(ctx, "No host for sync")
		return
	} else {
		self.SetStage("on_sync_complete", nil)
		if err := guest.GetDriver().RequestSyncConfigOnHost(ctx, guest, host, self); err != nil {
			self.SetStageFailed(ctx, err.Error())
			log.Errorf("SyncConfTask faled %v", err)
		}
	}
}

func (self *GuestSyncConfTask) OnSyncComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if fw_only, _ := self.GetParams().Bool("fw_only"); fw_only {
		db.OpsLog.LogEvent(guest, db.ACT_SYNC_CONF, nil, self.UserCred)
		self.SetStageComplete(ctx, guest.GetShortDesc())
	} else if data.Contains("task") {
		self.SetStage("on_disk_sync_complete", nil)
	} else {
		self.OnDiskSyncComplete(ctx, guest, data)
	}
}

func (self *GuestSyncConfTask) OnDiskSyncComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStage("on_sync_status_complete", nil)
	guest.StartSyncstatus(ctx, self.GetUserCred(), self.GetTaskId())
}

func (self *GuestSyncConfTask) OnDiskSyncCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_SYNC_CONF_FAIL, data.String(), self.UserCred)
	log.Errorf("Guest sync config failed: %v", data.String())
}

func (self *GuestSyncConfTask) OnSyncCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.GetUserCred(), models.VM_SYNC_FAIL, data.String())
	log.Errorf("Guest sync config failed: %v", data.String())
	db.OpsLog.LogEvent(guest, db.ACT_SYNC_CONF_FAIL, data.String(), self.UserCred)
}

func (self *GuestSyncConfTask) OnSyncStatusComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}
