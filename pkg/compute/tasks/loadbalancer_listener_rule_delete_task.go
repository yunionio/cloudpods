package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type LoadbalancerListenerRuleDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerListenerRuleDeleteTask{})
}

func (self *LoadbalancerListenerRuleDeleteTask) taskFail(ctx context.Context, lbr *models.SLoadbalancerListenerRule, reason string) {
	lbr.SetStatus(self.GetUserCred(), models.LB_STATUS_DELETE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerListenerRuleDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbr := obj.(*models.SLoadbalancerListenerRule)
	region := lbr.GetRegion()
	if region == nil {
		self.taskFail(ctx, lbr, fmt.Sprintf("failed to find region for lbr %s", lbr.Name))
		return
	}
	self.SetStage("OnLoadbalancerListenerRuleDeleteComplete", nil)
	if err := region.GetDriver().RequestDeleteLoadbalancerListenerRule(ctx, self.GetUserCred(), lbr, self); err != nil {
		self.taskFail(ctx, lbr, err.Error())
	}
}

func (self *LoadbalancerListenerRuleDeleteTask) OnLoadbalancerListenerRuleDeleteComplete(ctx context.Context, lbr *models.SLoadbalancerListenerRule, data jsonutils.JSONObject) {
	lbr.DoPendingDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerListenerRuleDeleteTask) OnLoadbalancerListenerRuleDeleteCompleteFailed(ctx context.Context, lbr *models.SLoadbalancerListenerRule, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbr, reason.String())
}
