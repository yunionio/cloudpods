package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/consts"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LoadbalancerLoadbalancerBackendGroupCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerLoadbalancerBackendGroupCreateTask{})
}

func (self *LoadbalancerLoadbalancerBackendGroupCreateTask) taskFail(ctx context.Context, lbacl *models.SLoadbalancerBackendGroup, reason string) {
	lbacl.SetStatus(self.GetUserCred(), consts.LB_CREATE_FAILED, reason)
	db.OpsLog.LogEvent(lbacl, db.ACT_ALLOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lbacl, logclient.ACT_CREATE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lbacl.Id, lbacl.Name, consts.LB_CREATE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerLoadbalancerBackendGroupCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbbg := obj.(*models.SLoadbalancerBackendGroup)
	region := lbbg.GetRegion()
	if region == nil {
		self.taskFail(ctx, lbbg, fmt.Sprintf("failed to find region for lb backendgroup %s", lbbg.Name))
		return
	}
	backends := []cloudprovider.SLoadbalancerBackend{}
	self.GetParams().Unmarshal(&backends, "backends")
	self.SetStage("OnLoadbalancerBackendGroupCreateComplete", nil)
	if err := region.GetDriver().RequestCreateLoadbalancerBackendGroup(ctx, self.GetUserCred(), lbbg, backends, self); err != nil {
		self.taskFail(ctx, lbbg, err.Error())
	}
}

func (self *LoadbalancerLoadbalancerBackendGroupCreateTask) OnLoadbalancerBackendGroupCreateComplete(ctx context.Context, lbbg *models.SLoadbalancerBackendGroup, data jsonutils.JSONObject) {
	lbbg.SetStatus(self.GetUserCred(), consts.LB_STATUS_ENABLED, "")
	db.OpsLog.LogEvent(lbbg, db.ACT_ALLOCATE, lbbg.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lbbg, logclient.ACT_CREATE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerLoadbalancerBackendGroupCreateTask) OnLoadbalancerBackendGroupCreateCompleteFailed(ctx context.Context, lbbg *models.SLoadbalancerBackendGroup, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbbg, reason.String())
}
