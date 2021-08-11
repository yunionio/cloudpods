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
	"fmt"
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

type SNatDEntryDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SNatDEntryDeleteTask{})
}

func (self *SNatDEntryDeleteTask) TaskFailed(ctx context.Context, dnatEntry *models.SNatDEntry, reason jsonutils.JSONObject) {
	dnatEntry.SetStatus(self.UserCred, api.NAT_STATUS_DELETE_FAILED, reason.String())
	db.OpsLog.LogEvent(dnatEntry, db.ACT_DELOCATE_FAIL, reason, self.UserCred)
	natgateway, err := dnatEntry.GetNatgateway()
	if err == nil {
		logclient.AddActionLogWithStartable(self, natgateway, logclient.ACT_NAT_DELETE_DNAT, reason, self.UserCred, false)
	} else {
		logclient.AddActionLogWithStartable(self, dnatEntry, logclient.ACT_NAT_DELETE_DNAT, reason, self.UserCred, false)
	}
	self.SetStageFailed(ctx, reason)
}

func (self *SNatDEntryDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	dnatEntry := obj.(*models.SNatDEntry)
	dnatEntry.SetStatus(self.UserCred, api.NAT_STATUS_DELETING, "")
	natgateway, err := dnatEntry.GetNatgateway()
	if err != nil {
		self.TaskFailed(ctx, dnatEntry, jsonutils.NewString(err.Error()))
	}
	cloudNatGateway, err := dnatEntry.GetINatGateway()
	if err != nil {
		self.TaskFailed(ctx, dnatEntry, jsonutils.NewString(fmt.Sprintf("Get NatGateway failed: %s", err)))
		return
	}

	cloudNatDEntry, err := cloudNatGateway.GetINatDEntryByID(dnatEntry.ExternalId)
	if errors.Cause(err) == cloudprovider.ErrNotFound {
		// already delete
	} else if err != nil {
		self.TaskFailed(ctx, dnatEntry, jsonutils.NewString(fmt.Sprintf("Get DNat Entry by ID '%s' failed", dnatEntry.ExternalId)))
		return
	} else if cloudNatDEntry != nil {
		err = cloudNatDEntry.Delete()
		if err != nil {
			self.TaskFailed(ctx, dnatEntry, jsonutils.NewString(fmt.Sprintf("Delete DNat Entry '%s' failed: %s", dnatEntry.ExternalId, err)))
			return
		}

		err = cloudprovider.WaitDeleted(cloudNatDEntry, 10*time.Second, 300*time.Second)
		if err != nil {
			self.TaskFailed(ctx, dnatEntry, jsonutils.NewString(err.Error()))
			return
		}
	}

	err = dnatEntry.Purge(ctx, self.UserCred)
	if err != nil {
		self.TaskFailed(ctx, dnatEntry, jsonutils.NewString(err.Error()))
		return
	}

	// Try to dissociate eip with natgateway if there is no nat rule using this eip and task is set ok even if
	// dissociate failed.
	region, _ := natgateway.GetRegion()
	err = region.GetDriver().RequestUnBindIPFromNatgateway(ctx, self, dnatEntry, natgateway)
	if err != nil {
		log.Debugf("fail to try to dissociate eip with natgateway %s", natgateway.GetId())
	}

	db.OpsLog.LogEvent(dnatEntry, db.ACT_DELETE, dnatEntry.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, natgateway, logclient.ACT_NAT_DELETE_DNAT, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
