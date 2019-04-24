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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

func init() {
	taskman.RegisterTask(GuestDeleteOnHostTask{})
}

type GuestDeleteOnHostTask struct {
	SGuestBaseTask
}

func (self *GuestDeleteOnHostTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	hostId, err := self.Params.GetString("host_id")
	if err != nil {
		self.OnFail(ctx, guest, "Missing param host id")
		return
	}
	host := models.HostManager.FetchHostById(hostId)
	if host == nil {
		self.OnFail(ctx, guest, "Host is nil")
		return
	}

	self.SetStage("OnStopGuest", nil)
	self.Params.Set("is_force", jsonutils.JSONTrue)
	if err := guest.GetDriver().RequestStopOnHost(ctx, guest, host, self); err != nil {
		log.Errorf("RequestStopGuestForDelete fail %s", err)
		self.OnStopGuest(ctx, guest, nil)
	}
}

func (self *GuestDeleteOnHostTask) OnStopGuest(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	hostId, _ := self.Params.GetString("host_id")
	host := models.HostManager.FetchHostById(hostId)

	isPurge := jsonutils.QueryBoolean(self.Params, "purge", false)
	disks := guest.GetDisks()

	for _, guestDiks := range disks {
		disk := guestDiks.GetDisk()
		storage := host.GetStorageByFilePath(disk.AccessPath)
		// storage := models.StorageManager.FetchStorageById(disk.BackupStorageId)
		if storage != nil && !isPurge {
			if err := host.GetHostDriver().RequestDeallocateBackupDiskOnHost(ctx, host, storage, disk, self); err != nil {
				self.OnFail(ctx, guest, err.Error())
				return
			}
		}
		if disk.BackupStorageId == storage.Id {
			_, err := db.Update(disk, func() error {
				disk.BackupStorageId = ""
				return nil
			})
			if err != nil {
				self.OnFail(ctx, guest, err.Error())
				return
			}
		}
	}
	if !isPurge {
		self.SetStage("OnUnDeployGuest", nil)
		guest.StartUndeployGuestTask(ctx, self.GetUserCred(), self.GetTaskId(), hostId)
	} else {
		self.OnUnDeployGuest(ctx, guest, nil)
	}
}

func (self *GuestDeleteOnHostTask) OnUnDeployGuest(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	hostId, _ := self.Params.GetString("host_id")
	if guest.BackupHostId == hostId {
		_, err := db.Update(guest, func() error {
			guest.BackupHostId = ""
			return nil
		})
		if err != nil {
			self.OnFail(ctx, guest, err.Error())
			return
		}
	}
	self.SetStage("OnSync", nil)
	guest.StartSyncTask(ctx, self.UserCred, false, self.GetTaskId())
}

func (self *GuestDeleteOnHostTask) OnSync(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestDeleteOnHostTask) OnFail(ctx context.Context, guest *models.SGuest, reason string) {
	failedStatus, _ := self.Params.GetString("failed_status")
	if len(failedStatus) > 0 {
		guest.SetStatus(self.UserCred, failedStatus, reason)
	}
	self.SetStageFailed(ctx, reason)
}
