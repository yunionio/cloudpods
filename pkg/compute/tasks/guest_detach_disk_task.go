package tasks

import (
	"context"
	"fmt"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/compute/models"
	"github.com/yunionio/pkg/utils"
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
		self.OnTaskFail(ctx, guest, err)
		return
	}
	disk := objDisk.(*models.SDisk)
	if disk == nil {
		self.OnTaskFail(ctx, guest, fmt.Errorf("Connot find disk %s", diskId))
		return
	}

	guest.DetachDisk(ctx, disk, self.UserCred)
	if disk.Status == models.DISK_INIT {
		self.OnSyncConfigComplete(ctx, guest, nil)
		return
	}
	host := guest.GetHost()
	purge := false
	if host != nil && host.Status == models.HOST_DISABLED && jsonutils.QueryBoolean(self.Params, "purge", false) {
		purge = true
	}
	detachStatus, err := guest.GetDriver().GetDetachDiskStatus()
	if err != nil {
		self.OnTaskFail(ctx, guest, err)
		return
	}
	if utils.IsInStringArray(guest.Status, detachStatus) && !purge {
		self.SetStage("on_sync_config_complete", nil)
		guest.GetDriver().RequestDetachDisk(ctx, guest, self)
		disk.SetStatus(self.UserCred, models.DISK_READY, "Disk detach")
	} else {
		self.OnSyncConfigComplete(ctx, guest, nil)
	}
}

func (self *GuestDetachDiskTask) OnSyncConfigComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	diskId, _ := self.Params.GetString("disk_id")
	objDisk, err := models.DiskManager.FetchById(diskId)
	if err != nil {
		self.OnTaskFail(ctx, guest, err)
		return
	}
	disk := objDisk.(*models.SDisk)
	if disk == nil {
		self.OnTaskFail(ctx, guest, fmt.Errorf("Connot find disk %s", diskId))
		return
	}
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
	} else if (disk.Status == models.DISK_READY || !keepDisk) && disk.GetGuestDiskCount() == 0 && disk.AutoDelete {
		self.SetStage("on_disk_delete_complete", nil)
		db.OpsLog.LogEvent(disk, db.ACT_DELETE, "", self.UserCred)
		err := guest.GetDriver().RequestDeleteDetachedDisk(ctx, disk, self, purge)
		if err != nil {
			self.OnTaskFail(ctx, guest, err)
			return
		}
	} else {
		self.SetStageComplete(ctx, nil)
	}
}

func (self *GuestDetachDiskTask) OnTaskFail(ctx context.Context, guest *models.SGuest, err error) {
	self.SetStageFailed(ctx, err.Error())
	log.Errorf("Guest %s GuestDetachDiskTask failed %s", guest.Id, err.Error())
}

func (self *GuestDetachDiskTask) OnDiskDeleteComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}
