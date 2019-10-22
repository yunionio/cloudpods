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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SNatSEntryDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SNatSEntryDeleteTask{})
}

func (self *SNatSEntryDeleteTask) TaskFailed(ctx context.Context, snatEntry *models.SNatSEntry, err error) {
	snatEntry.SetStatus(self.UserCred, api.NAT_STATUS_DELETE_FAILED, err.Error())
	db.OpsLog.LogEvent(snatEntry, db.ACT_DELOCATE_FAIL, err.Error(), self.UserCred)
	natgateway, err := snatEntry.GetNatgateway()
	if err == nil {
		logclient.AddActionLogWithStartable(self, natgateway, logclient.ACT_NAT_DELETE_SNAT, nil, self.UserCred, false)
	} else {
		logclient.AddActionLogWithStartable(self, snatEntry, logclient.ACT_DELETE, nil, self.UserCred, false)
	}
	self.SetStageFailed(ctx, err.Error())
}

func (self *SNatSEntryDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	snatEntry := obj.(*models.SNatSEntry)
	snatEntry.SetStatus(self.UserCred, api.NAT_STATUS_DELETING, "")
	natgateway, err := snatEntry.GetNatgateway()
	if err != nil {
		self.TaskFailed(ctx, snatEntry, err)
	}
	cloudNatGateway, err := snatEntry.GetINatGateway()
	if err != nil {
		self.TaskFailed(ctx, snatEntry, errors.Wrap(err, "Get NatGateway failed"))
		return
	}
	cloudNatSEntry, err := cloudNatGateway.GetINatSEntryByID(snatEntry.ExternalId)
	if err == cloudprovider.ErrNotFound {
		//already delete
	} else if err != nil {
		self.TaskFailed(ctx, snatEntry, errors.Wrapf(err, "Get SNat Entry by ID '%s' failed", snatEntry.ExternalId))
		return
	} else if cloudNatSEntry != nil {
		err = cloudNatSEntry.Delete()
		if err != nil {
			self.TaskFailed(ctx, snatEntry, errors.Wrapf(err, "Delete SNat Entry '%s' failed", snatEntry.ExternalId))
			return
		}

		err = cloudprovider.WaitDeleted(cloudNatSEntry, 10*time.Second, 300*time.Second)
		if err != nil {
			self.TaskFailed(ctx, snatEntry, err)
			return
		}
	}

	err = snatEntry.Purge(ctx, self.UserCred)
	if err != nil {
		self.TaskFailed(ctx, snatEntry, err)
		return
	}

	// Try to dissociate eip with natgateway if there is no nat rule using this eip and task is set ok even if
	// dissociate failed.
	err = natgateway.GetRegion().GetDriver().RequestUnBindIPFromNatgateway(ctx, self, snatEntry, natgateway)
	if err != nil {
		log.Debugf("fail to try to dissociate eip with natgateway %s", natgateway.GetId())
	}

	logclient.AddActionLogWithStartable(self, natgateway, logclient.ACT_NAT_DELETE_SNAT, nil, self.UserCred, true)

	db.OpsLog.LogEvent(snatEntry, db.ACT_DELETE, snatEntry.GetShortDesc(ctx), self.UserCred)
	self.SetStageComplete(ctx, nil)
}
