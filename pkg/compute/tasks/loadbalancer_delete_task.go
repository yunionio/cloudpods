package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/consts"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LoadbalancerDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerDeleteTask{})
}

func (self *LoadbalancerDeleteTask) taskFail(ctx context.Context, lb *models.SLoadbalancer, reason string) {
	lb.SetStatus(self.GetUserCred(), consts.LB_STATUS_DELETE_FAILED, reason)
	db.OpsLog.LogEvent(lb, db.ACT_DELOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lb, logclient.ACT_DELETE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lb.Id, lb.Name, consts.LB_STATUS_DELETE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lb := obj.(*models.SLoadbalancer)
	region := lb.GetRegion()
	if region == nil {
		self.taskFail(ctx, lb, fmt.Sprintf("failed to find region for lb %s", lb.Name))
		return
	}
	self.SetStage("OnLoadbalancerDeleteComplete", nil)
	if err := region.GetDriver().RequestDeleteLoadbalancer(ctx, self.GetUserCred(), lb, self); err != nil {
		self.taskFail(ctx, lb, err.Error())
	}
}

func (self *LoadbalancerDeleteTask) OnLoadbalancerDeleteComplete(ctx context.Context, lb *models.SLoadbalancer, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(lb, db.ACT_DELETE, lb.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lb, logclient.ACT_DELETE, nil, self.UserCred, true)
	lb.PendingDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerDeleteTask) OnLoadbalancerDeleteCompleteFailed(ctx context.Context, lb *models.SLoadbalancer, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lb, reason.String())
}
