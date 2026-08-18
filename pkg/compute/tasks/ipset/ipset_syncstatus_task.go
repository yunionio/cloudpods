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

package ipset

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type IpSetSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(IpSetSyncstatusTask{})
}

func (self *IpSetSyncstatusTask) taskFailed(ctx context.Context, ipset *models.SIpSet, err error) {
	ipset.SetStatus(ctx, self.UserCred, apis.STATUS_UNKNOWN, err.Error())
	logclient.AddActionLogWithStartable(self, ipset, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *IpSetSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	ipset := obj.(*models.SIpSet)
	iIpSet, err := ipset.GetICloudIpSet(ctx)
	if err != nil {
		self.taskFailed(ctx, ipset, errors.Wrapf(err, "GetICloudIpSet"))
		return
	}
	provider := ipset.GetCloudprovider()
	if provider == nil {
		self.taskFailed(ctx, ipset, errors.Errorf("empty cloudprovider"))
		return
	}
	err = ipset.SyncWithCloudIpSet(ctx, self.UserCred, iIpSet, provider, ipset.CloudregionId)
	if err != nil {
		self.taskFailed(ctx, ipset, errors.Wrapf(err, "SyncWithCloudIpSet"))
		return
	}
	self.SetStageComplete(ctx, nil)
}
