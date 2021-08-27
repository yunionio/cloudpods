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
)

func init() {
	taskman.RegisterTask(HAGuestStartTask{})
}

type HAGuestStartTask struct {
	GuestStartTask
}

func (self *HAGuestStartTask) OnInit(
	ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject,
) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_STARTING, nil, self.UserCred)
	self.RequestStartBacking(ctx, guest)
}

func (self *HAGuestStartTask) RequestStartBacking(ctx context.Context, guest *models.SGuest) {
	self.SetStage("OnStartBackupGuestComplete", nil)
	host := models.HostManager.FetchHostById(guest.BackupHostId)
	guest.SetStatus(self.UserCred, api.VM_BACKUP_STARTING, "")

	err := guest.GetDriver().RequestStartOnHost(ctx, guest, host, self.UserCred, self)
	if err != nil {
		self.OnStartCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *HAGuestStartTask) OnStartBackupGuestComplete(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	if data != nil {
		nbdServerPort, err := data.Int("nbd_server_port")
		if err == nil {
			backupHost := models.HostManager.FetchHostById(guest.BackupHostId)
			nbdServerUri := fmt.Sprintf("nbd:%s:%d", backupHost.AccessIp, nbdServerPort)
			guest.SetMetadata(ctx, "backup_nbd_server_uri", nbdServerUri, self.UserCred)
		} else {
			self.OnStartCompleteFailed(ctx, guest,
				jsonutils.NewString("Start backup guest result missing nbd_server_port"))
			return
		}
	}

	self.RequestStart(ctx, guest)
}

func (self *HAGuestStartTask) OnStartBackupGuestCompleteFailed(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	self.OnStartCompleteFailed(ctx, guest, data)
}
