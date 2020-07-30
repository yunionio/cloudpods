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

func (self *SNatSEntryCreateTask) TaskFailed(ctx context.Context, snatEntry models.INatHelper, reason jsonutils.JSONObject) {
	snatEntry.SetStatus(self.UserCred, api.NAT_STATUS_FAILED, reason.String())
	db.OpsLog.LogEvent(snatEntry, db.ACT_ALLOCATE_FAIL, reason, self.UserCred)
	natgateway, err := snatEntry.GetNatgateway()
	if err == nil {
		logclient.AddActionLogWithStartable(self, natgateway, logclient.ACT_NAT_CREATE_SNAT, reason, self.UserCred, false)
	} else {
		logclient.AddActionLogWithStartable(self, snatEntry, logclient.ACT_ALLOCATE, reason, self.UserCred, false)
	}
	self.SetStageFailed(ctx, reason)
}

func (self *SNatSEntryCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	snatEntry := obj.(*models.SNatSEntry)
	NatToBindIPStage(ctx, self, snatEntry)
}

func (self *SNatSEntryCreateTask) OnBindIPCompleteFailed(ctx context.Context, snatEntry *models.SNatSEntry,
	reason jsonutils.JSONObject) {

	self.TaskFailed(ctx, snatEntry, reason)
}

func (self *SNatSEntryCreateTask) OnBindIPComplete(ctx context.Context, snatEntry *models.SNatSEntry,
	body jsonutils.JSONObject) {

	cloudNatGateway, err := snatEntry.GetINatGateway()
	if err != nil {
		self.TaskFailed(ctx, snatEntry, jsonutils.NewString(fmt.Sprintf("Get NatGateway failed: %s", err)))
		return
	}

	externalIPID, err := self.Params.GetString("eip_external_id")
	// construct a DNat RUle
	snatRule := cloudprovider.SNatSRule{
		ExternalIP:   snatEntry.IP,
		ExternalIPID: externalIPID,
	}
	if self.Params.Contains("network_ext_id") {
		extID, _ := self.Params.GetString("network_ext_id")
		snatRule.NetworkID = extID
	} else {
		snatRule.SourceCIDR = snatEntry.SourceCIDR
	}
	extSnat, err := cloudNatGateway.CreateINatSEntry(snatRule)
	if err != nil {
		// create nat failed in cloud
		if self.Params.Contains("need_bind") {
			err1 := CreateINatFailedRollback(ctx, self, snatEntry)
			if err1 != nil {
				eip_id, _ := self.Params.GetString("eip_id")
				log.Errorf("roll back after failing to create snat in cloud so that eip %s need to sync with cloud", eip_id)
			}
		}
		self.TaskFailed(ctx, snatEntry, jsonutils.NewString(fmt.Sprintf("Create SNat Entry '%s' failed: %s", snatEntry.ExternalId, err)))
		return
	}

	err = cloudprovider.WaitStatus(extSnat, api.NAT_STAUTS_AVAILABLE, 10*time.Second, 300*time.Second)
	if err != nil {
		self.TaskFailed(ctx, snatEntry, jsonutils.NewString(err.Error()))
		return
	}

	err = db.SetExternalId(snatEntry, self.UserCred, extSnat.GetGlobalId())
	if err != nil {
		self.TaskFailed(ctx, snatEntry, jsonutils.NewString(fmt.Sprintf("set external id failed: %s", err)))
		return
	}

	snatEntry.SetStatus(self.UserCred, api.NAT_STAUTS_AVAILABLE, "")
	db.OpsLog.LogEvent(snatEntry, db.ACT_ALLOCATE, snatEntry.GetShortDesc(ctx), self.UserCred)
	natgateway, err := snatEntry.GetNatgateway()
	if err == nil {
		logclient.AddActionLogWithStartable(self, natgateway, logclient.ACT_NAT_CREATE_SNAT, nil, self.UserCred, true)
	} else {
		logclient.AddActionLogWithStartable(self, snatEntry, logclient.ACT_ALLOCATE, nil, self.UserCred, true)
	}
	self.SetStageComplete(ctx, nil)
}
