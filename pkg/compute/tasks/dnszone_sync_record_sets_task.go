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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DnsZoneSyncRecordSetsTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DnsZoneSyncRecordSetsTask{})
}

func (self *DnsZoneSyncRecordSetsTask) taskComplete(ctx context.Context, dnsZone *models.SDnsZone) {
	dnsZone.SetStatus(self.GetUserCred(), api.DNS_ZONE_STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}

func (self *DnsZoneSyncRecordSetsTask) taskFailed(ctx context.Context, dnsZone *models.SDnsZone, err error) {
	dnsZone.SetStatus(self.GetUserCred(), api.DNS_ZONE_STATUS_SYNC_RECORD_SETS_FAILED, err.Error())
	db.OpsLog.LogEvent(dnsZone, db.ACT_SYNC_RECORD_SETS, dnsZone.GetShortDesc(ctx), self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, dnsZone, logclient.ACT_SYNC_RECORD_SETS, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DnsZoneSyncRecordSetsTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	dnsZone := obj.(*models.SDnsZone)

	caches, err := dnsZone.GetDnsZoneCaches()
	if err != nil {
		self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "GetDnsZoneCaches"))
	}

	for i := range caches {
		if len(caches[i].ExternalId) > 0 {
			err = caches[i].SyncRecordSets(ctx, self.GetUserCred())
			if err != nil {
				self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "SyncRecordSets for cache %s", caches[i].Id))
				return
			}
		}
	}

	self.taskComplete(ctx, dnsZone)
}
