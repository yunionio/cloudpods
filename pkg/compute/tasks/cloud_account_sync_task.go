package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/skus"
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
		account = cloudproviders[0].GetCloudaccount()
		if account == nil {
			self.SetStageFailed(ctx, "cloudprovide fail to get valid cloudaccount")
			return
		}
		account.MarkStartSync(self.UserCred)
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
	for i := 0; i < len(cloudproviders); i += 1 {
		self.SyncCloudprovider(ctx, cloudproviders[i], syncRange)
	}
}

func (self *CloudAccountSyncInfoTask) SyncCloudprovider(ctx context.Context, cloudprovider *models.SCloudprovider, syncRange *models.SSyncRange) {
	// lockman.LockObject(ctx, cloudprovider)
	// defer lockman.ReleaseObject(ctx, cloudprovider)

	cloudprovider.StartSyncCloudProviderInfoTask(ctx, self.UserCred, syncRange, self.GetId())
}

func (self *CloudAccountSyncInfoTask) OnCloudaccountSyncComplete(ctx context.Context, items []db.IStandaloneModel, data jsonutils.JSONObject) {
	if len(items) > 0 {
		cloudprovider := items[0].(*models.SCloudprovider)
		account := cloudprovider.GetCloudaccount()
		if account != nil {
			account.SetStatus(self.UserCred, models.CLOUD_PROVIDER_CONNECTED, "")
		}

		// sync skus
		if err := skus.SyncSkusByProviderIds([]string{cloudprovider.Provider}); err != nil {
			self.SetStageFailed(ctx, err.Error())
		}
	}
	self.SetStageComplete(ctx, nil)
}
