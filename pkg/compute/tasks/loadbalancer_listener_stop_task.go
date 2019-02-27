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

type LoadbalancerListenerStopTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerListenerStopTask{})
}

func (self *LoadbalancerListenerStopTask) taskFail(ctx context.Context, lblis *models.SLoadbalancerListener, reason string) {
	lblis.SetStatus(self.GetUserCred(), consts.LB_STATUS_ENABLED, reason)
	db.OpsLog.LogEvent(lblis, db.ACT_DISABLE, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_DISABLE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lblis.Id, lblis.Name, consts.LB_STATUS_ENABLED, reason)
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
	lblis.SetStatus(self.GetUserCred(), consts.LB_STATUS_DISABLED, "")
	db.OpsLog.LogEvent(lblis, db.ACT_DISABLE, lblis.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_DISABLE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerListenerStopTask) OnLoadbalancerListenerStopCompleteFailed(ctx context.Context, lblis *models.SLoadbalancerListener, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lblis, reason.String())
}
