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

type LoadbalancerSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerSyncstatusTask{})
}

func (self *LoadbalancerSyncstatusTask) taskFail(ctx context.Context, lb *models.SLoadbalancer, reason string) {
	lb.SetStatus(self.GetUserCred(), models.LB_STATUS_UNKNOWN, reason)
	db.OpsLog.LogEvent(lb, db.ACT_SYNC_STATUS, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lb, logclient.ACT_SYNC_STATUS, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lb.Id, lb.Name, models.LB_SYNC_CONF_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lb := obj.(*models.SLoadbalancer)
	region := lb.GetRegion()
	if region == nil {
		self.taskFail(ctx, lb, fmt.Sprintf("failed to find region for lb %s", lb.Name))
		return
	}
	self.SetStage("OnLoadbalancerSyncstatusComplete", nil)
	if err := region.GetDriver().RequestSyncstatusLoadbalancer(ctx, self.GetUserCred(), lb, self); err != nil {
		self.taskFail(ctx, lb, err.Error())
	}
}

func (self *LoadbalancerSyncstatusTask) OnLoadbalancerSyncstatusComplete(ctx context.Context, lb *models.SLoadbalancer, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(lb, db.ACT_SYNC_STATUS, lb.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lb, logclient.ACT_SYNC_STATUS, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerSyncstatusTask) OnLoadbalancerSyncstatusCompleteFailed(ctx context.Context, lb *models.SLoadbalancer, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lb, reason.String())
}
