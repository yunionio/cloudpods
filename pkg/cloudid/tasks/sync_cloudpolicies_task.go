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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudid/models"
)

type SyncCloudpoliciesTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SyncCloudpoliciesTask{})
}

func (self *SyncCloudpoliciesTask) taskFailed(ctx context.Context, cloudaccount *models.SCloudaccount, err error) {
	self.SetStageFailed(ctx, err.Error())
}

func (self *SyncCloudpoliciesTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	account := obj.(*models.SCloudaccount)

	provider, err := account.GetProvider()
	if err != nil {
		self.taskFailed(ctx, account, errors.Wrap(err, "GetProvider"))
		return
	}

	policy, err := provider.GetISystemCloudpolicies()
	if err != nil {
		self.taskFailed(ctx, account, errors.Wrapf(err, "GetISystemCloudpolicies for %s(%s)", account.Name, account.Provider))
		return
	}

	result := account.SyncCloudpolicies(ctx, self.GetUserCred(), policy)
	log.Infof("Sync policies for %s(%s) result: %s", account.Name, account.Provider, result.Result())

	self.SetStageComplete(ctx, nil)
}
