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

type SNatSEntryCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SNatSEntryCreateTask{})
}

func (self *SNatSEntryCreateTask) TaskFailed(ctx context.Context, snatEntry *models.SNatSEntry, err error) {
	snatEntry.SetStatus(self.UserCred, api.NAT_STATUS_FAILED, err.Error())
	db.OpsLog.LogEvent(snatEntry, db.ACT_ALLOCATE_FAIL, err.Error(), self.UserCred)
	logclient.AddActionLogWithStartable(self, snatEntry, logclient.ACT_ALLOCATE, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, err.Error())
}

func (self *SNatSEntryCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	snatEntry := obj.(*models.SNatSEntry)
	snatEntry.SetStatus(self.UserCred, api.NAT_STATUS_ALLOCATE, "")
	cloudNatGateway, err := snatEntry.GetINatGateway()
	if err != nil {
		self.TaskFailed(ctx, snatEntry, errors.Wrap(err, "Get NatGateway failed"))
		return
	}

	externalIPID, err := body.GetString("external_ip_id")
	// construct a DNat RUle
	snatRule := cloudprovider.SNatSRule{
		SourceCIDR:   snatEntry.SourceCIDR,
		ExternalIP:   snatEntry.IP,
		ExternalIPID: externalIPID,
	}
	_, err = cloudNatGateway.CreateINatSEntry(snatRule)
	if err != nil {
		self.TaskFailed(ctx, snatEntry, errors.Wrapf(err, "Create SNat Entry '%s' failed", snatEntry.ExternalId))
		return
	}

	snatEntry.SetStatus(self.UserCred, api.NAT_STAUTS_AVAILABLE, "")

	logclient.AddActionLogWithStartable(self, snatEntry, logclient.ACT_ALLOCATE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
