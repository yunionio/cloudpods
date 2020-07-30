// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
			self.SetStageFailed(ctx, jsonutils.Marshal(err))
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
	self.SetStageFailed(ctx, body)

	logclient.AddActionLogWithStartable(self, account, logclient.ACT_DELETE, body, self.UserCred, false)
}
