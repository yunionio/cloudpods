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

type LoadbalancerStartTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerStartTask{})
}

func (self *LoadbalancerStartTask) taskFail(ctx context.Context, lb *models.SLoadbalancer, reason string) {
	lb.SetStatus(self.GetUserCred(), models.LB_STATUS_DISABLED, reason)
	db.OpsLog.LogEvent(lb, db.ACT_ENABLE, reason, self.UserCred)
	logclient.AddActionLog(lb, logclient.ACT_ENABLE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lb.Id, lb.Name, models.LB_STATUS_DISABLED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerStartTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lb := obj.(*models.SLoadbalancer)
	region := lb.GetRegion()
	if region == nil {
		self.taskFail(ctx, lb, fmt.Sprintf("failed to find region for lb %s", lb.Name))
		return
	}
	self.SetStage("OnLoadbalancerStartComplete", nil)
	if err := region.GetDriver().RequestStartLoadbalancer(ctx, self.GetUserCred(), lb, self); err != nil {
		self.taskFail(ctx, lb, err.Error())
	}
}

func (self *LoadbalancerStartTask) OnLoadbalancerStartComplete(ctx context.Context, lb *models.SLoadbalancer, data jsonutils.JSONObject) {
	lb.SetStatus(self.GetUserCred(), models.LB_STATUS_ENABLED, "")
	db.OpsLog.LogEvent(lb, db.ACT_ENABLE, lb.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLog(lb, logclient.ACT_ENABLE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerStartTask) OnLoadbalancerStartCompleteFailed(ctx context.Context, lb *models.SLoadbalancer, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lb, reason.String())
}
