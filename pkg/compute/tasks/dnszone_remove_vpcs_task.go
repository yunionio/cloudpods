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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DnsZoneRemoveVpcsTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DnsZoneRemoveVpcsTask{})
}

func (self *DnsZoneRemoveVpcsTask) taskFailed(ctx context.Context, zone *models.SDnsZone, err error) {
	zone.SetStatus(ctx, self.GetUserCred(), apis.STATUS_UNKNOWN, err.Error())
	logclient.AddActionLogWithContext(ctx, zone, logclient.ACT_REMOVE_VPCS, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DnsZoneRemoveVpcsTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
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

	iVpcIds, err := iZone.GetICloudVpcIds()
	if err != nil {
		self.taskFailed(ctx, zone, errors.Wrapf(err, "GetICloudVpcIds"))
		return
	}

	vpcIds := []string{}
	self.GetParams().Unmarshal(&vpcIds, "vpc_ids")

	for i := range vpcIds {
		vpcObj, err := models.VpcManager.FetchById(vpcIds[i])
		if err != nil {
			self.taskFailed(ctx, zone, errors.Wrapf(err, "Vpc.FetchById(%s)", vpcIds[i]))
			return
		}
		vpc := vpcObj.(*models.SVpc)
		region, err := vpc.GetRegion()
		if err != nil {
			self.taskFailed(ctx, zone, errors.Wrapf(err, "GetRegion(%s)", vpc.CloudregionId))
			return
		}
		opts := &cloudprovider.SPrivateZoneVpc{
			Id:       vpc.ExternalId,
			RegionId: region.GetRegionExtId(),
		}

		if utils.IsInStringArray(opts.Id, iVpcIds) {
			err = iZone.RemoveVpc(opts)
			if err != nil {
				self.taskFailed(ctx, zone, errors.Wrapf(err, "RemoveVpc(%s)", vpc.Name))
				return
			}
		}

		zone.RemoveVpc(ctx, vpcIds[i])
	}

	self.taskComplete(ctx, zone)
}

func (self *DnsZoneRemoveVpcsTask) taskComplete(ctx context.Context, zone *models.SDnsZone) {
	zone.SetStatus(ctx, self.GetUserCred(), api.DNS_ZONE_STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}
