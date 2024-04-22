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
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

func init() {
	taskman.RegisterTask(HAGuestStopTask{})
}

type HAGuestStopTask struct {
	GuestStopTask
}

func (self *HAGuestStopTask) OnGuestStopTaskComplete(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	host := models.HostManager.FetchHostById(guest.BackupHostId)
	if host.HostStatus != api.HOST_ONLINE {
		self.GuestStopTask.OnGuestStopTaskComplete(ctx, guest, data)
		return
	}

	self.SetStage("OnSlaveGuestStopTaskComplete", nil)
	drv, err := guest.GetDriver()
	if err != nil {
		self.OnGuestStopTaskCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	err = drv.RequestStopOnHost(ctx, guest, host, self, true)
	if err != nil {
		log.Errorf("RequestStopOnHost fail %s", err)
		self.OnGuestStopTaskCompleteFailed(
			ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (self *HAGuestStopTask) OnSlaveGuestStopTaskComplete(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	self.GuestStopTask.OnGuestStopTaskComplete(ctx, guest, data)
}

func (self *HAGuestStopTask) OnSlaveGuestStopTaskCompleteFailed(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	guest.SetBackupGuestStatus(self.UserCred, compute.VM_STOP_FAILED, data.String())
	self.OnGuestStopTaskCompleteFailed(ctx, guest, data)
}
