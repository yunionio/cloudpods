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

type LoadbalancerListenerDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerListenerDeleteTask{})
}

func (self *LoadbalancerListenerDeleteTask) taskFail(ctx context.Context, lblis *models.SLoadbalancerListener, reason string) {
	lblis.SetStatus(self.GetUserCred(), consts.LB_STATUS_DELETE_FAILED, reason)
	db.OpsLog.LogEvent(lblis, db.ACT_DELOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_DELETE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lblis.Id, lblis.Name, consts.LB_STATUS_DELETE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerListenerDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lblis := obj.(*models.SLoadbalancerListener)
	region := lblis.GetRegion()
	if region == nil {
		self.taskFail(ctx, lblis, fmt.Sprintf("failed to find region for lblis %s", lblis.Name))
		return
	}
	self.SetStage("OnLoadbalancerListenerDeleteComplete", nil)
	if err := region.GetDriver().RequestDeleteLoadbalancerListener(ctx, self.GetUserCred(), lblis, self); err != nil {
		self.taskFail(ctx, lblis, err.Error())
	}
}

func (self *LoadbalancerListenerDeleteTask) OnLoadbalancerListenerDeleteComplete(ctx context.Context, lblis *models.SLoadbalancerListener, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(lblis, db.ACT_DELETE, lblis.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lblis, logclient.ACT_DELETE, nil, self.UserCred, true)
	lblis.PreDeleteSubs(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerListenerDeleteTask) OnLoadbalancerListenerDeleteCompleteFailed(ctx context.Context, lblis *models.SLoadbalancerListener, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lblis, reason.String())
}
