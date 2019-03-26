package tasks

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type CloudProviderDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudProviderDeleteTask{})
}

func (self *CloudProviderDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	provider := obj.(*models.SCloudprovider)

	provider.SetStatus(self.UserCred, models.CLOUD_PROVIDER_DELETING, "StartDiskCloudproviderTask")

	err := provider.RealDelete(ctx, self.UserCred)
	if err != nil {
		provider.SetStatus(self.UserCred, models.CLOUD_PROVIDER_DELETE_FAILED, "StartDiskCloudproviderTask")
		self.SetStageFailed(ctx, err.Error())
		return
	}

	provider.SetStatus(self.UserCred, models.CLOUD_PROVIDER_DELETED, "StartDiskCloudproviderTask")

	self.SetStageComplete(ctx, nil)
}
