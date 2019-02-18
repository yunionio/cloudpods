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

type LoadbalancerAclDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerAclDeleteTask{})
}

func (self *LoadbalancerAclDeleteTask) taskFail(ctx context.Context, lbacl *models.SLoadbalancerAcl, reason string) {
	lbacl.SetStatus(self.GetUserCred(), models.LB_STATUS_DELETE_FAILED, reason)
	db.OpsLog.LogEvent(lbacl, db.ACT_DELOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lbacl, logclient.ACT_DELETE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lbacl.Id, lbacl.Name, models.LB_STATUS_DELETE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerAclDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbacl := obj.(*models.SLoadbalancerAcl)
	region := lbacl.GetRegion()
	if region == nil {
		self.taskFail(ctx, lbacl, fmt.Sprintf("failed to find region for lbacl %s", lbacl.Name))
		return
	}
	self.SetStage("OnLoadbalancerAclDeleteComplete", nil)
	if err := region.GetDriver().RequestDeleteLoadbalancerAcl(ctx, self.GetUserCred(), lbacl, self); err != nil {
		self.taskFail(ctx, lbacl, err.Error())
	}
}

func (self *LoadbalancerAclDeleteTask) OnLoadbalancerAclDeleteComplete(ctx context.Context, lbacl *models.SLoadbalancerAcl, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(lbacl, db.ACT_DELETE, lbacl.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lbacl, logclient.ACT_DELETE, nil, self.UserCred, true)
	lbacl.DoPendingDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerAclDeleteTask) OnLoadbalancerAclDeleteCompleteFailed(ctx context.Context, lbacl *models.SLoadbalancerAcl, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbacl, reason.String())
}
