package tasks

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type GuestInsertIsoTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestInsertIsoTask{})
}

func (self *GuestInsertIsoTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.prepareIsoImage(ctx, obj)
}

func (self *GuestInsertIsoTask) prepareIsoImage(ctx context.Context, obj db.IStandaloneModel) {
	imageId, _ := self.Params.GetString("image_id")
	db.OpsLog.LogEvent(obj, db.ACT_ISO_PREPARING, imageId, self.UserCred)
	var host *models.SHost
	if self.Params.Contains("host_id") {
		hostId, _ := self.Params.GetString("host_id")
		iHost, _ := models.HostManager.FetchById(hostId)
		host = iHost.(*models.SHost)
	} else {
		guest := obj.(*models.SGuest)
		host = guest.GetHost()
	}
	self.SetStage("OnIsoPrepareComplete", nil)
	host.StartImageCacheTask(ctx, self.UserCred, imageId, self.GetTaskId(), false)
}

func (self *GuestInsertIsoTask) OnIsoPrepareCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	imageId, _ := self.Params.GetString("image_id")
	db.OpsLog.LogEvent(obj, db.ACT_ISO_PREPARE_FAIL, imageId, self.UserCred)
	guest := obj.(*models.SGuest)
	guest.EjectIso(self.UserCred)
	self.SetStageFailed(ctx, "OnIsoPrepareCompleteFailed")
}

func (self *GuestInsertIsoTask) OnIsoPrepareComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	imageId, _ := data.GetString("image_id")
	jSize, err := data.Get("size")
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
	}
	size, err := jSize.Int()
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
	}
	name, _ := data.GetString("name")
	path, _ := data.GetString("path")
	guest := obj.(*models.SGuest)
	if guest.InsertIsoSucc(imageId, path, int(size), name) {
		db.OpsLog.LogEvent(guest, db.ACT_ISO_ATTACH, guest.GetDetailsIso(self.UserCred), self.UserCred)
		if guest.Status == models.VM_RUNNING {
			self.SetStage("OnConfigSyncComplete", nil)
			guest.GetDriver().RequestGuestHotAddIso(ctx, guest, path, self)
		} else {
			self.SetStageComplete(ctx, nil)
		}
	} else {
		self.SetStageComplete(ctx, nil)
	}
}

func (self *GuestInsertIsoTask) OnConfigSyncComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}
