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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestSyncstatusTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestSyncstatusTask{})
}

func (self *GuestSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	host, err := guest.GetHost()
	if err != nil {
		guest.SetStatus(ctx, self.UserCred, api.VM_UNKNOWN, fmt.Sprintf("get host error: %v", err))
		self.SetStageComplete(ctx, nil)
		return
	}
	if !host.IsBaremetal && host.HostStatus == api.HOST_OFFLINE {
		guest.SetStatus(ctx, self.UserCred, api.VM_UNKNOWN, "host offline")
		self.SetStageComplete(ctx, nil)
		return
	}
	self.SetStage("OnGetStatusComplete", nil)
	drv, err := guest.GetDriver()
	if err != nil {
		self.OnGetStatusCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	err = drv.RequestSyncstatusOnHost(ctx, guest, host, self.UserCred, self)
	if err != nil {
		self.OnGetStatusCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *GuestSyncstatusTask) getOriginStatus() string {
	os, _ := self.GetParams().GetString("origin_status")
	return os
}

func (self *GuestSyncstatusTask) OnGetStatusComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	log.Debugf("OnGetStatusSucc guest %s(%s) status %s", guest.Name, guest.Id, body)
	resp := new(api.HostUploadGuestStatusResponse)
	body.Unmarshal(resp)
	if err := guest.SetStatusFromHost(ctx, self.GetUserCred(), *resp, self.HasParentTask(), self.getOriginStatus()); err != nil {
		log.Warningf("SetStatusFromHost for guest %s error: %v", guest.GetId(), err)
	}
	logclient.AddSimpleActionLog(guest, logclient.ACT_VM_SYNC_STATUS, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *GuestSyncstatusTask) OnGetStatusCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(ctx, self.UserCred, api.VM_UNKNOWN, err.String())
	logclient.AddSimpleActionLog(guest, logclient.ACT_VM_SYNC_STATUS, err, self.UserCred, false)
	self.SetStageComplete(ctx, nil)
}
