package tasks

import (
	"context"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type CloudAccountDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudAccountDeleteTask{})
}

func (self *CloudAccountDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	account := obj.(*models.SCloudaccount)

	account.SetStatus(self.UserCred, api.CLOUD_PROVIDER_DELETING, "deleting")

	providers := account.GetCloudproviders()

	self.SetStage("OnAllCloudProviderDeleteComplete", nil)

	for i := range providers {
		err := providers[i].StartCloudproviderDeleteTask(ctx, self.UserCred, self.GetTaskId())
		if err != nil {
			account.SetStatus(self.UserCred, api.CLOUD_PROVIDER_DELETE_FAILED, err.Error())
			self.SetStageFailed(ctx, err.Error())
			return
		}
	}
}

func (self *CloudAccountDeleteTask) OnAllCloudProviderDeleteComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	account := obj.(*models.SCloudaccount)

	account.RealDelete(ctx, self.UserCred)

	self.SetStageComplete(ctx, nil)
}

func (self *CloudAccountDeleteTask) OnAllCloudProviderDeleteCompleteFailed(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	account := obj.(*models.SCloudaccount)

	account.SetStatus(self.UserCred, api.CLOUD_PROVIDER_DELETE_FAILED, body.String())
	self.SetStageFailed(ctx, body.String())
}
