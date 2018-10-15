package tasks

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/utils"
)

type CloudAccountSyncInfoTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudAccountSyncInfoTask{})
}

func (self *CloudAccountSyncInfoTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	account := obj.(*models.SCloudaccount)
	account.MarkStartSync(self.UserCred)
	if _, err := account.GetSubAccounts(); err != nil {
		account.SetStatus(self.UserCred, models.CLOUD_PROVIDER_DISCONNECTED, err.Error())
		self.SetStageFailed(ctx, err.Error())
		return
	}

	syncRange := models.SSyncRange{}
	syncRangeJson, _ := self.Params.Get("sync_range")
	if syncRangeJson != nil {
		syncRangeJson.Unmarshal(&syncRange)
	}
	// do sync
	exsitSubTask := false
	self.SetStage("on_cloudaccount_sync_complete", nil)
	for _, cloudprovider := range account.GetCloudproviders() {
		if cloudprovider.Enabled {
			exsitSubTask = true
			cloudprovider.StartSyncCloudProviderInfoTask(ctx, self.UserCred, &syncRange, self.GetTaskId())
		}
	}
	if !exsitSubTask {
		account.SetStatus(self.UserCred, models.CLOUD_PROVIDER_CONNECTED, "")
		self.SetStageComplete(ctx, nil)
	}
}

func (self *CloudAccountSyncInfoTask) OnCloudaccountSyncComplete(ctx context.Context, account *models.SCloudaccount, data jsonutils.JSONObject) {
	for _, cloudprovider := range account.GetCloudproviders() {
		if cloudprovider.Enabled &&
			utils.IsInStringArray(account.Status, []string{models.CLOUD_PROVIDER_START_SYNC, models.CLOUD_PROVIDER_SYNCING}) && time.Now().Sub(cloudprovider.LastSync) < time.Minute*20 {
			return
		}
	}
	account.SetStatus(self.UserCred, models.CLOUD_PROVIDER_CONNECTED, "")
	self.SetStageComplete(ctx, nil)
}
