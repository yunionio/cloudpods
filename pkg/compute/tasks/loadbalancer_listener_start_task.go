package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type LoadbalancerListenerStartTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerListenerStartTask{})
}

func (self *LoadbalancerListenerStartTask) taskFail(ctx context.Context, lblis *models.SLoadbalancerListener, reason string) {
	lblis.SetStatus(self.GetUserCred(), models.LB_STATUS_UNKNOWN, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerListenerStartTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lblis := obj.(*models.SLoadbalancerListener)
	region := lblis.GetRegion()
	if region == nil {
		self.taskFail(ctx, lblis, fmt.Sprintf("failed to find region for lblis %s", lblis.Name))
		return
	}
	self.SetStage("OnLoadbalancerListenerStartComplete", nil)
	if err := region.GetDriver().RequestStartLoadbalancerListener(ctx, self.GetUserCred(), lblis, self); err != nil {
		self.taskFail(ctx, lblis, err.Error())
	}
}

func (self *LoadbalancerListenerStartTask) OnLoadbalancerListenerStartComplete(ctx context.Context, lblis *models.SLoadbalancerListener, data jsonutils.JSONObject) {
	lblis.SetStatus(self.GetUserCred(), models.LB_STATUS_ENABLED, "")
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerListenerStartTask) OnLoadbalancerListenerStartCompleteFailed(ctx context.Context, lblis *models.SLoadbalancerListener, reason jsonutils.JSONObject) {
	lblis.SetStatus(self.GetUserCred(), models.LB_STATUS_START_FAILED, reason.String())
	self.SetStage("OnLoadbalancerListenerSyncStatusComplete", nil)
	lblis.StartLoadBalancerListenerSyncstatusTask(ctx, self.GetUserCred(), self.GetTaskId())
}

func (self *LoadbalancerListenerStartTask) OnLoadbalancerListenerSyncStatusComplete(ctx context.Context, lblis *models.SLoadbalancerListener, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerListenerStartTask) OnLoadbalancerListenerSyncStatusCompleteFailed(ctx context.Context, lblis *models.SLoadbalancerListener, data jsonutils.JSONObject) {
	self.taskFail(ctx, lblis, data.String())
}
