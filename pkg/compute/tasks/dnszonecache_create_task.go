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
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DnsZoneCacheCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DnsZoneCacheCreateTask{})
}

func (self *DnsZoneCacheCreateTask) taskFailed(ctx context.Context, cache *models.SDnsZoneCache, err error) {
	cache.SetStatus(self.GetUserCred(), api.DNS_ZONE_CACHE_STATUS_CREATE_FAILED, err.Error())
	db.OpsLog.LogEvent(cache, db.ACT_CREATE, cache.GetShortDesc(ctx), self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, cache, logclient.ACT_CREATE, err, self.UserCred, false)
	dnsZone, err := cache.GetDnsZone()
	if err == nil {
		dnsZone.SetStatus(self.GetUserCred(), api.DNS_ZONE_CACHE_STATUS_CREATE_FAILED, "")
		logclient.AddActionLogWithContext(ctx, dnsZone, logclient.ACT_CREATE, err, self.UserCred, false)
	}
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DnsZoneCacheCreateTask) taskComplete(ctx context.Context, cache *models.SDnsZoneCache) {
	dnsZone, err := cache.GetDnsZone()
	if err == nil {
		dnsZone.SetStatus(self.GetUserCred(), api.DNS_ZONE_STATUS_AVAILABLE, "")
	}
	cache.SetStatus(self.GetUserCred(), api.DNS_ZONE_CACHE_STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}

func (self *DnsZoneCacheCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cache := obj.(*models.SDnsZoneCache)

	dnsZone, err := cache.GetDnsZone()
	if err != nil {
		self.taskFailed(ctx, cache, errors.Wrapf(err, "GetDnsZone"))
		return
	}

	provider, err := cache.GetProvider()
	if err != nil {
		self.taskFailed(ctx, cache, errors.Wrapf(err, "GetProvider"))
		return
	}

	opts := cloudprovider.SDnsZoneCreateOptions{
		Name:     dnsZone.Name,
		Desc:     dnsZone.Description,
		ZoneType: cloudprovider.TDnsZoneType(dnsZone.ZoneType),
		Options:  dnsZone.Options,
	}

	iDnsZone, err := provider.CreateICloudDnsZone(&opts)
	if err != nil {
		self.taskFailed(ctx, cache, errors.Wrapf(err, "CreateICloudDnsZone"))
		return
	}

	err = cache.SyncWithCloudDnsZone(ctx, self.GetUserCred(), iDnsZone)
	if err != nil {
		self.taskFailed(ctx, cache, errors.Wrapf(err, "SyncWithCloudDnsZone"))
		return
	}

	self.SetStage("OnSyncRecordSetComplete", nil)
	dnsZone.StartDnsZoneSyncRecordSetsTask(ctx, self.GetUserCred(), self.GetTaskId())
}

func (self *DnsZoneCacheCreateTask) OnSyncRecordSetComplete(ctx context.Context, cache *models.SDnsZoneCache, data jsonutils.JSONObject) {
	self.taskComplete(ctx, cache)
}

func (self *DnsZoneCacheCreateTask) OnSyncRecordSetCompleteFailed(ctx context.Context, cache *models.SDnsZoneCache, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
