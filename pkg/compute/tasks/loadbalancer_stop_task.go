package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LoadbalancerStopTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerStopTask{})
}

func (self *LoadbalancerStopTask) taskFail(ctx context.Context, lb *models.SLoadbalancer, reason string) {
	lb.SetStatus(self.GetUserCred(), models.LB_STATUS_ENABLED, reason)
	db.OpsLog.LogEvent(lb, db.ACT_DISABLE, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lb, logclient.ACT_DISABLE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lb.Id, lb.Name, models.LB_STATUS_ENABLED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerStopTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lb := obj.(*models.SLoadbalancer)
	region := lb.GetRegion()
	if region == nil {
		self.taskFail(ctx, lb, fmt.Sprintf("failed to find region for lb %s", lb.Name))
		return
	}
	self.SetStage("OnLoadbalancerStopComplete", nil)
	if err := region.GetDriver().RequestStopLoadbalancer(ctx, self.GetUserCred(), lb, self); err != nil {
		self.taskFail(ctx, lb, err.Error())
	}
}

func (self *LoadbalancerStopTask) OnLoadbalancerStopComplete(ctx context.Context, lb *models.SLoadbalancer, data jsonutils.JSONObject) {
	lb.SetStatus(self.GetUserCred(), models.LB_STATUS_DISABLED, "")
	db.OpsLog.LogEvent(lb, db.ACT_DISABLE, lb.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lb, logclient.ACT_DISABLE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerStopTask) OnLoadbalancerStopCompleteFailed(ctx context.Context, lb *models.SLoadbalancer, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lb, reason.String())
}
