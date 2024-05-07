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
	host := models.HostManager.FetchHostById(guest.BackupHostId)
	if host.HostStatus != api.HOST_ONLINE {
		guest.SetGuestBackupMirrorJobFailed(ctx, self.UserCred)
		// request start master guest
		self.GuestStartTask.OnInit(ctx, guest, nil)
	} else {
		self.RequestStopBackupGuest(ctx, guest)
	}
}

func (self *HAGuestStartTask) RequestStopBackupGuest(ctx context.Context, guest *models.SGuest) {
	host := models.HostManager.FetchHostById(guest.BackupHostId)
	self.SetStage("OnBackupGuestStopComplete", nil)
	guest.SetStatus(ctx, self.UserCred, api.VM_BACKUP_STOPING, "HAGuestStartTask")
	drv, err := guest.GetDriver()
	if err != nil {
		guest.SetStatus(ctx, self.UserCred, api.VM_BACKUP_START_FAILED, err.Error())
		self.SetStageFailed(ctx, nil)
		return
	}
	err = drv.RequestStopOnHost(ctx, guest, host, self, false)
	if err != nil {
		guest.SetStatus(ctx, self.UserCred, api.VM_BACKUP_START_FAILED, err.Error())
		self.SetStageFailed(ctx, nil)
	}
}

func (self *HAGuestStartTask) OnBackupGuestStopComplete(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	db.OpsLog.LogEvent(guest, db.ACT_STARTING, nil, self.UserCred)
	self.RequestStartBacking(ctx, guest)
}

func (self *HAGuestStartTask) OnBackupGuestStopCompleteFailed(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	guest.SetStatus(ctx, self.UserCred, api.VM_BACKUP_START_FAILED, data.String())
	self.SetStageFailed(ctx, data)
}

func (self *HAGuestStartTask) RequestStartBacking(ctx context.Context, guest *models.SGuest) {
	self.SetStage("OnStartBackupGuestComplete", nil)
	host := models.HostManager.FetchHostById(guest.BackupHostId)
	guest.SetStatus(ctx, self.UserCred, api.VM_BACKUP_STARTING, "")

	if !guest.IsGuestBackupMirrorJobReady(ctx, self.UserCred) {
		hostMaster := models.HostManager.FetchHostById(guest.HostId)
		self.Params.Set("block_ready", jsonutils.JSONFalse)
		diskUri := fmt.Sprintf("%s/disks", hostMaster.GetFetchUrl(true))
		self.Params.Set("disk_uri", jsonutils.NewString(diskUri))
	} else {
		self.Params.Set("block_ready", jsonutils.JSONTrue)
	}

	drv, err := guest.GetDriver()
	if err != nil {
		self.OnStartCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}

	err = drv.RequestStartOnHost(ctx, guest, host, self.UserCred, self)
	if err != nil {
		self.OnStartCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
}

func (self *HAGuestStartTask) OnStartBackupGuestComplete(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	if data != nil && !jsonutils.QueryBoolean(data, "is_running", false) {
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
	if err := guest.ResetGuestQuorumChildIndex(ctx, self.UserCred); err != nil {
		self.OnStartBackupGuestCompleteFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("failed reset quorum child index: %s", err)))
		return
	}
	guest.SetBackupGuestStatus(self.UserCred, api.VM_RUNNING, "on start backup guest complete")
	self.RequestStart(ctx, guest)
}

func (self *HAGuestStartTask) OnStartBackupGuestCompleteFailed(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	guest.SetBackupGuestStatus(self.UserCred, api.VM_START_FAILED, data.String())
	self.OnStartCompleteFailed(ctx, guest, data)
}

func (self *HAGuestStartTask) OnStartComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if !guest.IsGuestBackupMirrorJobReady(ctx, self.UserCred) {
		drv, err := guest.GetDriver()
		if err != nil {
			guest.SetGuestBackupMirrorJobFailed(ctx, self.UserCred)
			guest.SetBackupGuestStatus(self.UserCred, api.VM_BLOCK_STREAM_FAIL, err.Error())
			return
		}
		if err := drv.RequestSlaveBlockStreamDisks(ctx, guest, self); err != nil {
			guest.SetGuestBackupMirrorJobFailed(ctx, self.UserCred)
			guest.SetBackupGuestStatus(self.UserCred, api.VM_BLOCK_STREAM_FAIL, err.Error())
		} else {
			guest.SetGuestBackupMirrorJobInProgress(ctx, self.UserCred)
			guest.SetBackupGuestStatus(self.UserCred, api.VM_BLOCK_STREAM, "on RequestSlaveBlockStreamDisks")
		}
	}

	self.GuestStartTask.OnStartComplete(ctx, guest, data)
}
