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

package cdn

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type CDNDomainSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CDNDomainSyncstatusTask{})
}

func (self *CDNDomainSyncstatusTask) taskFailed(ctx context.Context, domain *models.SCDNDomain, err error) {
	domain.SetStatus(ctx, self.GetUserCred(), api.CDN_DOMAIN_STATUS_UNKNOWN, err.Error())
	db.OpsLog.LogEvent(domain, db.ACT_SYNC_STATUS, domain.GetShortDesc(ctx), self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, domain, logclient.ACT_SYNC_STATUS, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *CDNDomainSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	domain := obj.(*models.SCDNDomain)

	iDomain, err := domain.GetICloudCDNDomain(ctx)
	if err != nil {
		self.taskFailed(ctx, domain, errors.Wrapf(err, "GetICloudCDNDomain"))
		return
	}

	err = domain.SyncWithCloudCDNDomain(ctx, self.GetUserCred(), iDomain)
	if err != nil {
		self.taskFailed(ctx, domain, errors.Wrapf(err, "SyncWithCloudCDNDomain"))
		return
	}

	self.SetStageComplete(ctx, nil)
}
