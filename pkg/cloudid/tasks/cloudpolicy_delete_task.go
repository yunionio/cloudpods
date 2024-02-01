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

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudid/models"
)

type CloudpolicyDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudpolicyDeleteTask{})
}

func (self *CloudpolicyDeleteTask) taskFailed(ctx context.Context, policy *models.SCloudpolicy, err error) {
	policy.SetStatus(ctx, self.GetUserCred(), api.CLOUD_USER_STATUS_DELETE_FAILED, err.Error())
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *CloudpolicyDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	policy := obj.(*models.SCloudpolicy)

	caches, err := policy.GetCloudpolicycaches()
	if err != nil {
		self.taskFailed(ctx, policy, errors.Wrapf(err, "GetCloudpolicycaches"))
		return
	}
	for i := range caches {
		if len(caches[i].ExternalId) > 0 {
			provider, err := caches[i].GetProvider()
			if err != nil {
				self.taskFailed(ctx, policy, errors.Wrapf(err, "GetProvider for cache %s(%s)", caches[i].Name, caches[i].Id))
				return
			}
			policies, err := provider.GetICustomCloudpolicies()
			if err != nil {
				self.taskFailed(ctx, policy, errors.Wrapf(err, "GetICustomCloudpolicies for account %s provider %s", caches[i].CloudaccountId, caches[i].CloudproviderId))
				return
			}
			for i := range policies {
				if policies[i].GetGlobalId() == caches[i].ExternalId {
					err = policies[i].Delete()
					if err != nil {
						self.taskFailed(ctx, policy, errors.Wrapf(err, "Delete %s", policies[i].GetName()))
						return
					}
				}
			}
		}
	}

	policy.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}
