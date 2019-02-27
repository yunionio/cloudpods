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

type LoadbalancerCertificateCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LoadbalancerCertificateCreateTask{})
}

func (self *LoadbalancerCertificateCreateTask) taskFail(ctx context.Context, lbcert *models.SLoadbalancerCertificate, reason string) {
	lbcert.SetStatus(self.GetUserCred(), consts.LB_CREATE_FAILED, reason)
	db.OpsLog.LogEvent(lbcert, db.ACT_ALLOCATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, lbcert, logclient.ACT_CREATE, reason, self.UserCred, false)
	notifyclient.NotifySystemError(lbcert.Id, lbcert.Name, consts.LB_CREATE_FAILED, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *LoadbalancerCertificateCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	lbcert := obj.(*models.SLoadbalancerCertificate)
	region := lbcert.GetRegion()
	if region == nil {
		self.taskFail(ctx, lbcert, fmt.Sprintf("failed to find region for lbcert %s", lbcert.Name))
		return
	}
	self.SetStage("OnLoadbalancerCertificateCreateComplete", nil)
	if err := region.GetDriver().RequestCreateLoadbalancerCertificate(ctx, self.GetUserCred(), lbcert, self); err != nil {
		self.taskFail(ctx, lbcert, err.Error())
	}
}

func (self *LoadbalancerCertificateCreateTask) OnLoadbalancerCertificateCreateComplete(ctx context.Context, lbcert *models.SLoadbalancerCertificate, data jsonutils.JSONObject) {
	lbcert.SetStatus(self.GetUserCred(), consts.LB_STATUS_ENABLED, "")
	db.OpsLog.LogEvent(lbcert, db.ACT_ALLOCATE, lbcert.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, lbcert, logclient.ACT_CREATE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *LoadbalancerCertificateCreateTask) OnLoadbalancerCertificateCreateCompleteFailed(ctx context.Context, lbcert *models.SLoadbalancerCertificate, reason jsonutils.JSONObject) {
	self.taskFail(ctx, lbcert, reason.String())
}
