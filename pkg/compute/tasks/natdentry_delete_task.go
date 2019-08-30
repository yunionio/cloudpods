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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SNatDEntryDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SNatDEntryDeleteTask{})
}

func (self *SNatDEntryDeleteTask) taskFailed(ctx context.Context, dnatEntry *models.SNatDEntry, err error) {
	dnatEntry.SetStatus(self.UserCred, api.NAT_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(dnatEntry, db.ACT_DELOCATE_FAIL, err.Error(), self.UserCred)
	logclient.AddActionLogWithStartable(self, dnatEntry, logclient.ACT_DELETE, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, err.Error())
}

func (self *SNatDEntryDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	dnatEntry := obj.(*models.SNatDEntry)
	dnatEntry.SetStatus(self.UserCred, api.NAT_STATUS_DELETING, "")
	cloudNatGateway, err := dnatEntry.GetINatGateway()
	if err != nil {
		self.taskFailed(ctx, dnatEntry, errors.Wrap(err, "Get NatGateway failed"))
		return
	}
	cloudNatDEntry, err := cloudNatGateway.GetINatDEntryByID(dnatEntry.ExternalId)
	if err != nil {
		self.taskFailed(ctx, dnatEntry, errors.Wrapf(err, "Get DNat Entry by ID '%s' failed", dnatEntry.ExternalId))
		return
	}
	if cloudNatDEntry == nil {
		err = dnatEntry.Purge(ctx, self.UserCred)
		if err != nil {
			self.taskFailed(ctx, dnatEntry, err)
			return
		}
		logclient.AddActionLogWithStartable(self, dnatEntry, logclient.ACT_DELETE, nil, self.UserCred, true)
		self.SetStageComplete(ctx, nil)
		return
	}

	err = cloudNatDEntry.Delete()
	if err != nil {
		self.taskFailed(ctx, dnatEntry, errors.Wrapf(err, "Delete DNat Entry '%s' failed", dnatEntry.ExternalId))
		return
	}

	err = cloudprovider.WaitDeleted(cloudNatDEntry, 10*time.Second, 300*time.Second)
	if err != nil {
		self.taskFailed(ctx, dnatEntry, err)
		return
	}

	err = dnatEntry.Purge(ctx, self.UserCred)
	if err != nil {
		self.taskFailed(ctx, dnatEntry, err)
		return
	}

	db.OpsLog.LogEvent(dnatEntry, db.ACT_DELETE, dnatEntry.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, dnatEntry, logclient.ACT_DELETE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
