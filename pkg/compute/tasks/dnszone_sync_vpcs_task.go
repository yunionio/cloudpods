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
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DnsZoneSyncVpcsTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DnsZoneSyncVpcsTask{})
}

func (self *DnsZoneSyncVpcsTask) taskFailed(ctx context.Context, dnsZone *models.SDnsZone, err error) {
	dnsZone.SetStatus(self.GetUserCred(), api.DNS_ZONE_STATUS_SYNC_VPCS_FAILED, err.Error())
	db.OpsLog.LogEvent(dnsZone, db.ACT_SYNC_VPCS, dnsZone.GetShortDesc(ctx), self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, dnsZone, logclient.ACT_SYNC_VPCS, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DnsZoneSyncVpcsTask) taskComplete(ctx context.Context, dnsZone *models.SDnsZone) {
	db.OpsLog.LogEvent(dnsZone, db.ACT_SYNC_VPCS, dnsZone.GetShortDesc(ctx), self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, dnsZone, logclient.ACT_SYNC_VPCS, "", self.UserCred, true)
	dnsZone.SetStatus(self.GetUserCred(), api.DNS_ZONE_STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}

func (self *DnsZoneSyncVpcsTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	dnsZone := obj.(*models.SDnsZone)

	accounts, err := dnsZone.GetCloudaccounts()
	if err != nil {
		self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "GetCloudaccounts"))
		return
	}
	accountIds := []string{}
	for i := range accounts {
		cache, err := dnsZone.RegisterCache(ctx, self.GetUserCred(), accounts[i].Id)
		if err != nil {
			self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "dnsZone.RegisterCache"))
			return
		}
		provider, err := cache.GetProvider(ctx)
		if err != nil {
			self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "GetProvider"))
			return
		}

		if len(cache.ExternalId) == 0 {
			opts := cloudprovider.SDnsZoneCreateOptions{
				Name:     dnsZone.Name,
				Desc:     dnsZone.Description,
				ZoneType: cloudprovider.TDnsZoneType(dnsZone.ZoneType),
				Options:  dnsZone.Options,
			}
			vpcs, err := cache.GetVpcs()
			if err != nil {
				self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "GetVpcs"))
				return
			}
			for _, vpc := range vpcs {
				iVpc, err := vpc.GetIVpc(ctx)
				if err != nil {
					self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "GetIVpc for vpc %s", vpc.Name))
					return
				}
				opts.Vpcs = append(opts.Vpcs, cloudprovider.SPrivateZoneVpc{
					Id:       iVpc.GetGlobalId(),
					RegionId: iVpc.GetRegion().GetId(),
				})
			}
			iDnsZone, err := provider.CreateICloudDnsZone(&opts)
			if err != nil {
				self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "CreateICloudDnsZone"))
				return
			}
			err = cache.SyncWithCloudDnsZone(ctx, self.GetUserCred(), iDnsZone)
			if err != nil {
				self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "SyncWithCloudDnsZone"))
				return
			}
		} else {
			err := cache.SyncVpcForCloud(ctx, self.GetUserCred())
			if err != nil {
				self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "SyncVpcForCloud"))
				return
			}
		}
		accountIds = append(accountIds, accounts[i].Id)
	}

	caches, err := dnsZone.GetDnsZoneCaches()
	if err != nil {
		self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "GetDnsZoneCaches"))
		return
	}

	for i := range caches {
		lockman.LockObject(ctx, &caches[i])
		defer lockman.ReleaseObject(ctx, &caches[i])

		if !utils.IsInStringArray(caches[i].CloudaccountId, accountIds) {
			if len(caches[i].ExternalId) > 0 {
				iDnsZone, err := caches[i].GetICloudDnsZone(ctx)
				if err != nil {
					if errors.Cause(err) != cloudprovider.ErrNotFound {
						self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "GetICloudDnsZone"))
						return
					}
				} else {
					err = iDnsZone.Delete()
					if err != nil {
						self.taskFailed(ctx, dnsZone, errors.Wrapf(err, "iDnsZone.Delete"))
						return
					}
				}
				caches[i].RealDelete(ctx, self.GetUserCred())
			}
		}
	}

	self.SetStage("OnSyncRecordSetComplete", nil)
	dnsZone.StartDnsZoneSyncRecordSetsTask(ctx, self.GetUserCred(), self.GetTaskId())
}

func (self *DnsZoneSyncVpcsTask) OnSyncRecordSetComplete(ctx context.Context, dnsZone *models.SDnsZone, data jsonutils.JSONObject) {
	self.taskComplete(ctx, dnsZone)
}

func (self *DnsZoneSyncVpcsTask) OnSyncRecordSetCompleteFailed(ctx context.Context, cache *models.SDnsZoneCache, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
