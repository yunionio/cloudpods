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

type LoadbalancerListenerCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerListenerCreateTask{})
}

func (self *LoadbalancerListenerCreateTask) taskFail(ctx context.Context, lblis *models.SLoadbalancerListener, reason string) {
	lblis.SetStatus(self.GetUserCred(), models.LB_CREATE_FAILED, reason)
	db.OpsLog.LogEvent(lblis, db.ACT_ALLOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLog(lblis, logclient.ACT_CREATE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lblis.Id, lblis.Name, models.LB_CREATE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerListenerCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lblis := obj.(*models.SLoadbalancerListener)
	region := lblis.GetRegion()
	if region == nil {
		self.taskFail(ctx, lblis, fmt.Sprintf("failed to find region for lblis %s", lblis.Name))
		return
	}
	self.SetStage("OnLoadbalancerListenerCreateComplete", nil)
	if err := region.GetDriver().RequestCreateLoadbalancerListener(ctx, self.GetUserCred(), lblis, self); err != nil {
		self.taskFail(ctx, lblis, err.Error())
	}
}

func (self *LoadbalancerListenerCreateTask) OnLoadbalancerListenerCreateComplete(ctx context.Context, lblis *models.SLoadbalancerListener, data jsonutils.JSONObject) {
	lblis.SetStatus(self.GetUserCred(), models.LB_STATUS_ENABLED, "")
	db.OpsLog.LogEvent(lblis, db.ACT_ALLOCATE, lblis.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLog(lblis, logclient.ACT_CREATE, nil, self.UserCred, true)
	self.SetStage("OnLoadbalancerListenerStartComplete", nil)
	lblis.StartLoadBalancerListenerStartTask(ctx, self.GetUserCred(), self.GetTaskId())
}

func (self *LoadbalancerListenerCreateTask) OnLoadbalancerListenerCreateCompleteFailed(ctx context.Context, lblis *models.SLoadbalancerListener, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lblis, reason.String())
}

func (self *LoadbalancerListenerCreateTask) OnLoadbalancerListenerStartComplete(ctx context.Context, lblis *models.SLoadbalancerListener, data jsonutils.JSONObject) {
	lblis.SetStatus(self.GetUserCred(), models.LB_STATUS_ENABLED, "")
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerListenerCreateTask) OnLoadbalancerListenerStartCompleteFailed(ctx context.Context, lblis *models.SLoadbalancerListener, reason jsonutils.JSONObject) {
	lblis.SetStatus(self.GetUserCred(), models.LB_STATUS_DISABLED, reason.String())
	self.SetStageFailed(ctx, reason.String())
}
