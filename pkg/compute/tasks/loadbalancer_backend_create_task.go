package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type LoadbalancerBackendCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerBackendCreateTask{})
}

func (self *LoadbalancerBackendCreateTask) taskFail(ctx context.Context, lbb *models.SLoadbalancerBackend, reason string) {
	lbb.SetStatus(self.GetUserCred(), models.LB_STATUS_CREATE_FAILED, reason)
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
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerBackendCreateTask) OnLoadbalancerBackendCreateCompleteFailed(ctx context.Context, lbb *models.SLoadbalancerBackend, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbb, reason.String())
}
