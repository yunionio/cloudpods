package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type CloudAccountSyncInfoTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudAccountSyncInfoTask{})
}

func (self *CloudAccountSyncInfoTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	cloudaccount := obj.(*models.SCloudaccount)

	db.OpsLog.LogEvent(cloudaccount, db.ACT_SYNCING_HOST, "", self.UserCred)
	// cloudaccount.MarkSyncing(self.UserCred)

	// do sync
	err := cloudaccount.SyncCallSyncAccountTask(ctx, self.UserCred)

	if err != nil {
		cloudaccount.MarkEndSync(self.UserCred)
		db.OpsLog.LogEvent(cloudaccount, db.ACT_SYNC_HOST_FAILED, err, self.UserCred)
		self.SetStageFailed(ctx, err.Error())
		logclient.AddActionLogWithStartable(self, cloudaccount, logclient.ACT_CLOUD_SYNC, err, self.UserCred, false)
		return
	}

	syncRange := models.SSyncRange{}
	syncRangeJson, _ := self.Params.Get("sync_range")
	if syncRangeJson != nil {
		syncRangeJson.Unmarshal(&syncRange)
	} else {
		syncRange.FullSync = true
	}

	if !syncRange.NeedSyncInfo() {
		cloudaccount.MarkEndSync(self.UserCred)
		db.OpsLog.LogEvent(cloudaccount, db.ACT_SYNC_HOST_COMPLETE, "", self.UserCred)
		self.SetStageComplete(ctx, nil)
		logclient.AddActionLogWithStartable(self, cloudaccount, logclient.ACT_CLOUD_SYNC, "", self.UserCred, true)
		return
	}

	self.SetStage("on_cloudaccount_sync_complete", nil)

	cloudproviders := cloudaccount.GetEnabledCloudproviders()
	for i := range cloudproviders {
		cloudproviders[i].StartSyncCloudProviderInfoTask(ctx, self.UserCred, &syncRange, self.GetId())
	}
}

func (self *CloudAccountSyncInfoTask) OnCloudaccountSyncComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cloudaccount := obj.(*models.SCloudaccount)
	cloudaccount.MarkEndSync(self.UserCred)
	db.OpsLog.LogEvent(cloudaccount, db.ACT_SYNC_HOST_COMPLETE, "", self.UserCred)
	self.SetStageComplete(ctx, nil)
	logclient.AddActionLogWithStartable(self, cloudaccount, logclient.ACT_CLOUD_SYNC, "", self.UserCred, true)
}
