package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

func init() {
	taskman.RegisterTask(HAGuestStopTask{})
}

type HAGuestStopTask struct {
	GuestStopTask
}

func (self *HAGuestStopTask) OnGuestStopTaskComplete(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	host := models.HostManager.FetchHostById(guest.BackupHostId)
	self.SetStage("OnSlaveGuestStopTaskComplete", nil)
	err := guest.GetDriver().RequestStopOnHost(ctx, guest, host, self)
	if err != nil {
		log.Errorf("RequestStopOnHost fail %s", err)
		self.OnGuestStopTaskCompleteFailed(
			ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (self *HAGuestStopTask) OnSlaveGuestStopTaskComplete(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	self.GuestStopTask.OnGuestStopTaskComplete(ctx, guest, data)
}

func (self *HAGuestStopTask) OnSlaveGuestStopTaskCompleteFailed(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	self.OnGuestStopTaskCompleteFailed(ctx, guest, data)
}
