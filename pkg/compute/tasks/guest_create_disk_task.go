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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type GuestCreateDiskTask struct {
	SGuestBaseTask
}

func (self *GuestCreateDiskTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStage("on_disk_prepared", nil)
	guest := obj.(*models.SGuest)
	err := guest.GetDriver().DoGuestCreateDisksTask(ctx, guest, self)
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
	}
}

func (self *GuestCreateDiskTask) OnDiskPrepared(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestCreateDiskTask) OnDiskPreparedFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data.String())
}

/* --------------------------------------------- */
/* -----------KVMGuestCreateDiskTask------------ */
/* --------------------------------------------- */

type KVMGuestCreateDiskTask struct {
	SGuestCreateDiskBaseTask
}

func (self *KVMGuestCreateDiskTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStage("on_kvm_disk_prepared", nil)
	self.OnKvmDiskPrepared(ctx, obj, data)
}

func (self *KVMGuestCreateDiskTask) OnKvmDiskPrepared(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	var diskReady = true
	disks, err := self.GetInputDisks()
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
		return
	}

	for _, d := range disks {
		diskId := d.DiskId
		if !diskReady {
			break
		}
		iDisk, err := models.DiskManager.FetchById(diskId)
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
			return
		}
		if iDisk == nil {
			self.SetStageFailed(ctx, "Disk not found")
			return
		}
		disk := iDisk.(*models.SDisk)
		if disk.Status == api.DISK_INIT {
			snapshotId := d.SnapshotId
			err = disk.StartDiskCreateTask(ctx, self.UserCred, false, snapshotId, self.GetTaskId())
			if err != nil {
				self.SetStageFailed(ctx, err.Error())
				return
			}
			diskReady = false
			break
		}
	}
	for _, d := range disks {
		if !diskReady {
			break
		}
		diskId := d.DiskId
		iDisk, err := models.DiskManager.FetchById(diskId)
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
			return
		}
		if iDisk == nil {
			self.SetStageFailed(ctx, "Disk not found")
			return
		}
		disk := iDisk.(*models.SDisk)
		if disk.Status != api.DISK_READY {
			diskReady = false
			break
		}
		err = self.attachDisk(ctx, disk, d.Driver, d.Cache, d.Mountpoint)
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
			return
		}
	}
	if diskReady {
		guest := obj.(*models.SGuest)
		if guest.Status == api.VM_RUNNING {
			self.SetStage("on_config_sync_complete", nil)
			err := guest.StartSyncTask(ctx, self.UserCred, false, self.GetTaskId())
			if err != nil {
				self.SetStageFailed(ctx, err.Error())
			}
		} else {
			self.SetStageComplete(ctx, nil)
		}
	}
}

func (self *KVMGuestCreateDiskTask) OnConfigSyncComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

type SGuestCreateDiskBaseTask struct {
	SGuestBaseTask
}

func (self *SGuestCreateDiskBaseTask) GetInputDisks() ([]api.DiskConfig, error) {
	disks := make([]api.DiskConfig, 0)
	err := self.GetParams().Unmarshal(&disks, "disks")
	return disks, err
}

func (self *SGuestCreateDiskBaseTask) attachDisk(ctx context.Context, disk *models.SDisk, driver, cache, mountpoint string) error {
	guest := self.getGuest()
	attached, err := guest.IsAttach2Disk(disk)
	if err != nil {
		return errors.Wrapf(err, "IsAttach2Disk")
	}
	if attached {
		return nil
	}
	err = guest.AttachDisk(ctx, disk, self.UserCred, driver, cache, mountpoint)
	if err != nil {
		return errors.Wrapf(err, "AttachDisk")
	}
	return nil
}

type ManagedGuestCreateDiskTask struct {
	SGuestCreateDiskBaseTask
}

func (self *ManagedGuestCreateDiskTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStage("on_managed_disk_prepared", nil)
	self.OnManagedDiskPrepared(ctx, obj, data)
}

func (self *ManagedGuestCreateDiskTask) OnManagedDiskPrepared(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disks, err := self.GetInputDisks()
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
		return
	}

	for _, d := range disks {
		diskId := d.DiskId
		iDisk, err := models.DiskManager.FetchById(diskId)
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
			return
		}
		disk := iDisk.(*models.SDisk)
		if disk.Status == api.DISK_INIT {
			snapshot := d.SnapshotId
			err = disk.StartDiskCreateTask(ctx, self.UserCred, false, snapshot, self.GetTaskId())
			if err != nil {
				self.SetStageFailed(ctx, err.Error())
				return
			}
			return
		}
	}

	guest := obj.(*models.SGuest)

	for _, d := range disks {
		diskId := d.DiskId
		iDisk, err := models.DiskManager.FetchById(diskId)
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
			return
		}
		disk := iDisk.(*models.SDisk)
		if disk.Status != api.DISK_READY {
			self.SetStageFailed(ctx, fmt.Sprintf("disk %s is not ready", disk.Id))
			return
		}

		iVM, e := guest.GetIVM()
		if e != nil {
			self.SetStageFailed(ctx, "iVM not found")
			return
		}

		err = iVM.AttachDisk(ctx, disk.GetExternalId())
		if err != nil {
			log.Debugf("Attach Disk %s to guest fail: %s", diskId, err)
			self.SetStageFailed(ctx, fmt.Sprintf("Attach iDisk to guest fail error: %v", err))
			return
		}
		err = self.attachDisk(ctx, disk, d.Driver, d.Cache, d.Mountpoint)
		if err != nil {
			log.Debugf("Attach Disk %s to guest fail: %s", diskId, err)
			self.SetStageFailed(ctx, fmt.Sprintf("Attach Disk to guest fail error: %v", err))
			return
		}
		time.Sleep(time.Second * 5)
	}

	self.SetStageComplete(ctx, nil)
}

/*
func (self *ManagedGuestCreateDiskTask) OnConfigSyncComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}
*/

type ESXiGuestCreateDiskTask struct {
	SGuestCreateDiskBaseTask
}

func (self *ESXiGuestCreateDiskTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	host := guest.GetHost()
	if host == nil {
		self.SetStageFailed(ctx, "no valid host")
		return
	}

	disks, err := self.GetInputDisks()
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
		return
	}
	for _, d := range disks {
		diskId := d.DiskId
		iDisk, err := models.DiskManager.FetchById(diskId)
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
			return
		}
		disk := iDisk.(*models.SDisk)
		if disk.Status != api.DISK_INIT {
			self.SetStageFailed(ctx, fmt.Sprintf("Disk %s already created??", diskId))
			return
		}
		ivm, err := guest.GetIVM()
		if err != nil {
			self.SetStageFailed(ctx, fmt.Sprintf("fail to find iVM for %s, error: %v", guest.GetName(), err))
			return
		}
		if len(d.Driver) == 0 {
			osProf := guest.GetOSProfile()
			d.Driver = osProf.DiskDriver
		}
		err = ivm.CreateDisk(ctx, disk.DiskSize, disk.Id, d.Driver)
		if err != nil {
			self.SetStageFailed(ctx, fmt.Sprintf("ivm.CreateDisk fail %s, error: %v", guest.GetName(), err))
			return
		}
		idisks, err := ivm.GetIDisks()
		if err != nil {
			self.SetStageFailed(ctx, fmt.Sprintf("ivm.GetIDisks fail %s", err))
			return
		}

		err = self.attachDisk(ctx, disk, d.Driver, d.Cache, d.Mountpoint)
		if err != nil {
			self.SetStageFailed(ctx, fmt.Sprintf("self.attachDisk fail %v", err))
			return
		}

		log.Debugf("diskcount after create: %d", len(idisks))

		vdisk := idisks[len(idisks)-1]

		_, err = db.Update(disk, func() error {
			disk.DiskSize = vdisk.GetDiskSizeMB()
			disk.AccessPath = vdisk.GetAccessPath()
			disk.ExternalId = vdisk.GetGlobalId()
			return nil
		})
		if err != nil {
			self.SetStageFailed(ctx, fmt.Sprintf("disk.GetModelManager().TableSpec().Update fail %s", err))
			return
		}

		disk.SetStatus(self.UserCred, api.DISK_READY, "create disk success")
		disk.GetStorage().ClearSchedDescCache()
		db.OpsLog.LogEvent(disk, db.ACT_ALLOCATE, disk.GetShortDesc(ctx), self.UserCred)
		db.OpsLog.LogAttachEvent(ctx, guest, disk, self.UserCred, disk.GetShortDesc(ctx))
	}

	self.SetStageComplete(ctx, nil)
}

type GuestCreateBackupDisksTask struct {
	SGuestBaseTask
}

func (self *GuestCreateBackupDisksTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.CreateBackups(ctx, guest, nil)
}

func (self *GuestCreateBackupDisksTask) CreateBackups(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	body := jsonutils.NewDict()
	var diskIndex int64 = 0
	if self.Params.Contains("disk_index") {
		diskIndex, _ = self.Params.Int("disk_index")
	}
	body.Set("disk_index", jsonutils.NewInt(diskIndex+1))
	self.SetStage("CreateBackups", body)

	guestDisks := guest.GetDisks()
	if int(diskIndex) == len(guestDisks) {
		self.SetStageComplete(ctx, nil)
	} else {
		err := guestDisks[diskIndex].GetDisk().StartCreateBackupTask(ctx, self.UserCred, self.GetTaskId())
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
		}
	}
}

func init() {
	taskman.RegisterTask(GuestCreateBackupDisksTask{})
	taskman.RegisterTask(GuestCreateDiskTask{})
	taskman.RegisterTask(KVMGuestCreateDiskTask{})
	taskman.RegisterTask(ManagedGuestCreateDiskTask{})
	taskman.RegisterTask(ESXiGuestCreateDiskTask{})
}
