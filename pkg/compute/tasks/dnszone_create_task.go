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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DnsZoneCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DnsZoneCreateTask{})
}

func (self *DnsZoneCreateTask) taskFailed(ctx context.Context, zone *models.SDnsZone, err error) {
	zone.SetStatus(ctx, self.GetUserCred(), api.DNS_ZONE_STATUS_CREATE_FAILE, err.Error())
	db.OpsLog.LogEvent(zone, db.ACT_CREATE, zone.GetShortDesc(ctx), self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, zone, logclient.ACT_CREATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DnsZoneCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	zone := obj.(*models.SDnsZone)
	if len(zone.ManagerId) == 0 {
		self.taskComplete(ctx, zone)
		return
	}

	provider, err := zone.GetProvider(ctx)
	if err != nil {
		self.taskFailed(ctx, zone, errors.Wrapf(err, "GetProviderIdsQuery"))
		return
	}

	opts := &cloudprovider.SDnsZoneCreateOptions{
		Name:     zone.Name,
		Desc:     zone.Description,
		ZoneType: cloudprovider.TDnsZoneType(zone.ZoneType),
		Vpcs:     []cloudprovider.SPrivateZoneVpc{},
	}

	if zone.ZoneType == string(cloudprovider.PrivateZone) {
		vpcs, err := zone.GetVpcs()
		if err != nil {
			self.taskFailed(ctx, zone, errors.Wrapf(err, "GetVpcs"))
			return
		}
		for i := range vpcs {
			region, err := vpcs[i].GetRegion()
			if err != nil {
				self.taskFailed(ctx, zone, errors.Wrapf(err, "GetRegion"))
				return
			}
			opts.Vpcs = append(opts.Vpcs, cloudprovider.SPrivateZoneVpc{
				Id:       vpcs[i].ExternalId,
				RegionId: region.GetRegionExtId(),
			})
		}
	}

	iZone, err := provider.CreateICloudDnsZone(opts)
	if err != nil {
		self.taskFailed(ctx, zone, errors.Wrapf(err, "CreateICloudDnsZone"))
		return
	}
	_, err = db.Update(zone, func() error {
		zone.ExternalId = iZone.GetGlobalId()
		return nil
	})
	if err != nil {
		self.taskFailed(ctx, zone, errors.Wrapf(err, "set external id"))
		return
	}

	result := zone.SyncRecords(ctx, self.UserCred, iZone, false)
	log.Infof("sync records for zone %s result: %s", zone.Name, result.Result())

	self.taskComplete(ctx, zone)
}

func (self *DnsZoneCreateTask) taskComplete(ctx context.Context, zone *models.SDnsZone) {
	zone.SetStatus(ctx, self.GetUserCred(), api.DNS_ZONE_STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}
