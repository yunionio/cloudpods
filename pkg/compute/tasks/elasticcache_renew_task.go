package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ElasticcacheRenewTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ElasticcacheRenewTask{})
}

func (self *ElasticcacheRenewTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	instance := obj.(*models.SElasticcache)

	durationStr, _ := self.GetParams().GetString("duration")
	bc, _ := billing.ParseBillingCycle(durationStr)

	region, _ := instance.GetRegion()
	exp, err := region.GetDriver().RequestRenewElasticcache(ctx, self.UserCred, instance, bc)
	if err != nil {
		db.OpsLog.LogEvent(instance, db.ACT_REW_FAIL, err, self.UserCred)
		logclient.AddActionLogWithStartable(self, instance, logclient.ACT_RENEW, err, self.UserCred, false)
		instance.SetStatus(self.GetUserCred(), api.ELASTIC_CACHE_RENEW_FAILED, err.Error())
		self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}

	err = instance.SaveRenewInfo(ctx, self.UserCred, &bc, &exp, "")
	if err != nil {
		msg := fmt.Sprintf("SaveRenewInfo fail %s", err)
		log.Errorf(msg)
		self.SetStageFailed(ctx, jsonutils.NewString(msg))
		return
	}

	logclient.AddActionLogWithStartable(self, instance, logclient.ACT_RENEW, nil, self.UserCred, true)
	self.SetStage("OnElasticcacheSyncstatus", nil)
	models.StartResourceSyncStatusTask(ctx, self.UserCred, instance, "ElasticcacheSyncstatusTask", self.GetTaskId())
}

func (self *ElasticcacheRenewTask) OnElasticcacheSyncstatusComplete(ctx context.Context, ec *models.SElasticcache, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *ElasticcacheRenewTask) OnElasticcacheSyncstatusCompleteFailed(ctx context.Context, ec *models.SElasticcache, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
