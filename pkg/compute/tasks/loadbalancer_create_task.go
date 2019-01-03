package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type LoadbalancerCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerCreateTask{})
}

func (self *LoadbalancerCreateTask) taskFail(ctx context.Context, lb *models.SLoadbalancer, reason string) {
	lb.SetStatus(self.GetUserCred(), models.LB_STATUS_CREATE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lb := obj.(*models.SLoadbalancer)
	region := lb.GetRegion()
	if region == nil {
		self.taskFail(ctx, lb, fmt.Sprintf("failed to find region for lb %s", lb.Name))
		return
	}
	self.SetStage("OnLoadbalancerCreateComplete", nil)
	if err := region.GetDriver().RequestCreateLoadbalancer(ctx, self.GetUserCred(), lb, self); err != nil {
		self.taskFail(ctx, lb, err.Error())
	}
}

func (self *LoadbalancerCreateTask) OnLoadbalancerCreateComplete(ctx context.Context, lb *models.SLoadbalancer, data jsonutils.JSONObject) {
	lb.SetStatus(self.GetUserCred(), models.LB_STATUS_ENABLED, "")
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerCreateTask) OnLoadbalancerCreateCompleteFailed(ctx context.Context, lb *models.SLoadbalancer, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lb, reason.String())
}
