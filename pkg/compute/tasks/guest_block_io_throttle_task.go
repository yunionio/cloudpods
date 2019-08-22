package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type GuestBlockIoThrottleTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestBlockIoThrottleTask{})
}

func (self *GuestBlockIoThrottleTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	url := fmt.Sprintf("/servers/%s/io-throttle", guest.Id)
	headers := self.GetTaskRequestHeader()
	host := guest.GetHost()
	self.SetStage("OnIoThrottle", nil)
	_, err := host.Request(ctx, self.UserCred, "POST", url, headers, self.Params)
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
	}
}

func (self *GuestBlockIoThrottleTask) OnIoThrottle(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetMetadata(ctx, "io-throttle", self.Params.String(), self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *GuestBlockIoThrottleTask) OnIoThrottleFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data.String())
}
