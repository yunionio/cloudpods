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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudid/models"
)

type CloudproviderSyncResourcesTask struct {
	taskman.STask
}

var (
	CloudproviderSyncWorkerManager *appsrv.SWorkerManager
)

func init() {
	CloudproviderSyncWorkerManager = appsrv.NewWorkerManager("CloudproviderSyncWorkerManager", 30, 1024, false)
	taskman.RegisterTaskAndWorker(CloudproviderSyncResourcesTask{}, CloudproviderSyncWorkerManager)
}

func (self *CloudproviderSyncResourcesTask) taskFailed(ctx context.Context, cp *models.SCloudprovider, err error) {
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *CloudproviderSyncResourcesTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	cp := obj.(*models.SCloudprovider)

	provider, err := cp.GetProvider()
	if err != nil {
		self.taskFailed(ctx, cp, errors.Wrap(err, "GetProvider"))
		return
	}

	driver, err := cp.GetDriver()
	if err != nil {
		self.taskFailed(ctx, cp, errors.Wrap(err, "GetDriver"))
		return
	}

	err = driver.RequestSyncCloudproviderResources(ctx, self.GetUserCred(), cp, provider)
	if err != nil {
		if errors.Cause(err) != cloudprovider.ErrNotImplemented {
			self.taskFailed(ctx, cp, errors.Wrap(err, "RequestSyncCloudproviderResources"))
			return
		}
	}

	self.SetStageComplete(ctx, nil)
}
