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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudid/models"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SyncCloudIdResourcesTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SyncCloudIdResourcesTask{})
}

func (self *SyncCloudIdResourcesTask) taskFailed(ctx context.Context, cloudaccount *models.SCloudaccount, err error) {
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *SyncCloudIdResourcesTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	account := obj.(*models.SCloudaccount)

	provider, err := account.GetProvider()
	if err != nil {
		self.taskFailed(ctx, account, errors.Wrap(err, "GetProvider"))
		return
	}

	err = account.SyncCustomCloudpoliciesFromCloud(ctx, self.GetUserCred())
	if err != nil {
		self.taskFailed(ctx, account, errors.Wrapf(err, "SyncCustomCloudpoliciesFromCloud"))
		return
	}

	groups, err := provider.GetICloudgroups()
	if err != nil && (errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented) {
		self.taskFailed(ctx, account, errors.Wrapf(err, "GetICloudgroups"))
		return
	}

	err = account.SyncCloudgroupcaches(ctx, self.GetUserCred(), groups)
	if err != nil {
		self.taskFailed(ctx, account, errors.Wrapf(err, "SyncCloudgroupcaches"))
		return
	}

	self.SetStage("OnSyncCloudusersComplete", nil)
	account.StartSyncCloudusersTask(ctx, self.GetUserCred(), self.GetTaskId())
	self.SetStageComplete(ctx, nil)
}

func (self *SyncCloudIdResourcesTask) OnSyncCloudusersComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *SyncCloudIdResourcesTask) OnClouduserSyncCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
