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
	taskman.RegisterTask(GuestDeployBackupTask{})
	taskman.RegisterTask(HAGuestDeployTask{})
}

type HAGuestDeployTask struct {
	GuestDeployTask
}

func (self *HAGuestDeployTask) OnDeployWaitServerStop(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	host := models.HostManager.FetchHostById(guest.BackupHostId)
	if host.HostStatus != api.HOST_ONLINE {
		self.GuestDeployTask.OnDeployWaitServerStop(ctx, guest, data)
	} else {
		self.DeployBackup(ctx, guest, nil)
	}
}

func (self *HAGuestDeployTask) DeployBackup(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStage("OnDeploySlaveGuestComplete", nil)
	host := models.HostManager.FetchHostById(guest.BackupHostId)
	drv, err := guest.GetDriver()
	if err != nil {
		self.OnDeployGuestFail(ctx, guest, err)
		return
	}
	err = drv.RequestDeployGuestOnHost(ctx, guest, host, self)
	if err != nil {
		self.OnDeployGuestFail(ctx, guest, err)
		return
	}
	guest.SetStatus(ctx, self.UserCred, api.VM_DEPLOYING_BACKUP, "")
}

func (self *HAGuestDeployTask) OnDeploySlaveGuestComplete(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	guest.SetGuestBackupMirrorJobNotReady(ctx, self.UserCred)
	host, _ := guest.GetHost()
	self.SetStage("OnDeployGuestComplete", nil)
	self.DeployOnHost(ctx, guest, host)
}

func (self *HAGuestDeployTask) OnDeploySlaveGuestCompleteFailed(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	guest.SetGuestBackupMirrorJobNotReady(ctx, self.UserCred)
	self.OnDeployGuestFail(ctx, guest, fmt.Errorf("deploy backup failed %s", data))
}

type GuestDeployBackupTask struct {
	HAGuestDeployTask
}

func (self *GuestDeployBackupTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if len(guest.BackupHostId) == 0 {
		self.OnDeployGuestCompleteFailed(ctx, guest, jsonutils.NewString("Guest dosen't have backup host"))
		return
	}
	self.SetStage("OnDeployGuestComplete", nil)
	host := models.HostManager.FetchHostById(guest.BackupHostId)
	drv, err := guest.GetDriver()
	if err != nil {
		self.OnDeployGuestCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	err = drv.RequestDeployGuestOnHost(ctx, guest, host, self)
	if err != nil {
		self.OnDeployGuestCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (self *GuestDeployBackupTask) OnDeployGuestComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetGuestBackupMirrorJobNotReady(ctx, self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *GuestDeployBackupTask) OnDeployGuestCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetGuestBackupMirrorJobNotReady(ctx, self.UserCred)
	guest.SetStatus(ctx, self.UserCred, api.VM_DEPLOYING_BACKUP_FAILED, data.String())
	self.SetStageComplete(ctx, nil)
}
