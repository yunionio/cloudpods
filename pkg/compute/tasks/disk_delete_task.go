package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
)

type DiskDeleteTask struct {
	SDiskBaseTask
}

func init() {
	taskman.RegisterTask(DiskDeleteTask{})
}

func (self *DiskDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)

	if disk.GetGuestDiskCount() > 0 {
		reason := "Disk has been attached to server"
		self.SetStageFailed(ctx, reason)
		db.OpsLog.LogEvent(disk, db.ACT_DELOCATE_FAIL, reason, self.UserCred)
		return
	}
	if options.Options.EnablePendingDelete && !disk.PendingDeleted && !jsonutils.QueryBoolean(self.Params, "purge", false) && !jsonutils.QueryBoolean(self.Params, "override_pending_delete", false) {
		self.startPendingDeleteDisk(ctx, disk)
	} else {
		self.startDeleteDisk(ctx, disk)
	}
}

func (self *DiskDeleteTask) startDeleteDisk(ctx context.Context, disk *models.SDisk) {
	db.OpsLog.LogEvent(disk, db.ACT_DELOCATING, disk.GetShortDesc(), self.UserCred)
	if disk.Status == models.DISK_INIT {
		self.OnGuestDiskDeleteSucc(ctx, disk, nil)
		return
	}
	storage := disk.GetStorage()
	host := storage.GetMasterHost()
	isPurge := false
	if (host == nil || !host.Enabled) && jsonutils.QueryBoolean(self.Params, "purge", false) {
		isPurge = true
	}
	disk.SetStatus(self.UserCred, models.DISK_DEALLOC, "")
	if isPurge {
		self.OnGuestDiskDeleteSucc(ctx, disk, nil)
	} else {
		if len(disk.BackupStorageId) > 0 {
			self.SetStage("OnMasterStorageDeleteDiskSucc", nil)
		} else {
			self.SetStage("OnGuestDiskDeleteSucc", nil)
		}
		if host == nil {
			self.OnGuestDiskDeleteFailed(ctx, disk, fmt.Errorf("fail to find master host"))
		} else if err := host.GetHostDriver().RequestDeallocateDiskOnHost(host, storage, disk, self); err != nil {
			self.OnGuestDiskDeleteFailed(ctx, disk, err)
		}
	}
}

func (self *DiskDeleteTask) OnMasterStorageDeleteDiskSucc(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	self.SetStage("OnGuestDiskDeleteSucc", nil)
	storage := models.StorageManager.FetchStorageById(disk.BackupStorageId)
	host := storage.GetMasterHost()
	if host == nil {
		self.OnGuestDiskDeleteFailed(ctx, disk, fmt.Errorf("backup storage %s fail to find master host", disk.BackupStorageId))
	} else if err := host.GetHostDriver().RequestDeallocateDiskOnHost(host, storage, disk, self); err != nil {
		self.OnGuestDiskDeleteFailed(ctx, disk, err)
	}
}

func (self *DiskDeleteTask) startPendingDeleteDisk(ctx context.Context, disk *models.SDisk) {
	disk.DoPendingDelete(ctx, self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *DiskDeleteTask) OnGuestDiskDeleteSucc(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	if obj == nil {
		self.SetStageComplete(ctx, nil)
		return
	}
	disk := obj.(*models.SDisk)
	self.CleanHostSchedCache(disk)
	db.OpsLog.LogEvent(disk, db.ACT_DELOCATE, disk.GetShortDesc(), self.UserCred)
	disk.RealDelete(ctx, self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *DiskDeleteTask) OnGuestDiskDeleteFailed(ctx context.Context, disk *models.SDisk, resion error) {
	disk.SetStatus(self.GetUserCred(), models.DISK_DEALLOC_FAILED, resion.Error())
	self.SetStageFailed(ctx, resion.Error())
	db.OpsLog.LogEvent(disk, db.ACT_DELOCATE_FAIL, disk.GetShortDesc(), self.GetUserCred())
}
