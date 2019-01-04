package tasks

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type GuestEjectISOTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestEjectISOTask{})
}

func (self *GuestEjectISOTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.startEjectIso(ctx, obj)
}

func (self *GuestEjectISOTask) startEjectIso(ctx context.Context, obj db.IStandaloneModel) {
	guest := obj.(*models.SGuest)
	if guest.EjectIso(self.UserCred) && guest.Status == models.VM_RUNNING {
		self.SetStage("OnConfigSyncComplete", nil)
		guest.StartSyncTask(ctx, self.UserCred, false, self.GetId())
	} else {
		self.SetStageComplete(ctx, nil)
	}
}

func (self *GuestEjectISOTask) OnConfigSyncComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}
