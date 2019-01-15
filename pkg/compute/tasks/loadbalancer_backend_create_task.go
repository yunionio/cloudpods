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

type LoadbalancerBackendCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerBackendCreateTask{})
}

func (self *LoadbalancerBackendCreateTask) taskFail(ctx context.Context, lbb *models.SLoadbalancerBackend, reason string) {
	lbb.SetStatus(self.GetUserCred(), models.LB_CREATE_FAILED, reason)
	db.OpsLog.LogEvent(lbb, db.ACT_ALLOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLog(lbb, logclient.ACT_CREATE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lbb.Id, lbb.Name, models.LB_CREATE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerBackendCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbb := obj.(*models.SLoadbalancerBackend)
	region := lbb.GetRegion()
	if region == nil {
		self.taskFail(ctx, lbb, fmt.Sprintf("failed to find region for lbb %s", lbb.Name))
		return
	}
	self.SetStage("OnLoadbalancerBackendCreateComplete", nil)
	if err := region.GetDriver().RequestCreateLoadbalancerBackend(ctx, self.GetUserCred(), lbb, self); err != nil {
		self.taskFail(ctx, lbb, err.Error())
	}
}

func (self *LoadbalancerBackendCreateTask) OnLoadbalancerBackendCreateComplete(ctx context.Context, lbb *models.SLoadbalancerBackend, data jsonutils.JSONObject) {
	lbb.SetStatus(self.GetUserCred(), models.LB_STATUS_ENABLED, "")
	db.OpsLog.LogEvent(lbb, db.ACT_ALLOCATE, lbb.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLog(lbb, logclient.ACT_CREATE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerBackendCreateTask) OnLoadbalancerBackendCreateCompleteFailed(ctx context.Context, lbb *models.SLoadbalancerBackend, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbb, reason.String())
}
