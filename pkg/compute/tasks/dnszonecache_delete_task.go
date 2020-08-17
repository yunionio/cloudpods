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

type DnsZoneCacheDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DnsZoneCacheDeleteTask{})
}

func (self *DnsZoneCacheDeleteTask) taskFailed(ctx context.Context, cache *models.SDnsZoneCache, err error) {
	cache.SetStatus(self.GetUserCred(), api.DNS_ZONE_CACHE_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(cache, db.ACT_DELETE, cache.GetShortDesc(ctx), self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, cache, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DnsZoneCacheDeleteTask) taskComplete(ctx context.Context, cache *models.SDnsZoneCache) {
	dnsZone, err := cache.GetDnsZone()
	if err == nil {
		dnsZone.SetStatus(self.GetUserCred(), api.DNS_ZONE_STATUS_AVAILABLE, "")
	}
	cache.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}

func (self *DnsZoneCacheDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cache := obj.(*models.SDnsZoneCache)

	if len(cache.ExternalId) == 0 {
		self.taskComplete(ctx, cache)
		return
	}

	iDnsZone, err := cache.GetICloudDnsZone()
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, cache)
			return
		}
		self.taskFailed(ctx, cache, errors.Wrapf(err, "GetICloudDnsZone"))
		return
	}

	err = iDnsZone.Delete()
	if err != nil {
		self.taskFailed(ctx, cache, errors.Wrapf(err, "iDnsZone.Delete"))
		return
	}

	self.taskComplete(ctx, cache)
}
