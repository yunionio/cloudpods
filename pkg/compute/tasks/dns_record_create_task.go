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

type DnsRecordCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DnsRecordCreateTask{})
}

func (self *DnsRecordCreateTask) taskFailed(ctx context.Context, record *models.SDnsRecord, err error) {
	record.SetStatus(ctx, self.GetUserCred(), api.DNS_ZONE_STATUS_CREATE_FAILE, err.Error())
	db.OpsLog.LogEvent(record, db.ACT_CREATE, record.GetShortDesc(ctx), self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, record, logclient.ACT_CREATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DnsRecordCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	record := obj.(*models.SDnsRecord)
	zone, err := record.GetDnsZone()
	if err != nil {
		self.taskFailed(ctx, record, errors.Wrapf(err, "GetDnsZone"))
		return
	}

	if len(zone.ManagerId) == 0 {
		self.taskComplete(ctx, zone, record)
		return
	}

	iZone, err := zone.GetICloudDnsZone(ctx)
	if err != nil {
		self.taskFailed(ctx, record, errors.Wrapf(err, "GetICloudDnsZone"))
		return
	}

	opts := &cloudprovider.DnsRecord{
		DnsName:     record.Name,
		Desc:        record.Description,
		Enabled:     record.Enabled.Bool(),
		DnsType:     cloudprovider.TDnsType(record.DnsType),
		DnsValue:    record.DnsValue,
		Ttl:         record.TTL,
		MxPriority:  record.MxPriority,
		PolicyType:  cloudprovider.TDnsPolicyType(record.PolicyType),
		PolicyValue: cloudprovider.TDnsPolicyValue(record.PolicyValue),
	}

	id, err := iZone.AddDnsRecord(opts)
	if err != nil {
		self.taskFailed(ctx, record, errors.Wrapf(err, "AddDnsRecord"))
		return
	}
	_, err = db.Update(record, func() error {
		record.ExternalId = id
		return nil
	})
	if err != nil {
		self.taskFailed(ctx, record, errors.Wrapf(err, "set external id %s", id))
		return
	}

	self.taskComplete(ctx, zone, record)
}

func (self *DnsRecordCreateTask) taskComplete(ctx context.Context, zone *models.SDnsZone, record *models.SDnsRecord) {
	record.SetStatus(ctx, self.UserCred, api.DNS_RECORDSET_STATUS_AVAILABLE, "")
	logclient.AddSimpleActionLog(zone, logclient.ACT_ALLOCATE, record, self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    record,
		Action: notifyclient.ActionCreate,
	})
	self.SetStageComplete(ctx, nil)
}
