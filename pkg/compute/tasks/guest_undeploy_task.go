package tasks

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type GuestUndeployTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestUndeployTask{})
}

func (self *GuestUndeployTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	targetHostId, _ := self.Params.GetString("target_host_id")
	self.SetStage("OnGuestUndeployComplete", nil)
	if len(targetHostId) == 0 {
		if len(guest.BackupHostId) > 0 {
			self.SetStage("OnMasetHostUndeployGuestComplete", nil)
		}
		targetHostId = guest.HostId
	}
	var host *models.SHost
	if len(targetHostId) > 0 {
		host = models.HostManager.FetchHostById(targetHostId)
	}
	if host != nil {
		err := guest.GetDriver().RequestUndeployGuestOnHost(ctx, guest, host, self)
		if err != nil {
			self.OnStartDeleteGuestFail(ctx, err)
		}
	} else {
		self.SetStageComplete(ctx, nil)
	}
}

func (self *GuestUndeployTask) OnMasetHostUndeployGuestComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStage("OnGuestUndeployComplete", nil)
	host := models.HostManager.FetchHostById(guest.BackupHostId)
	if host != nil {
		err := guest.GetDriver().RequestUndeployGuestOnHost(ctx, guest, host, self)
		if err != nil {
			self.OnStartDeleteGuestFail(ctx, err)
		}
	} else {
		self.SetStageComplete(ctx, nil)
	}
}

func (self *GuestUndeployTask) OnGuestUndeployComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestUndeployTask) OnStartDeleteGuestFail(ctx context.Context, err error) {
	jsonErr, ok := err.(*httputils.JSONClientError)
	if ok {
		if jsonErr.Code == 404 {
			self.SetStageComplete(ctx, nil)
			return
		}
	}
	self.SetStageFailed(ctx, err.Error())
}
