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

type SyncCloudIdResourcesTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SyncCloudIdResourcesTask{})
}

func (self *SyncCloudIdResourcesTask) taskFailed(ctx context.Context, cloudaccount *models.SCloudaccount, err jsonutils.JSONObject) {
	self.SetStageFailed(ctx, err)
}

func (self *SyncCloudIdResourcesTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	account := obj.(*models.SCloudaccount)

	provider, err := account.GetProvider()
	if err != nil {
		self.taskFailed(ctx, account, jsonutils.NewString(errors.Wrap(err, "GetProvider").Error()))
		return
	}

	policy, err := provider.GetISystemCloudpolicies()
	if err != nil {
		log.Errorf("GetISystemCloudpolicies for %s(%s) failed: %v", account.Name, account.Provider, err)
	} else {
		result := account.SyncCloudpolicies(ctx, self.GetUserCred(), policy)
		log.Infof("Sync policies for %s(%s) result: %s", account.Name, account.Provider, result.Result())
	}

	groups, err := provider.GetICloudgroups()
	if err != nil {
		log.Errorf("GetICloudgroups for %s(%s) failed: %v", account.Name, account.Provider, err)
	} else {
		result := account.SyncCloudgroupcaches(ctx, self.GetUserCred(), groups)
		log.Infof("Sync groups for %s(%s) result: %s", account.Name, account.Provider, result.Result())
	}

	self.SetStage("OnSyncCloudusersComplete", nil)
	account.StartSyncCloudusersTask(ctx, self.GetUserCred(), self.GetParentId())
	self.SetStageComplete(ctx, nil)
}

func (self *SyncCloudIdResourcesTask) OnSyncCloudusersComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *SyncCloudIdResourcesTask) OnClouduserSyncCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
