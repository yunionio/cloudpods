package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

const (
	CONVERT_TASK = "convert_task"
)

type BaremetalConvertHypervisorTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(BaremetalConvertHypervisorTask{})
}

func (self *BaremetalConvertHypervisorTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	baremetal := obj.(*models.SHost)

	baremetal.SetStatus(self.UserCred, models.HOST_STATUS_CONVERTING, "")

	self.SetStage("on_guest_deploy_complete", nil)

	guestId, _ := self.Params.GetString("server_id")
	guestObj, err := models.GuestManager.FetchById(guestId)
	if err != nil { // unlikely
		log.Errorf("fail to find server %s", guestId)
		return
	}

	guest := guestObj.(*models.SGuest)
	params, _ := self.Params.Get("server_params")
	paramsDict := params.(*jsonutils.JSONDict)
	pendingUsage := models.SQuota{}
	self.GetPendingUsage(&pendingUsage)
	guest.StartGuestCreateTask(ctx, self.UserCred, paramsDict, &pendingUsage, self.GetId())
}
