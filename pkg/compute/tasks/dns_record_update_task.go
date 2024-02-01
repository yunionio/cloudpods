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

type DnsRecordUpdateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DnsRecordUpdateTask{})
}

func (self *DnsRecordUpdateTask) taskFailed(ctx context.Context, record *models.SDnsRecord, err error) {
	record.SetStatus(ctx, self.GetUserCred(), apis.STATUS_UNKNOWN, err.Error())
	db.OpsLog.LogEvent(record, db.ACT_UPDATE, record.GetShortDesc(ctx), self.GetUserCred())
	logclient.AddActionLogWithContext(ctx, record, logclient.ACT_UPDATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DnsRecordUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	record := obj.(*models.SDnsRecord)

	if len(record.ExternalId) == 0 {
		self.taskComplete(ctx, record)
		return
	}

	zone, err := record.GetDnsZone()
	if err != nil {
		self.taskFailed(ctx, record, errors.Wrapf(err, "GetDnsZone"))
		return
	}

	iZone, err := zone.GetICloudDnsZone(ctx)
	if err != nil {
		self.taskFailed(ctx, record, errors.Wrapf(err, "GetICloudDnsZone"))
		return
	}

	iRec, err := iZone.GetIDnsRecordById(record.ExternalId)
	if err != nil {
		self.taskFailed(ctx, record, errors.Wrapf(err, "GetIDnsRecordById"))
		return
	}

	opts := &cloudprovider.DnsRecord{
		Desc:        record.Description,
		DnsName:     record.Name,
		DnsValue:    record.DnsValue,
		DnsType:     cloudprovider.TDnsType(record.DnsType),
		Enabled:     record.Enabled.Bool(),
		Ttl:         record.TTL,
		MxPriority:  record.MxPriority,
		PolicyType:  cloudprovider.TDnsPolicyType(record.PolicyType),
		PolicyValue: cloudprovider.TDnsPolicyValue(record.PolicyValue),
	}

	err = iRec.Update(opts)
	if err != nil {
		self.taskFailed(ctx, record, errors.Wrapf(err, "Update"))
		return
	}

	self.taskComplete(ctx, record)
}

func (self *DnsRecordUpdateTask) taskComplete(ctx context.Context, record *models.SDnsRecord) {
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    record,
		Action: notifyclient.ActionUpdate,
	})
	record.SetStatus(ctx, self.UserCred, apis.SKU_STATUS_AVAILABLE, "")
	self.SetStageComplete(ctx, nil)
}
