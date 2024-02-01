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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestBlockIoThrottleTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestBlockIoThrottleTask{})
}

func (self *GuestBlockIoThrottleTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	url := fmt.Sprintf("/servers/%s/io-throttle", guest.Id)
	headers := self.GetTaskRequestHeader()
	host, _ := guest.GetHost()
	self.SetStage("OnIoThrottle", nil)

	input := new(api.ServerSetDiskIoThrottleInput)
	if err := self.Params.Unmarshal(input); err != nil {
		self.OnIoThrottleFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	_, err := host.Request(ctx, self.UserCred, "POST", url, headers, jsonutils.Marshal(input))
	if err != nil {
		self.OnIoThrottleFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (self *GuestBlockIoThrottleTask) OnIoThrottle(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_VM_IO_THROTTLE, "", self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_VM_IO_THROTTLE, "", self.UserCred, true)
	self.SetStage("OnGuestSync", nil)
	guest.StartSyncstatus(ctx, self.UserCred, self.Id)
}

func (self *GuestBlockIoThrottleTask) OnGuestSync(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	oldStatus, _ := self.Params.GetString("old_status")
	if len(oldStatus) > 0 {
		guest.SetStatus(ctx, self.UserCred, oldStatus, "on io throttle")
	}
	self.SetStageComplete(ctx, nil)
}

func (self *GuestBlockIoThrottleTask) OnGuestSyncFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.OnIoThrottleFailed(ctx, guest, data)
}

func (self *GuestBlockIoThrottleTask) OnIoThrottleFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_VM_IO_THROTTLE_FAIL, data, self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_VM_IO_THROTTLE, data, self.UserCred, false)
	guest.SetStatus(ctx, self.UserCred, api.VM_IO_THROTTLE_FAIL, data.String())
	self.SetStageFailed(ctx, data)
}
