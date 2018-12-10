package tasks

import (
	"context"

	"yunion.io/x/jsonutils"

	"fmt"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/billing"
)

type GuestRenewTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestRenewTask{})
}

func (self *GuestRenewTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	durationStr, _ := self.GetParams().GetString("duration")
	bc, _ := billing.ParseBillingCycle(durationStr)

	exp, err := guest.GetDriver().RequestRenewInstance(guest, bc)
	if err != nil {
		msg := fmt.Sprintf("RequestRenewInstance failed %s", err)
		log.Errorf(msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	err = guest.SaveRenewInfo(self.UserCred, &bc, &exp)
	if err != nil {
		msg := fmt.Sprintf("SaveRenewInfo fail %s", err)
		log.Errorf(msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	self.SetStageComplete(ctx, nil)
}
