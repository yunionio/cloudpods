package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type LoadbalancerListenerStopTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerListenerStopTask{})
}

func (self *LoadbalancerListenerStopTask) taskFail(ctx context.Context, lblis *models.SLoadbalancerListener, reason string) {
	lblis.SetStatus(self.GetUserCred(), models.LB_STATUS_UNKNOWN, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerListenerStopTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lblis := obj.(*models.SLoadbalancerListener)
	region := lblis.GetRegion()
	if region == nil {
		self.taskFail(ctx, lblis, fmt.Sprintf("failed to find region for lblis %s", lblis.Name))
		return
	}
	self.SetStage("OnLoadbalancerListenerStopComplete", nil)
	if err := region.GetDriver().RequestStopLoadbalancerListener(ctx, self.GetUserCred(), lblis, self); err != nil {
		self.taskFail(ctx, lblis, err.Error())
	}
}

func (self *LoadbalancerListenerStopTask) OnLoadbalancerListenerStopComplete(ctx context.Context, lblis *models.SLoadbalancerListener, data jsonutils.JSONObject) {
	lblis.SetStatus(self.GetUserCred(), models.LB_STATUS_DISABLED, "")
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerListenerStopTask) OnLoadbalancerListenerStopCompleteFailed(ctx context.Context, lblis *models.SLoadbalancerListener, reason jsonutils.JSONObject) {
	lblis.SetStatus(self.GetUserCred(), models.LB_STATUS_START_FAILED, reason.String())
	self.SetStage("OnLoadbalancerListenerSyncStatusComplete", nil)
	lblis.StartLoadBalancerListenerSyncstatusTask(ctx, self.GetUserCred(), self.GetTaskId())
}

func (self *LoadbalancerListenerStopTask) OnLoadbalancerListenerSyncStatusComplete(ctx context.Context, lblis *models.SLoadbalancerListener, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerListenerStopTask) OnLoadbalancerListenerSyncStatusCompleteFailed(ctx context.Context, lblis *models.SLoadbalancerListener, data jsonutils.JSONObject) {
	self.taskFail(ctx, lblis, data.String())
}
