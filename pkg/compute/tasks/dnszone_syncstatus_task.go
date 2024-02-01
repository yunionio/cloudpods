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

type DnsZoneSyncstatusTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DnsZoneSyncstatusTask{})
}

func (self *DnsZoneSyncstatusTask) taskFailed(ctx context.Context, zone *models.SDnsZone, err error) {
	zone.SetStatus(ctx, self.GetUserCred(), apis.STATUS_UNKNOWN, err.Error())
	self.SetStageComplete(ctx, nil)
}

func (self *DnsZoneSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	zone := obj.(*models.SDnsZone)

	if len(zone.ManagerId) == 0 {
		self.taskComplete(ctx, zone)
		return
	}

	iZone, err := zone.GetICloudDnsZone(ctx)
	if err != nil {
		self.taskFailed(ctx, zone, errors.Wrapf(err, "GetICloudDnsZone"))
		return
	}

	result := zone.SyncRecords(ctx, self.UserCred, iZone, false)
	if result.IsError() {
		logclient.AddActionLogWithContext(ctx, zone, logclient.ACT_CLOUD_SYNC, errors.Wrapf(err, "SyncRecords"), self.UserCred, false)
	}

	zone.SetStatus(ctx, self.UserCred, iZone.GetStatus(), "")
	self.SetStageComplete(ctx, nil)
}

func (self *DnsZoneSyncstatusTask) taskComplete(ctx context.Context, dnsZone *models.SDnsZone) {
	dnsZone.SetStatus(ctx, self.GetUserCred(), api.DNS_ZONE_STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}
