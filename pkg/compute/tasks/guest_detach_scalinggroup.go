package tasks

import (
	"context"
	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestDetachScalingGroupTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(GuestDetachScalingGroupTask{})
}

func (self *GuestDetachScalingGroupTask) taskFailed(ctx context.Context, sgg *models.SScalingGroupGuest, sg *models.SScalingGroup, reason string) {
	if sgg == nil {
		return
	}
	sgg.SetGuestStatus(api.SG_GUEST_STATUS_REMOVE_FAILED)
	if sg == nil {
		model, _ := models.ScalingGroupManager.FetchById(sgg.ScalingGroupId)
		if model == nil {
			return
		}
		sg = model.(*models.SScalingGroup)
	}
	logclient.AddActionLogWithStartable(self, sg, logclient.ACT_REMOVE_GUEST, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *GuestDetachScalingGroupTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	sgId, _ := self.Params.GetString("scaling_group")

	var sgg *models.SScalingGroupGuest
	sggs, _ := models.ScalingGroupGuestManager.Fetch(sgId, guest.Id)
	if len(sggs) > 0 {
		sgg = &sggs[0]
	}
	// todo: 进行一些准备工作, 比如从负载均衡组移除，从数据库组移除

	delete, _ := self.Params.Bool("delete_server")
	if !delete {
		self.OnDeleteGuestComplete(ctx, guest, body)
		return
	}
	self.SetStage("OnDeleteGuestComplete", nil)
	if err := guest.StartDeleteGuestTask(ctx, self.UserCred, self.Id, true, true, true); err != nil {
		self.taskFailed(ctx, sgg, nil, err.Error())
	}
}

func (self *GuestDetachScalingGroupTask) OnDeleteGuestComplete(ctx context.Context, guest *models.SGuest,
	data jsonutils.JSONObject) {

	sgId, _ := self.Params.GetString("scaling_group")

	var sgg *models.SScalingGroupGuest
	sggs, _ := models.ScalingGroupGuestManager.Fetch(sgId, guest.Id)
	if len(sggs) > 0 {
		sgg = &sggs[0]
	}
	sgg.Detach(ctx, self.UserCred)

	model, _ := models.ScalingGroupManager.FetchById(sgg.ScalingGroupId)
	if model == nil {
		return
	}
	logclient.AddActionLogWithStartable(self, model, logclient.ACT_REMOVE_GUEST, "", self.UserCred, true)
}

func (self *GuestDetachScalingGroupTask) OnDeleteGuestCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	sgId, _ := self.Params.GetString("scaling_group")

	var sgg *models.SScalingGroupGuest
	sggs, _ := models.ScalingGroupGuestManager.Fetch(sgId, guest.Id)
	if len(sggs) > 0 {
		sgg = &sggs[0]
	}
	self.taskFailed(ctx, sgg, nil, "delete guest failed")
}
