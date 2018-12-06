package tasks

import (
	"context"
	"fmt"
	"time"

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
			snapshotId, _ := self.Params.GetString(fmt.Sprintf("disk.%d.snapshot", diskIndex))
			log.Errorln("XXXXXXXXXXXXXX", snapshotId)
			err = disk.StartDiskCreateTask(ctx, self.UserCred, false, snapshotId, self.GetTaskId())
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

type ManagedGuestCreateDiskTask struct {
	SGuestBaseTask
}

func (self *ManagedGuestCreateDiskTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStage("on_managed_disk_prepared", nil)
	self.OnManagedDiskPrepared(ctx, obj, data)
}

func (self *ManagedGuestCreateDiskTask) OnManagedDiskPrepared(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	var diskIndex = 0

	for self.Params.Contains(fmt.Sprintf("disk.%d.id", diskIndex)) {
		diskId, err := self.Params.GetString(fmt.Sprintf("disk.%d.id", diskIndex))
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
			return
		}
		iDisk, err := models.DiskManager.FetchById(diskId)
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
			return
		}
		disk := iDisk.(*models.SDisk)
		if disk.Status == models.DISK_INIT {
			snapInfo, _ := self.Params.GetString(fmt.Sprintf("disk.%d.snapshot", diskIndex))
			err = disk.StartDiskCreateTask(ctx, self.UserCred, false, snapInfo, self.GetTaskId())
			if err != nil {
				self.SetStageFailed(ctx, err.Error())
				return
			}
			return
		}
		diskIndex += 1
	}

	diskIndex = 0
	guest := obj.(*models.SGuest)

	for self.Params.Contains(fmt.Sprintf("disk.%d.id", diskIndex)) {
		diskId, err := self.Params.GetString(fmt.Sprintf("disk.%d.id", diskIndex))
		if err != nil {
			return
		}
		iDisk, err := models.DiskManager.FetchById(diskId)
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
			return
		}
		disk := iDisk.(*models.SDisk)
		if disk.Status != models.DISK_READY {
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
			self.SetStageFailed(ctx, "Attach Disk to guest fail")
			return
		}
		time.Sleep(time.Second * 5)
		diskIndex += 1
	}

	self.SetStageComplete(ctx, nil)
}

/*
func (self *ManagedGuestCreateDiskTask) OnConfigSyncComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}
*/

type ESXiGuestCreateDiskTask struct {
	SGuestBaseTask
}

func (self *ESXiGuestCreateDiskTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	host := guest.GetHost()
	if host == nil {
		self.SetStageFailed(ctx, "no valid host")
		return
	}

	diskIndex := 0
	for {
		diskKey := fmt.Sprintf("disk.%d.id", diskIndex)
		if !self.Params.Contains(diskKey) {
			break
		}
		diskId, _ := self.Params.GetString(diskKey)
		diskIndex += 1
		guestDisk := guest.GetGuestDisk(diskId)
		if guestDisk == nil {
			self.SetStageFailed(ctx, "fail to find guestdisk")
			return
		}
		disk := guestDisk.GetDisk()
		if disk == nil {
			self.SetStageFailed(ctx, fmt.Sprintf("Disk %s not found", diskId))
			return
		}
		if disk.Status != models.DISK_INIT {
			self.SetStageFailed(ctx, fmt.Sprintf("Disk %s already created??", diskId))
			return
		}
		ivm, err := guest.GetIVM()
		if err != nil {
			self.SetStageFailed(ctx, fmt.Sprintf("fail to find iVM for %s", guest.GetName()))
			return
		}
		err = ivm.CreateDisk(ctx, disk.DiskSize, disk.Id, guestDisk.Driver)
		if err != nil {
			self.SetStageFailed(ctx, fmt.Sprintf("ivm.CreateDisk fail %s", guest.GetName()))
			return
		}
		idisks, err := ivm.GetIDisks()
		if err != nil {
			self.SetStageFailed(ctx, fmt.Sprintf("ivm.GetIDisks fail %s", err))
			return
		}

		log.Debugf("diskcount after create: %d", len(idisks))

		vdisk := idisks[len(idisks)-1]

		_, err = disk.GetModelManager().TableSpec().Update(disk, func() error {
			disk.DiskSize = vdisk.GetDiskSizeMB()
			disk.AccessPath = vdisk.GetAccessPath()
			disk.ExternalId = vdisk.GetGlobalId()
			return nil
		})
		if err != nil {
			self.SetStageFailed(ctx, fmt.Sprintf("disk.GetModelManager().TableSpec().Update fail %s", err))
			return
		}

		disk.SetStatus(self.UserCred, models.DISK_READY, "create disk success")
		disk.GetStorage().ClearSchedDescCache()
		db.OpsLog.LogEvent(disk, db.ACT_ALLOCATE, disk.GetShortDesc(), self.UserCred)
		db.OpsLog.LogAttachEvent(guest, disk, self.UserCred, disk.GetShortDesc())
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
		err := guestDisks[diskIndex].GetDisk().StratCreateBackupTask(ctx, self.UserCred, self.GetTaskId())
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
