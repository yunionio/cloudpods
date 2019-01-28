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

type BaremetalServerRebuildRootTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(BaremetalServerRebuildRootTask{})
}

func (self *BaremetalServerRebuildRootTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if jsonutils.QueryBoolean(self.Params, "need_stop", false) {
		self.SetStage("OnStopServerComplete", nil)
		guest.StartGuestStopTask(ctx, self.UserCred, false, self.GetTaskId())
		return
	}
	self.StartRebuildRootDisk(ctx, guest)
}

func (self *BaremetalServerRebuildRootTask) OnStopServerComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.StartRebuildRootDisk(ctx, guest)
}

func (self *BaremetalServerRebuildRootTask) StartRebuildRootDisk(ctx context.Context, guest *models.SGuest) {
	if guest.Status != models.VM_ADMIN {
		guest.SetStatus(self.UserCred, models.VM_REBUILD_ROOT, "")
	}
	imageId, _ := self.Params.GetString("image_id")
	db.OpsLog.LogEvent(guest, db.ACT_REBUILDING_ROOT, imageId, self.UserCred)
	gds := guest.CategorizeDisks()
	oldStatus := gds.Root.Status
	_, err := gds.Root.GetModelManager().TableSpec().Update(gds.Root, func() error {
		gds.Root.TemplateId = imageId
		gds.Root.Status = models.DISK_REBUILD
		return nil
	})
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
		logclient.AddActionLog(guest, logclient.ACT_VM_REBUILD, err, self.UserCred, false)
		return
	} else {
		db.OpsLog.LogEvent(gds.Root, db.ACT_UPDATE_STATUS,
			fmt.Sprintf("%s=>%s", oldStatus, models.DISK_REBUILD), self.UserCred)
	}
	self.SetStage("OnRebuildRootDiskComplete", nil)

	// clear logininfo
	loginParams := make(map[string]interface{})
	loginParams["login_account"] = "none"
	loginParams["login_key"] = "none"
	loginParams["login_key_timestamp"] = "none"
	guest.SetAllMetadata(ctx, loginParams, self.UserCred)
	guest.StartGuestDeployTask(ctx, self.UserCred, self.Params, "rebuild", self.GetTaskId())
}

func (self *BaremetalServerRebuildRootTask) OnRebuildRootDiskComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_REBUILD_ROOT, "", self.UserCred)
	self.SetStage("OnSyncStatusComplete", nil)
	guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
}

func (self *BaremetalServerRebuildRootTask) OnRebuildRootDiskCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_REBUILD_ROOT_FAIL, data, self.UserCred)
	if guest.Status != models.VM_ADMIN {
		guest.SetStatus(self.UserCred, models.VM_REBUILD_ROOT_FAIL, "")
	}
}

func (self *BaremetalServerRebuildRootTask) OnSyncStatusComplete(ctx context.Context, _ *models.SGuest, _ jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}
