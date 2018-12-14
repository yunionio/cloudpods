package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
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
	if disk.Status == models.DISK_INIT {
		self.OnSyncConfigComplete(ctx, guest, nil)
		return
	}
	disk.SetStatus(self.UserCred, models.DISK_DETACHING, "Disk detach")

	host := guest.GetHost()
	purge := false
	if host != nil && host.Status == models.HOST_DISABLED && jsonutils.QueryBoolean(self.Params, "purge", false) {
		purge = true
	}
	detachStatus, err := guest.GetDriver().GetDetachDiskStatus()
	if err != nil {
		self.OnTaskFail(ctx, guest, disk, err)
		return
	}
	if utils.IsInStringArray(guest.Status, detachStatus) && !purge {
		self.SetStage("on_sync_config_complete", nil)
		guest.GetDriver().RequestDetachDisk(ctx, guest, self)
	} else {
		self.OnSyncConfigComplete(ctx, guest, nil)
	}
}

func (self *GuestDetachDiskTask) OnSyncConfigComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
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
	disk.SetDiskReady(ctx, self.UserCred, "")
	keepDisk := jsonutils.QueryBoolean(self.Params, "keep_disk", true)
	host := guest.GetHost()
	purge := false
	if host != nil && host.Status == models.HOST_DISABLED && jsonutils.QueryBoolean(self.Params, "purge", false) {
		purge = true
	}
	if disk.Status == models.DISK_INIT {
		db.OpsLog.LogEvent(disk, db.ACT_DELETE, "", self.UserCred)
		disk.RealDelete(ctx, self.UserCred)
		self.SetStageComplete(ctx, nil)
	} else if (disk.Status != models.DISK_READY || !keepDisk) && disk.GetGuestDiskCount() == 0 && disk.AutoDelete {
		self.SetStage("on_disk_delete_complete", nil)
		db.OpsLog.LogEvent(disk, db.ACT_DELETE, "", self.UserCred)
		err := guest.GetDriver().RequestDeleteDetachedDisk(ctx, disk, self, purge)
		if err != nil {
			self.OnTaskFail(ctx, guest, disk, err)
		}
	} else {
		self.SetStageComplete(ctx, nil)
	}
}

func (self *GuestDetachDiskTask) OnSyncConfigCompleteFailed(ctx context.Context, obj db.IStandaloneModel, resion jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	driver, _ := self.Params.GetString("driver")
	cache, _ := self.Params.GetString("cache")
	mountpoint, _ := self.Params.GetString("mountpoint")
	diskId, _ := self.Params.GetString("disk_id")
	objDisk, err := models.DiskManager.FetchById(diskId)
	if err != nil {
		self.OnTaskFail(ctx, guest, nil, err)
		return
	}
	disk := objDisk.(*models.SDisk)
	db.OpsLog.LogEvent(disk, db.ACT_DETACH, resion.String(), self.UserCred)
	disk.SetDiskReady(ctx, self.UserCred, "")
	err = guest.AttachDisk(ctx, disk, self.UserCred, driver, cache, mountpoint)
	if err != nil {
		self.OnTaskFail(ctx, guest, disk, err)
		return
	}
}

func (self *GuestDetachDiskTask) OnTaskFail(ctx context.Context, guest *models.SGuest, disk *models.SDisk, err error) {
	if disk != nil {
		disk.SetDiskReady(ctx, self.UserCred, "")
	}
	guest.SetStatus(self.UserCred, models.VM_DETACH_DISK_FAILED, err.Error())
	self.SetStageFailed(ctx, err.Error())
	log.Errorf("Guest %s GuestDetachDiskTask failed %s", guest.Id, err.Error())
}

func (self *GuestDetachDiskTask) OnDiskDeleteComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}
