package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type CloudAccountImportTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudAccountImportTask{})
}

func (self *CloudAccountImportTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	cloudAccount := obj.(*models.SCloudaccount)

	cloudAccount.MarkStartSync(self.UserCred)

	autoCreateProject := jsonutils.QueryBoolean(self.Params, "auto_create_project", false)
	autoSync := jsonutils.QueryBoolean(self.Params, "auto_sync", false)

	newProviders := make([]models.SCloudprovider, 0)
	subAccounts, err := cloudAccount.GetSubAccounts()
	if err != nil {
		cloudAccount.SetStatus(self.UserCred, models.CLOUD_PROVIDER_DISCONNECTED, "")
		self.SetStageFailed(ctx, err.Error())
		return
	}
	for i := 0; i < len(subAccounts); i += 1 {
		provider, isNew, err := cloudAccount.ImportSubAccount(ctx, self.UserCred, subAccounts[i], autoCreateProject)
		if err != nil {
			cloudAccount.SetStatus(self.UserCred, models.CLOUD_PROVIDER_DISCONNECTED, "")
			self.SetStageFailed(ctx, err.Error())
			return
		}
		if isNew {
			newProviders = append(newProviders, *provider)
		}
	}

	if autoSync {
		self.SetStage("SyncNewCloudProviderComplete", nil)
		cloudAccount.StartSyncCloudProviderInfoTask(ctx, self.UserCred, newProviders, nil, self.GetTaskId())
	} else {
		self.complete(ctx, cloudAccount)
	}
}

func (self *CloudAccountImportTask) SyncNewCloudProviderComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	cloudAccount := obj.(*models.SCloudaccount)
	self.complete(ctx, cloudAccount)
}

func (self *CloudAccountImportTask) complete(ctx context.Context, cloudAccount *models.SCloudaccount) {
	cloudAccount.SetStatus(self.UserCred, models.CLOUD_PROVIDER_CONNECTED, "import complete")
	self.SetStageComplete(ctx, nil)
}
