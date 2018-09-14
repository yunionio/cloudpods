package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/log"
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
	SGuestBaseTask
}

func (self *KVMGuestCreateDiskTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStage("on_kvm_disk_prepared", nil)
	self.OnKvmDiskPrepared(ctx, obj, data)
}

func (self *KVMGuestCreateDiskTask) OnKvmDiskPrepared(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	var diskIndex = 0
	var diskReady = true
	for {
		diskId, err := self.Params.GetString(fmt.Sprintf("disk.%d.id", diskIndex))
		if !diskReady || err != nil {
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
		if disk.Status == models.DISK_INIT {
			snapInfo, err := self.Params.GetString(fmt.Sprintf("disk.%d.snapshot", diskIndex))
			if err != nil {
				snapInfo = ""
			}
			err = disk.StartDiskCreateTask(ctx, self.UserCred, false, snapInfo, self.GetTaskId())
			if err != nil {
				self.SetStageFailed(ctx, err.Error())
				return
			}
			diskReady = false
			break
		}
		diskIndex += 1
	}
	diskIndex = 0
	for {
		diskId, err := self.Params.GetString(fmt.Sprintf("disk.%d.id", diskIndex))
		if !diskReady || err != nil {
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
		if disk.Status != models.DISK_READY {
			diskReady = false
			break
		}
		diskIndex += 1
	}
	if diskReady {
		guest := obj.(*models.SGuest)
		if guest.Status == models.VM_RUNNING {
			self.SetStage("on_config_sync_complete", nil)
			err := guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
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

type AliyunGuestCreateDiskTask struct {
	SGuestBaseTask
}

func (self *AliyunGuestCreateDiskTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStage("on_aliyun_disk_prepared", nil)
	self.OnAliyunDiskPrepared(ctx, obj, data)
}

func (self *AliyunGuestCreateDiskTask) OnAliyunDiskPrepared(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	var diskIndex = 0
	var diskReady = true
	for {
		diskId, err := self.Params.GetString(fmt.Sprintf("disk.%d.id", diskIndex))
		if !diskReady || err != nil {
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
		if disk.Status == models.DISK_INIT {
			snapInfo, err := self.Params.GetString(fmt.Sprintf("disk.%d.snapshot", diskIndex))
			if err != nil {
				snapInfo = ""
			}
			err = disk.StartDiskCreateTask(ctx, self.UserCred, false, snapInfo, self.GetTaskId())
			if err != nil {
				self.SetStageFailed(ctx, err.Error())
				return
			}
			diskReady = false
			break
		}
		diskIndex += 1
	}
	diskIndex = 0
	guest := obj.(*models.SGuest)

	for {
		diskId, err := self.Params.GetString(fmt.Sprintf("disk.%d.id", diskIndex))
		if !diskReady || err != nil {
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
		if disk.Status != models.DISK_READY {
			diskReady = false
			break
		}

		ihost, err := guest.GetHost().GetIHost()
		if err != nil {
			self.SetStageFailed(ctx, "Host not found")
			return
		}

		iVM, e := ihost.GetIVMById(guest.GetExternalId())
		if e != nil {
			self.SetStageFailed(ctx, "Aliyun VM not found")
			return
		}

		err = iVM.AttachDisk(disk.GetExternalId())
		if err != nil {
			log.Debugf("Attach Disk %s to guest fail: %s", diskId, err)
			self.SetStageFailed(ctx, "Attach Disk to guest fail")
			return
		}
		diskIndex += 1
	}

	if diskReady {
		if guest.Status == models.VM_RUNNING {
			self.SetStage("on_config_sync_complete", nil)
			err := guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
			if err != nil {
				self.SetStageFailed(ctx, err.Error())
			}
		} else {
			self.SetStageComplete(ctx, nil)
		}
	}
}

func (self *AliyunGuestCreateDiskTask) OnConfigSyncComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *AliyunGuestCreateDiskTask) AttachAliyunDisks(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

type ManagedGuestCreateDiskTask struct {
	SGuestBaseTask
}

func (self *ManagedGuestCreateDiskTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStage("on_managed_disk_prepared", nil)
	self.OnManagedDiskPrepared(ctx, obj, data)
}

func (self *ManagedGuestCreateDiskTask) OnManagedDiskPrepared(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	var diskIndex = 0
	var diskReady = true
	for {
		diskId, err := self.Params.GetString(fmt.Sprintf("disk.%d.id", diskIndex))
		if !diskReady || err != nil {
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
		if disk.Status == models.DISK_INIT {
			snapInfo, err := self.Params.GetString(fmt.Sprintf("disk.%d.snapshot", diskIndex))
			if err != nil {
				snapInfo = ""
			}
			err = disk.StartDiskCreateTask(ctx, self.UserCred, false, snapInfo, self.GetTaskId())
			if err != nil {
				self.SetStageFailed(ctx, err.Error())
				return
			}
			diskReady = false
			break
		}
		diskIndex += 1
	}
	diskIndex = 0
	guest := obj.(*models.SGuest)

	for {
		diskId, err := self.Params.GetString(fmt.Sprintf("disk.%d.id", diskIndex))
		if !diskReady || err != nil {
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
		if disk.Status != models.DISK_READY {
			diskReady = false
			break
		}

		ihost, err := guest.GetHost().GetIHost()
		if err != nil {
			self.SetStageFailed(ctx, "Host not found")
			return
		}

		iVM, e := ihost.GetIVMById(guest.GetExternalId())
		if e != nil {
			self.SetStageFailed(ctx, "Aliyun VM not found")
			return
		}

		err = iVM.AttachDisk(disk.GetExternalId())
		if err != nil {
			log.Debugf("Attach Disk %s to guest fail: %s", diskId, err)
			self.SetStageFailed(ctx, "Attach Disk to guest fail")
			return
		}
		diskIndex += 1
	}

	if diskReady {
		if guest.Status == models.VM_RUNNING {
			self.SetStage("on_config_sync_complete", nil)
			err := guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
			if err != nil {
				self.SetStageFailed(ctx, err.Error())
			}
		} else {
			self.SetStageComplete(ctx, nil)
		}
	}
}

func (self *ManagedGuestCreateDiskTask) OnConfigSyncComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *ManagedGuestCreateDiskTask) AttachManagedDisks(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func init() {
	taskman.RegisterTask(GuestCreateDiskTask{})
	taskman.RegisterTask(KVMGuestCreateDiskTask{})
	taskman.RegisterTask(AliyunGuestCreateDiskTask{})
	taskman.RegisterTask(ManagedGuestCreateDiskTask{})
}
