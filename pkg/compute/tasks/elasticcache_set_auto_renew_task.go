package tasks

import (
	"context"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ElasticcacheSetAutoRenewTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ElasticcacheSetAutoRenewTask{})
}

func (self *ElasticcacheSetAutoRenewTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	ec := obj.(*models.SElasticcache)

	autoRenew, _ := self.GetParams().Bool("auto_renew")
	region, _ := ec.GetRegion()
	err := region.GetDriver().RequestElasticcacheSetAutoRenew(ctx, self.UserCred, ec, autoRenew, self)
	if err != nil {
		db.OpsLog.LogEvent(ec, db.ACT_SET_AUTO_RENEW_FAIL, err, self.UserCred)
		logclient.AddActionLogWithStartable(self, ec, logclient.ACT_SET_AUTO_RENEW, err, self.UserCred, false)
		ec.SetStatus(self.GetUserCred(), api.ELASTIC_CACHE_SET_AUTO_RENEW_FAILED, err.Error())
		self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}

	logclient.AddActionLogWithStartable(self, ec, logclient.ACT_SET_AUTO_RENEW, nil, self.UserCred, true)
	self.SetStage("OnElasticcacheSyncstatus", nil)
	models.StartResourceSyncStatusTask(ctx, self.UserCred, ec, "ElasticcacheSyncstatusTask", self.GetTaskId())
}

func (self *ElasticcacheSetAutoRenewTask) OnElasticcacheSyncstatusComplete(ctx context.Context, ec *models.SElasticcache, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *ElasticcacheSetAutoRenewTask) OnElasticcacheSyncstatusCompleteFailed(ctx context.Context, ec *models.SElasticcache, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
