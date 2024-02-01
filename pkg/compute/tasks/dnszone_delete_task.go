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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DnsZoneDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DnsZoneDeleteTask{})
}

func (self *DnsZoneDeleteTask) taskFailed(ctx context.Context, dnsZone *models.SDnsZone, err error) {
	dnsZone.SetStatus(ctx, self.GetUserCred(), api.DNS_ZONE_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(dnsZone, db.ACT_DELETE, dnsZone.GetShortDesc(ctx), self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, dnsZone, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DnsZoneDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	zone := obj.(*models.SDnsZone)

	iZone, err := zone.GetICloudDnsZone(ctx)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, zone)
			return
		}
		self.taskFailed(ctx, zone, errors.Wrapf(err, "GetICloudDnsZone"))
		return
	}

	err = iZone.Delete()
	if err != nil {
		self.taskFailed(ctx, zone, errors.Wrapf(err, "Delete"))
		return
	}

	self.taskComplete(ctx, zone)
}

func (self *DnsZoneDeleteTask) taskComplete(ctx context.Context, zone *models.SDnsZone) {
	zone.RealDelete(ctx, self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, zone, logclient.ACT_DELETE, nil, self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    zone,
		Action: notifyclient.ActionDelete,
	})
	self.SetStageComplete(ctx, nil)
}
