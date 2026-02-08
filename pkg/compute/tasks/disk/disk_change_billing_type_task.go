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

package disk

import (
	"context"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DiskChangeBillingTypeTask struct {
	SDiskBaseTask
}

func init() {
	taskman.RegisterTask(DiskChangeBillingTypeTask{})
}

func (self *DiskChangeBillingTypeTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)

	idisk, err := disk.GetIDisk(ctx)
	if err != nil {
		self.taskFail(ctx, disk, errors.Wrapf(err, "GetIDisk"))
		return
	}

	var billType billing_api.TBillingType
	switch disk.BillingType {
	case billing_api.BILLING_TYPE_POSTPAID:
		billType = billing_api.BILLING_TYPE_PREPAID
	case billing_api.BILLING_TYPE_PREPAID:
		billType = billing_api.BILLING_TYPE_POSTPAID
	}

	err = idisk.ChangeBillingType(string(billType))
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotImplemented {
			disk.SetStatus(ctx, self.GetUserCred(), api.DISK_READY, "")
			self.SetStageComplete(ctx, nil)
			return
		}
		self.taskFail(ctx, disk, errors.Wrapf(err, "ChangeBillingType"))
		return
	}

	idisk.Refresh()

	db.Update(disk, func() error {
		disk.BillingType = billType
		disk.ExpiredAt = time.Time{}
		disk.AutoRenew = false
		if disk.BillingType == billing_api.BILLING_TYPE_PREPAID {
			disk.AutoRenew = idisk.IsAutoRenew()
			disk.ExpiredAt = idisk.GetExpiredAt()
		}
		return nil
	})

	self.taskComplete(ctx, disk)
}

func (self *DiskChangeBillingTypeTask) taskComplete(ctx context.Context, disk *models.SDisk) {
	disk.SetStatus(ctx, self.GetUserCred(), api.DISK_READY, "")
	logclient.AddActionLogWithStartable(self, disk, logclient.ACT_CHANGE_BILLING_TYPE, disk.BillingType, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *DiskChangeBillingTypeTask) taskFail(ctx context.Context, disk *models.SDisk, err error) {
	disk.SetStatus(ctx, self.GetUserCred(), apis.STATUS_CHANGE_BILLING_TYPE_FAILED, "")
	logclient.AddActionLogWithStartable(self, disk, logclient.ACT_CHANGE_BILLING_TYPE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}
