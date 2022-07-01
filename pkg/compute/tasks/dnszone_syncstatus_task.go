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

type DnsZoneSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DnsZoneSyncstatusTask{})
}

func (self *DnsZoneSyncstatusTask) taskComplete(ctx context.Context, dnsZone *models.SDnsZone) {
	dnsZone.SetStatus(self.GetUserCred(), api.DNS_ZONE_STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}

func (self *DnsZoneSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	dnsZone := obj.(*models.SDnsZone)

	caches, err := dnsZone.GetDnsZoneCaches()
	if err != nil {
		self.taskComplete(ctx, dnsZone)
		return
	}

	for i := range caches {
		if len(caches[i].ExternalId) > 0 {
			status := api.DNS_ZONE_CACHE_STATUS_UNKNOWN
			iZone, err := caches[i].GetICloudDnsZone(ctx)
			if err != nil {
				logclient.AddActionLogWithContext(ctx, &caches[i], logclient.ACT_SYNC_STATUS, errors.Wrapf(err, "GetICloudDnsZone"), self.UserCred, false)
			} else {
				status = iZone.GetStatus()
			}
			if caches[i].Status != status {
				caches[i].SetStatus(self.GetUserCred(), status, "")
			}
			if len(caches) == 1 {
				account, _ := caches[i].GetCloudaccount()
				if account != nil {
					dnsZone.SyncDnsRecordSets(ctx, self.GetUserCred(), account.Provider, iZone)
				}
			}
		}
	}

	self.taskComplete(ctx, dnsZone)
}
