package tasks

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

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
		if disk.Status == models.DISK_INIT {
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
		if disk.Status != models.DISK_READY {
			diskReady = false
			break
		}
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

type SGuestCreateDiskBaseTask struct {
	SGuestBaseTask
}

func (self *SGuestCreateDiskBaseTask) GetInputDisks() ([]api.DiskConfig, error) {
	disks := make([]api.DiskConfig, 0)
	err := self.GetParams().Unmarshal(&disks, "disks")
	return disks, err
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
		if disk.Status == models.DISK_INIT {
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

		disk.SetStatus(self.UserCred, models.DISK_READY, "create disk success")
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
