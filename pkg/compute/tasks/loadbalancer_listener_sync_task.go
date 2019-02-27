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

type LoadbalancerListenerSyncTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerListenerSyncTask{})
}

func (self *LoadbalancerListenerSyncTask) taskFail(ctx context.Context, lblis *models.SLoadbalancerListener, reason string) {
	lblis.SetStatus(self.GetUserCred(), consts.LB_SYNC_CONF_FAILED, reason)
	db.OpsLog.LogEvent(lblis, db.ACT_SYNC_CONF, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_SYNC_CONF, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lblis.Id, lblis.Name, consts.LB_SYNC_CONF_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerListenerSyncTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lblis := obj.(*models.SLoadbalancerListener)
	region := lblis.GetRegion()
	if region == nil {
		self.taskFail(ctx, lblis, fmt.Sprintf("failed to find region for lblis %s", lblis.Name))
		return
	}
	self.SetStage("OnLoadbalancerListenerSyncComplete", nil)
	if err := region.GetDriver().RequestSyncLoadbalancerListener(ctx, self.GetUserCred(), lblis, self); err != nil {
		self.taskFail(ctx, lblis, err.Error())
	}
}

func (self *LoadbalancerListenerSyncTask) OnLoadbalancerListenerSyncComplete(ctx context.Context, lblis *models.SLoadbalancerListener, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(lblis, db.ACT_SYNC_CONF, lblis.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_SYNC_CONF, nil, self.UserCred, true)
	self.SetStage("OnLoadbalancerListenerSyncStatusComplete", nil)
	lblis.StartLoadBalancerListenerSyncstatusTask(ctx, self.GetUserCred(), self.GetParams(), self.GetTaskId())
}

func (self *LoadbalancerListenerSyncTask) OnLoadbalancerListenerSyncCompleteFailed(ctx context.Context, lblis *models.SLoadbalancerListener, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lblis, reason.String())
}

func (self *LoadbalancerListenerSyncTask) OnLoadbalancerListenerSyncStatusComplete(ctx context.Context, lblis *models.SLoadbalancerListener, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerListenerSyncTask) OnLoadbalancerListenerSyncStatusCompleteFailed(ctx context.Context, lblis *models.SLoadbalancerListener, reason jsonutils.JSONObject) {
	lblis.SetStatus(self.GetUserCred(), consts.LB_STATUS_UNKNOWN, reason.String())
	self.SetStageFailed(ctx, reason.String())
}
