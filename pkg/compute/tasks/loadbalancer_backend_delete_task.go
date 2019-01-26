package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type LoadbalancerBackendDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerBackendDeleteTask{})
}

func (self *LoadbalancerBackendDeleteTask) taskFail(ctx context.Context, lbb *models.SLoadbalancerBackend, reason string) {
	lbb.SetStatus(self.GetUserCred(), models.LB_STATUS_DELETE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerBackendDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbb := obj.(*models.SLoadbalancerBackend)
	region := lbb.GetRegion()
	if region == nil {
		self.taskFail(ctx, lbb, fmt.Sprintf("failed to find region for lbb %s", lbb.Name))
		return
	}
	self.SetStage("OnLoadbalancerBackendDeleteComplete", nil)
	if err := region.GetDriver().RequestDeleteLoadbalancerBackend(ctx, self.GetUserCred(), lbb, self); err != nil {
		self.taskFail(ctx, lbb, err.Error())
	}
}

func (self *LoadbalancerBackendDeleteTask) OnLoadbalancerBackendDeleteComplete(ctx context.Context, lbb *models.SLoadbalancerBackend, data jsonutils.JSONObject) {
	lbb.DoPendingDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerBackendDeleteTask) OnLoadbalancerBackendDeleteCompleteFailed(ctx context.Context, lbb *models.SLoadbalancerBackend, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbb, reason.String())
}
