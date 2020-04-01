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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type PolicyDefinitionSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(PolicyDefinitionSyncstatusTask{})
}

func (self *PolicyDefinitionSyncstatusTask) taskFailed(ctx context.Context, definition *models.SPolicyDefinition, err error) {
	definition.SetStatus(self.UserCred, api.POLICY_DEFINITION_STATUS_UNKNOWN, err.Error())
	db.OpsLog.LogEvent(definition, db.ACT_SYNC_STATUS, err.Error(), self.GetUserCred())
	logclient.AddActionLogWithStartable(self, definition, logclient.ACT_SYNC_STATUS, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, err.Error())
}

func (self *PolicyDefinitionSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	definition := obj.(*models.SPolicyDefinition)
	cloudprovider := definition.GetCloudprovider()
	if cloudprovider == nil {
		self.taskFailed(ctx, definition, fmt.Errorf("failed to get cloudprovider for policy definition %s", definition.Name))
		return
	}
	provider, err := cloudprovider.GetProvider()
	if err != nil {
		self.taskFailed(ctx, definition, errors.Wrap(err, "GetProvider"))
		return
	}
	policyDefinitions, err := provider.GetICloudPolicyDefinitions()
	if err != nil {
		self.taskFailed(ctx, definition, errors.Wrap(err, "GetICloudPolicyDefinitions"))
		return
	}
	for i := range policyDefinitions {
		if policyDefinitions[i].GetGlobalId() == definition.ExternalId {
			err = definition.SyncWithCloudPolicyDefinition(ctx, self.GetUserCred(), cloudprovider, policyDefinitions[i])
			if err != nil {
				self.taskFailed(ctx, definition, errors.Wrap(err, "SyncWithCloudPolicyDefinition"))
				return
			}
			self.SetStageComplete(ctx, nil)
			return
		}
	}
	self.taskFailed(ctx, definition, fmt.Errorf("failed to found policy definition %s from cloud", definition.Name))
}
