package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type CloudAccountSyncInfoTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudAccountSyncInfoTask{})
}

func (self *CloudAccountSyncInfoTask) OnInit(ctx context.Context, objs []db.IStandaloneModel, body jsonutils.JSONObject) {
	cloudproviders := make([]*models.SCloudprovider, 0)
	for _, obj := range objs {
		cloudprovider := obj.(*models.SCloudprovider)
		if cloudprovider.Enabled {
			cloudproviders = append(cloudproviders, cloudprovider)
		}
	}

	var account *models.SCloudaccount
	if len(cloudproviders) > 0 {
		if account = models.CloudaccountManager.FetchCloudaccountById(cloudproviders[0].CloudaccountId); account == nil {
			account.SetStatus(self.UserCred, models.CLOUD_PROVIDER_CONNECTED, "")
			self.SetStageComplete(ctx, nil)
			return
		}
	} else {
		account.SetStatus(self.UserCred, models.CLOUD_PROVIDER_CONNECTED, "")
		self.SetStageComplete(ctx, nil)
		return
	}
	if _, err := account.GetSubAccounts(); err != nil {
		account.SetStatus(self.UserCred, models.CLOUD_PROVIDER_DISCONNECTED, "")
		self.SetStageFailed(ctx, err.Error())
		return
	}

	syncRange := models.SSyncRange{}
	syncRangeJson, _ := self.Params.Get("sync_range")
	if syncRangeJson != nil {
		syncRangeJson.Unmarshal(&syncRange)
	}

	if len(syncRange.Host) == 0 && !syncRange.FullSync {
		account.SetStatus(self.UserCred, models.CLOUD_PROVIDER_CONNECTED, "")
		self.SetStageComplete(ctx, nil)
		return
	}
	// do sync
	self.SetStage("on_cloudaccount_sync_complete", nil)
	self.SyncCloudaccount(ctx, account, cloudproviders, &syncRange)
}

func (self *CloudAccountSyncInfoTask) SyncCloudaccount(ctx context.Context, account *models.SCloudaccount, cloudproviders []*models.SCloudprovider, syncRange *models.SSyncRange) {
	for _, cloudprovider := range cloudproviders {
		self.SyncCloudprovider(ctx, cloudprovider, syncRange)
	}
}

func (self *CloudAccountSyncInfoTask) SyncCloudprovider(ctx context.Context, cloudprovider *models.SCloudprovider, syncRange *models.SSyncRange) {
	lockman.LockObject(ctx, cloudprovider)
	defer lockman.ReleaseObject(ctx, cloudprovider)

	cloudprovider.StartSyncCloudProviderInfoTask(ctx, self.UserCred, syncRange, self.GetId())
}

func (self *CloudAccountSyncInfoTask) OnCloudaccountSyncComplete(ctx context.Context, items []db.IStandaloneModel, data jsonutils.JSONObject) {
	if len(items) > 0 {
		cloudprovider := items[0].(*models.SCloudprovider)
		if account := models.CloudaccountManager.FetchCloudaccountById(cloudprovider.CloudaccountId); account != nil {
			account.SetStatus(self.UserCred, models.CLOUD_PROVIDER_CONNECTED, "")
		}
	}
	self.SetStageComplete(ctx, nil)
}
