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

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DnsRecordDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DnsRecordDeleteTask{})
}

func (self *DnsRecordDeleteTask) taskFailed(ctx context.Context, record *models.SDnsRecord, err error) {
	record.SetStatus(ctx, self.GetUserCred(), apis.STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(record, db.ACT_DELETE, record.GetShortDesc(ctx), self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, record, logclient.ACT_DELETE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DnsRecordDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	record := obj.(*models.SDnsRecord)

	zone, err := record.GetDnsZone()
	if err != nil {
		self.taskFailed(ctx, record, errors.Wrapf(err, "GetDnsZone"))
		return
	}

	if len(record.ExternalId) == 0 {
		self.taskComplete(ctx, zone, record)
		return
	}

	iZone, err := zone.GetICloudDnsZone(ctx)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, zone, record)
			return
		}
		self.taskFailed(ctx, record, errors.Wrapf(err, "GetICloudDnsZone"))
		return
	}

	iRec, err := iZone.GetIDnsRecordById(record.ExternalId)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, zone, record)
			return
		}
		self.taskFailed(ctx, record, errors.Wrapf(err, "GetIDnsRecordById"))
		return
	}

	err = iRec.Delete()
	if err != nil {
		self.taskFailed(ctx, record, errors.Wrapf(err, "Delete"))
		return
	}

	self.taskComplete(ctx, zone, record)
}

func (self *DnsRecordDeleteTask) taskComplete(ctx context.Context, zone *models.SDnsZone, record *models.SDnsRecord) {
	logclient.AddActionLogWithContext(ctx, zone, logclient.ACT_DELETE, record, self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    record,
		Action: notifyclient.ActionDelete,
	})
	record.RealDelete(ctx, self.UserCred)
	self.SetStageComplete(ctx, nil)
}
