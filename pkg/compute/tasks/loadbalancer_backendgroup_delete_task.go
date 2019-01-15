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

type LoadbalancerBackendGroupDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerBackendGroupDeleteTask{})
}

func (self *LoadbalancerBackendGroupDeleteTask) taskFail(ctx context.Context, lbbg *models.SLoadbalancerBackendGroup, reason string) {
	lbbg.SetStatus(self.GetUserCred(), models.LB_STATUS_DELETE_FAILED, reason)
	db.OpsLog.LogEvent(lbbg, db.ACT_DELOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLog(lbbg, logclient.ACT_DELETE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lbbg.Id, lbbg.Name, models.LB_STATUS_DELETE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerBackendGroupDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbbg := obj.(*models.SLoadbalancerBackendGroup)
	region := lbbg.GetRegion()
	if region == nil {
		self.taskFail(ctx, lbbg, fmt.Sprintf("failed to find region for lb %s", lbbg.Name))
		return
	}
	self.SetStage("OnLoadbalancerBackendGroupDeleteComplete", nil)
	if err := region.GetDriver().RequestDeleteLoadbalancerBackendGroup(ctx, self.GetUserCred(), lbbg, self); err != nil {
		self.taskFail(ctx, lbbg, err.Error())
	}
}

func (self *LoadbalancerBackendGroupDeleteTask) OnLoadbalancerBackendGroupDeleteComplete(ctx context.Context, lbbg *models.SLoadbalancerBackendGroup, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(lbbg, db.ACT_DELETE, lbbg.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLog(lbbg, logclient.ACT_DELETE, nil, self.UserCred, true)
	lbbg.PreDeleteSubs(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerBackendGroupDeleteTask) OnLoadbalancerBackendGroupDeleteCompleteFailed(ctx context.Context, lbbg *models.SLoadbalancerBackendGroup, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbbg, reason.String())
}
