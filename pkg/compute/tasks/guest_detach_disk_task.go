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

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestDetachDiskTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestDetachDiskTask{})
}

func (self *GuestDetachDiskTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	diskId, _ := self.Params.GetString("disk_id")
	objDisk, err := models.DiskManager.FetchById(diskId)
	if err != nil {
		self.OnTaskFail(ctx, guest, nil, err)
		return
	}
	disk := objDisk.(*models.SDisk)
	if disk == nil {
		self.OnTaskFail(ctx, guest, nil, fmt.Errorf("Connot find disk %s", diskId))
		return
	}

	guestdisks := disk.GetGuestdisks()
	if len(guestdisks) > 0 {
		guestdisk := guestdisks[0]
		self.Params.Add(jsonutils.NewString(guestdisk.Driver), "driver")
		self.Params.Add(jsonutils.NewString(guestdisk.CacheMode), "cache")
		self.Params.Add(jsonutils.NewString(guestdisk.Mountpoint), "mountpoint")
	}

	guest.DetachDisk(ctx, disk, self.UserCred)
	host := guest.GetHost()
	if host != nil && !host.Enabled && jsonutils.QueryBoolean(self.Params, "purge", false) {
		self.OnDetachDiskComplete(ctx, guest, nil)
		return
	}

	self.SetStage("OnDetachDiskComplete", nil)
	err = guest.GetDriver().RequestDetachDisk(ctx, guest, disk, self)
	if err != nil {
		self.OnDetachDiskCompleteFailed(ctx, guest, jsonutils.Marshal(map[string]string{"error": err.Error()}))
	}
}

func (self *GuestDetachDiskTask) OnDetachDiskComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	diskId, _ := self.Params.GetString("disk_id")
	objDisk, err := models.DiskManager.FetchById(diskId)
	if err != nil {
		self.OnTaskFail(ctx, guest, nil, err)
		return
	}
	disk := objDisk.(*models.SDisk)
	if disk == nil {
		self.OnTaskFail(ctx, guest, nil, fmt.Errorf("Connot find disk %s", diskId))
		return
	}
	disk.SetStatus(self.UserCred, api.DISK_READY, "")
	keepDisk := jsonutils.QueryBoolean(self.Params, "keep_disk", true)
	host := guest.GetHost()
	purge := false
	if host != nil && !host.Enabled && jsonutils.QueryBoolean(self.Params, "purge", false) {
		purge = true
	}
	if !keepDisk && disk.AutoDelete {
		cnt, _ := disk.GetGuestDiskCount()
		if cnt == 0 {
			self.SetStage("OnDiskDeleteComplete", nil)
			db.OpsLog.LogEvent(disk, db.ACT_DELETE, "", self.UserCred)
			err := guest.GetDriver().RequestDeleteDetachedDisk(ctx, disk, self, purge)
			if err != nil {
				self.OnTaskFail(ctx, guest, disk, err)
			}
			return
		}
	}
	self.OnDiskDeleteComplete(ctx, guest, nil)
}

func (self *GuestDetachDiskTask) OnDetachDiskCompleteFailed(ctx context.Context, obj db.IStandaloneModel, reason jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	driver, _ := self.Params.GetString("driver")
	cache, _ := self.Params.GetString("cache")
	mountpoint, _ := self.Params.GetString("mountpoint")
	diskId, _ := self.Params.GetString("disk_id")
	objDisk, err := models.DiskManager.FetchById(diskId)
	if err != nil {
		log.Warningf("failed to fetch disk by id %s error: %v", diskId, err)
		self.OnTaskFail(ctx, guest, nil, errors.New(reason.String()))
		return
	}
	disk := objDisk.(*models.SDisk)
	db.OpsLog.LogEvent(disk, db.ACT_DETACH, reason.String(), self.UserCred)
	disk.SetStatus(self.UserCred, api.DISK_READY, "")
	err = guest.AttachDisk(ctx, disk, self.UserCred, driver, cache, mountpoint)
	if err != nil {
		log.Warningf("recover attach disk %s(%s) for guest %s(%s) error: %v", disk.Name, disk.Id, guest.Name, guest.Id, err)
	}
	self.OnTaskFail(ctx, guest, nil, errors.New(reason.String()))
}

func (self *GuestDetachDiskTask) OnTaskFail(ctx context.Context, guest *models.SGuest, disk *models.SDisk, err error) {
	if disk != nil {
		disk.SetStatus(self.UserCred, api.DISK_READY, "")
	}
	guest.SetStatus(self.UserCred, api.VM_DETACH_DISK_FAILED, err.Error())
	self.SetStageFailed(ctx, err.Error())
	log.Errorf("Guest %s GuestDetachDiskTask failed %s", guest.Id, err.Error())
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_DETACH_DISK, err.Error(), self.UserCred, false)
}

func (self *GuestDetachDiskTask) OnDiskDeleteComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.SetStage("OnSyncstatusComplete", nil)
	guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
}

func (self *GuestDetachDiskTask) OnSyncstatusComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
	logclient.AddActionLogWithStartable(self, obj, logclient.ACT_VM_DETACH_DISK, nil, self.UserCred, true)
}

func (self *GuestDetachDiskTask) OnSyncstatusCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.OnSyncstatusComplete(ctx, obj, data)
}
