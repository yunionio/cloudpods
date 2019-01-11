package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type LoadbalancerAclSyncTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerAclSyncTask{})
}

func (self *LoadbalancerAclSyncTask) taskFail(ctx context.Context, lbacl *models.SLoadbalancerAcl, reason string) {
	lbacl.SetStatus(self.GetUserCred(), models.LB_STATUS_SYNCING_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerAclSyncTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbacl := obj.(*models.SLoadbalancerAcl)
	region := lbacl.GetRegion()
	if region == nil {
		self.taskFail(ctx, lbacl, fmt.Sprintf("failed to find region for lbacl %s", lbacl.Name))
		return
	}
	self.SetStage("OnLoadbalancerAclSyncComplete", nil)
	if err := region.GetDriver().RequestSyncLoadbalancerAcl(ctx, self.GetUserCred(), lbacl, self); err != nil {
		self.taskFail(ctx, lbacl, err.Error())
	}
}

func (self *LoadbalancerAclSyncTask) OnLoadbalancerAclSyncComplete(ctx context.Context, lbacl *models.SLoadbalancerAcl, data jsonutils.JSONObject) {
	lbacl.SetStatus(self.GetUserCred(), models.LB_STATUS_ENABLED, "")
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerAclSyncTask) OnLoadbalancerAclSyncCompleteFailed(ctx context.Context, lbacl *models.SLoadbalancerAcl, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbacl, reason.String())
}
