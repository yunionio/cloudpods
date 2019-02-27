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

type LoadbalancerListenerStartTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerListenerStartTask{})
}

func (self *LoadbalancerListenerStartTask) taskFail(ctx context.Context, lblis *models.SLoadbalancerListener, reason string) {
	lblis.SetStatus(self.GetUserCred(), consts.LB_STATUS_DISABLED, reason)
	db.OpsLog.LogEvent(lblis, db.ACT_ENABLE, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_ENABLE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lblis.Id, lblis.Name, consts.LB_STATUS_DISABLED, reason)
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
	lblis.SetStatus(self.GetUserCred(), consts.LB_STATUS_ENABLED, "")
	db.OpsLog.LogEvent(lblis, db.ACT_ENABLE, lblis.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_ENABLE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerListenerStartTask) OnLoadbalancerListenerStartCompleteFailed(ctx context.Context, lblis *models.SLoadbalancerListener, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lblis, reason.String())
}
