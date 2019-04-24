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

	if len(providers) == 0 {
		self.OnAllCloudProviderDeleteComplete(ctx, account, nil)
		return
	}

	self.SetStage("OnAllCloudProviderDeleteComplete", nil)

	for i := range providers {
		err := providers[i].StartCloudproviderDeleteTask(ctx, self.UserCred, self.GetTaskId())
		if err != nil {
			// very unlikely
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

	logclient.AddActionLogWithStartable(self, account, logclient.ACT_DELETE, nil, self.UserCred, true)
}

func (self *CloudAccountDeleteTask) OnAllCloudProviderDeleteCompleteFailed(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	account := obj.(*models.SCloudaccount)

	account.SetStatus(self.UserCred, api.CLOUD_PROVIDER_DELETE_FAILED, body.String())
	self.SetStageFailed(ctx, body.String())

	logclient.AddActionLogWithStartable(self, account, logclient.ACT_DELETE, body, self.UserCred, false)
}
