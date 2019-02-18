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

type LoadbalancerCertificateDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerCertificateDeleteTask{})
}

func (self *LoadbalancerCertificateDeleteTask) taskFail(ctx context.Context, lbcert *models.SLoadbalancerCertificate, reason string) {
	lbcert.SetStatus(self.GetUserCred(), models.LB_STATUS_DELETE_FAILED, reason)
	db.OpsLog.LogEvent(lbcert, db.ACT_DELOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lbcert, logclient.ACT_DELETE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lbcert.Id, lbcert.Name, models.LB_STATUS_DELETE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerCertificateDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbcert := obj.(*models.SLoadbalancerCertificate)
	region := lbcert.GetRegion()
	if region == nil {
		self.taskFail(ctx, lbcert, fmt.Sprintf("failed to find region for lbcert %s", lbcert.Name))
		return
	}
	self.SetStage("OnLoadbalancerCertificateDeleteComplete", nil)
	if err := region.GetDriver().RequestDeleteLoadbalancerCertificate(ctx, self.GetUserCred(), lbcert, self); err != nil {
		self.taskFail(ctx, lbcert, err.Error())
	}
}

func (self *LoadbalancerCertificateDeleteTask) OnLoadbalancerCertificateDeleteComplete(ctx context.Context, lbcert *models.SLoadbalancerCertificate, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(lbcert, db.ACT_DELETE, lbcert.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lbcert, logclient.ACT_DELETE, nil, self.UserCred, true)
	lbcert.DoPendingDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerCertificateDeleteTask) OnLoadbalancerCertificateDeleteCompleteFailed(ctx context.Context, lbcert *models.SLoadbalancerCertificate, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbcert, reason.String())
}
