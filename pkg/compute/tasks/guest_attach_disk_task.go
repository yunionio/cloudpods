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

type GuestAttachDiskTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestAttachDiskTask{})
}

func (self *GuestAttachDiskTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	diskId, _ := self.Params.GetString("disk_id")
	objDisk, err := models.DiskManager.FetchById(diskId)
	if err != nil {
		self.OnTaskFail(ctx, guest, nil, jsonutils.NewString(err.Error()))
		return
	}
	disk := objDisk.(*models.SDisk)
	if disk == nil {
		self.OnTaskFail(ctx, guest, nil, jsonutils.NewString(fmt.Sprintf("Connot find disk %s", diskId)))
		return
	}

	driver, _ := self.Params.GetString("driver")
	cache, _ := self.Params.GetString("cache")
	mountpoint, _ := self.Params.GetString("mountpoint")
	var bootIndex *int8
	if self.Params.Contains("boot_index") {
		bd, _ := self.Params.Int("boot_index")
		bd8 := int8(bd)
		bootIndex = &bd8
	}

	err = guest.AttachDisk(ctx, disk, self.UserCred, driver, cache, mountpoint, bootIndex)
	if err != nil {
		self.OnTaskFail(ctx, guest, nil, jsonutils.NewString(err.Error()))
		return
	}
	disk.SetStatus(ctx, self.UserCred, api.DISK_ATTACHING, "Disk attach")
	self.SetStage("OnSyncConfigComplete", nil)
	drv, err := guest.GetDriver()
	if err != nil {
		self.OnTaskFail(ctx, guest, disk, jsonutils.NewString("GetDriver"))
		return
	}
	drv.RequestAttachDisk(ctx, guest, disk, self)
}

func (self *GuestAttachDiskTask) OnSyncConfigComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	diskId, _ := self.Params.GetString("disk_id")
	objDisk, err := models.DiskManager.FetchById(diskId)
	if err != nil {
		self.OnTaskFail(ctx, guest, nil, jsonutils.NewString(err.Error()))
		return
	}
	disk := objDisk.(*models.SDisk)
	if disk == nil {
		self.OnTaskFail(ctx, guest, nil, jsonutils.NewString(fmt.Sprintf("Connot find disk %s", diskId)))
		return
	}
	disk.SetDiskReady(ctx, self.UserCred, "")
	self.SetStageComplete(ctx, nil)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_ATTACH_DISK, nil, self.UserCred, true)
}

func (self *GuestAttachDiskTask) OnSyncConfigCompleteFailed(ctx context.Context, obj db.IStandaloneModel, reason jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	diskId, _ := self.Params.GetString("disk_id")
	objDisk, err := models.DiskManager.FetchById(diskId)
	if err != nil {
		self.OnTaskFail(ctx, guest, nil, jsonutils.NewString(err.Error()))
		return
	}
	disk := objDisk.(*models.SDisk)
	db.OpsLog.LogEvent(disk, db.ACT_ATTACH, reason.String(), self.UserCred)
	disk.SetStatus(ctx, self.UserCred, api.DISK_READY, "")
	guest.DetachDisk(ctx, disk, self.UserCred)
	self.OnTaskFail(ctx, guest, disk, reason)
}

func (self *GuestAttachDiskTask) OnTaskFail(ctx context.Context, guest *models.SGuest, disk *models.SDisk, err jsonutils.JSONObject) {
	if disk != nil {
		disk.SetStatus(ctx, self.UserCred, api.DISK_READY, "")
	}
	guest.SetStatus(ctx, self.UserCred, api.VM_ATTACH_DISK_FAILED, err.String())
	self.SetStageFailed(ctx, err)
	log.Errorf("Guest %s GuestAttachDiskTask failed %s", guest.Name, err.String())
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_ATTACH_DISK, err, self.UserCred, false)
}
