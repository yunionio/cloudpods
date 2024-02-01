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
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SecurityGroupSyncTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SecurityGroupSyncTask{})
}

func (self *SecurityGroupSyncTask) taskFailed(ctx context.Context, secgroup *models.SSecurityGroup, err error) {
	secgroup.SetStatus(ctx, self.UserCred, apis.STATUS_UNKNOWN, "")
	logclient.AddActionLogWithContext(ctx, secgroup, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *SecurityGroupSyncTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	secgroup := obj.(*models.SSecurityGroup)

	iGroup, err := secgroup.GetISecurityGroup(ctx)
	if err != nil {
		self.taskFailed(ctx, secgroup, errors.Wrapf(err, "GetISecurityGroup"))
		return
	}

	rules, err := iGroup.GetRules()
	if err != nil {
		self.taskFailed(ctx, secgroup, errors.Wrapf(err, "GetRules"))
		return
	}

	provider, err := secgroup.GetCloudprovider()
	if err != nil {
		self.taskFailed(ctx, secgroup, errors.Wrapf(err, "GetCloudprovider"))
		return
	}

	err = secgroup.SyncWithCloudSecurityGroup(ctx, self.GetUserCred(), iGroup, provider.GetOwnerId(), false)
	if err != nil {
		self.taskFailed(ctx, secgroup, errors.Wrapf(err, "GetRules"))
		return
	}

	result := secgroup.SyncRules(ctx, self.GetUserCred(), rules)
	if result.IsError() {
		self.taskFailed(ctx, secgroup, errors.Wrapf(result.AllError(), "SyncRules"))
		return
	}

	secgroup.SetStatus(ctx, self.GetUserCred(), api.SECGROUP_STATUS_READY, "")
	self.SetStageComplete(ctx, nil)
}
