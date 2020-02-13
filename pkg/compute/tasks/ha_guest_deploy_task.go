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
)

func init() {
	taskman.RegisterTask(GuestDeployBackupTask{})
	taskman.RegisterTask(HAGuestDeployTask{})
}

type HAGuestDeployTask struct {
	GuestDeployTask
}

func (self *HAGuestDeployTask) OnDeployGuestComplete(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	self.DeployBackup(ctx, guest, data)
}

func (self *HAGuestDeployTask) DeployBackup(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	host := models.HostManager.FetchHostById(guest.BackupHostId)
	err := guest.GetDriver().RequestDeployGuestOnHost(ctx, guest, host, self)
	if err != nil {
		log.Errorf("request_deploy_guest_on_host %s", err)
		self.OnDeployGuestFail(ctx, guest, err)
	} else {
		guest.SetStatus(self.UserCred, api.VM_DEPLOYING_BACKUP, "")
	}
}

func (self *HAGuestDeployTask) OnDeploySlaveGuestComplete(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	self.GuestDeployTask.OnDeployGuestComplete(ctx, guest, data)
}

func (self *HAGuestDeployTask) OnDeploySlaveGuestCompleteFailed(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	self.OnDeployGuestFail(ctx, guest, fmt.Errorf("deploy backup failed %s", data))
}

type GuestDeployBackupTask struct {
	HAGuestDeployTask
}

func (self *GuestDeployBackupTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if len(guest.BackupHostId) == 0 {
		self.SetStageFailed(ctx, "Guest dosen't have backup host")
	}
	self.SetStage("OnDeployGuestComplete", nil)
	self.DeployBackup(ctx, guest, nil)
}

func (self *GuestDeployBackupTask) OnDeployGuestComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestDeployBackupTask) OnDeployGuestCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetStatus(self.UserCred, api.VM_DEPLOYING_BACKUP_FAILED, data.String())
	self.SetStageComplete(ctx, nil)
}
