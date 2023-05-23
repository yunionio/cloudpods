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

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudid/models"
)

type CloudpolicyCacheTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudpolicyCacheTask{})
}

func (self *CloudpolicyCacheTask) taskFailed(ctx context.Context, policy *models.SCloudpolicycache, err error) {
	policy.SetStatus(self.GetUserCred(), apis.STATUS_CREATE_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *CloudpolicyCacheTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	cache := obj.(*models.SCloudpolicycache)

	err := cache.CacheCustomCloudpolicy()
	if err != nil {
		self.taskFailed(ctx, cache, errors.Wrapf(err, "CacheCustomCloudpolicy"))
		return
	}

	self.SetStageComplete(ctx, nil)
}
