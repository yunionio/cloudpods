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

type SNatSEntryDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SNatSEntryDeleteTask{})
}

func (self *SNatSEntryDeleteTask) taskFailed(ctx context.Context, snatEntry *models.SNatSEntry, err error) {
	snatEntry.SetStatus(self.UserCred, api.NAT_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(snatEntry, db.ACT_DELOCATE_FAIL, err.Error(), self.UserCred)
	logclient.AddActionLogWithStartable(self, snatEntry, logclient.ACT_DELETE, err.Error(), self.UserCred, false)
	self.SetStageFailed(ctx, err.Error())
}

func (self *SNatSEntryDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	snatEntry := obj.(*models.SNatSEntry)
	snatEntry.SetStatus(self.UserCred, api.NAT_STATUS_DELETING, "")
	cloudNatGateway, err := snatEntry.GetINatGateway()
	if err != nil {
		self.taskFailed(ctx, snatEntry, errors.Wrap(err, "Get NatGateway failed"))
		return
	}
	cloudNatSEntry, err := cloudNatGateway.GetINatSEntryByID(snatEntry.ExternalId)
	if err != nil {
		self.taskFailed(ctx, snatEntry, errors.Wrapf(err, "Get SNat Entry by ID '%s' failed", snatEntry.ExternalId))
		return
	}
	if cloudNatSEntry == nil {
		err = snatEntry.Purge(ctx, self.UserCred)
		if err != nil {
			self.taskFailed(ctx, snatEntry, err)
			return
		}
		logclient.AddActionLogWithStartable(self, snatEntry, logclient.ACT_DELETE, nil, self.UserCred, true)
		self.SetStageComplete(ctx, nil)
		return
	}

	err = cloudNatSEntry.Delete()
	if err != nil {
		self.taskFailed(ctx, snatEntry, errors.Wrapf(err, "Delete SNat Entry '%s' failed", snatEntry.ExternalId))
		return
	}

	err = cloudprovider.WaitDeleted(cloudNatSEntry, 10*time.Second, 300*time.Second)
	if err != nil {
		self.taskFailed(ctx, snatEntry, err)
		return
	}

	err = snatEntry.Purge(ctx, self.UserCred)
	if err != nil {
		self.taskFailed(ctx, snatEntry, err)
		return
	}

	db.OpsLog.LogEvent(snatEntry, db.ACT_DELETE, snatEntry.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, snatEntry, logclient.ACT_DELETE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
