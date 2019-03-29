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
