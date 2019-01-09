package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
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
	taskman.RegisterTask(PrepaidRecycleHostRenewTask{})
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

	err = guest.SaveRenewInfo(ctx, self.UserCred, &bc, &exp)
	if err != nil {
		msg := fmt.Sprintf("SaveRenewInfo fail %s", err)
		log.Errorf(msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	self.SetStageComplete(ctx, nil)
}

type PrepaidRecycleHostRenewTask struct {
	taskman.STask
}

func (self *PrepaidRecycleHostRenewTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	host := obj.(*models.SHost)

	durationStr, _ := self.GetParams().GetString("duration")
	bc, _ := billing.ParseBillingCycle(durationStr)

	ihost, err := host.GetIHost()
	if err != nil {
		msg := fmt.Sprintf("host.GetIHost fail %s", err)
		log.Errorf(msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	iVM, err := ihost.GetIVMById(host.RealExternalId)
	if err != nil {
		msg := fmt.Sprintf("ihost.GetIVMById fail %s", err)
		log.Errorf(msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	log.Debugf("expire before %s", iVM.GetExpiredAt())

	err = iVM.Renew(bc)
	if err != nil {
		msg := fmt.Sprintf("iVM.Renew fail %s", err)
		log.Errorf(msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	err = iVM.Refresh()
	if err != nil {
		msg := fmt.Sprintf("refresh after renew fail %s", err)
		log.Errorf(msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	log.Debugf("expire after %s", iVM.GetExpiredAt())

	exp := iVM.GetExpiredAt()

	err = host.DoSaveRenewInfo(ctx, self.UserCred, &bc, &exp)
	if err != nil {
		msg := fmt.Sprintf("SaveRenewInfo fail %s", err)
		log.Errorf(msg)
		self.SetStageFailed(ctx, msg)
		return
	}

	self.SetStageComplete(ctx, nil)
}
