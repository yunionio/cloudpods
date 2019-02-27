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

type LoadbalancerAclSyncTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerAclSyncTask{})
}

func (self *LoadbalancerAclSyncTask) taskFail(ctx context.Context, lbacl *models.SLoadbalancerAcl, reason string) {
	lbacl.SetStatus(self.GetUserCred(), consts.LB_SYNC_CONF_FAILED, reason)
	db.OpsLog.LogEvent(lbacl, db.ACT_SYNC_CONF, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lbacl, logclient.ACT_SYNC_CONF, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lbacl.Id, lbacl.Name, consts.LB_SYNC_CONF_FAILED, reason)
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
	lbacl.SetStatus(self.GetUserCred(), consts.LB_STATUS_ENABLED, "")
	db.OpsLog.LogEvent(lbacl, db.ACT_SYNC_CONF, lbacl.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lbacl, logclient.ACT_SYNC_CONF, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerAclSyncTask) OnLoadbalancerAclSyncCompleteFailed(ctx context.Context, lbacl *models.SLoadbalancerAcl, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbacl, reason.String())
}
