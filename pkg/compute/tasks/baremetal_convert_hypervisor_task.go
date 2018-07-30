package tasks

import (
	"context"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/compute/models"
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
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
