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
	db.OpsLog.LogEvent(disk, db.ACT_DELOCATING, disk.GetShortDesc(ctx), self.UserCred)
	if disk.Status == models.DISK_INIT {
		self.OnGuestDiskDeleteComplete(ctx, disk, nil)
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
		self.OnGuestDiskDeleteComplete(ctx, disk, nil)
	} else {
		if len(disk.BackupStorageId) > 0 {
			self.SetStage("OnMasterStorageDeleteDiskComplete", nil)
		} else {
			self.SetStage("OnGuestDiskDeleteComplete", nil)
		}
		if host == nil {
			self.OnGuestDiskDeleteCompleteFailed(ctx, disk, jsonutils.NewString("fail to find master host"))
		} else if err := host.GetHostDriver().RequestDeallocateDiskOnHost(ctx, host, storage, disk, self); err != nil {
			self.OnGuestDiskDeleteCompleteFailed(ctx, disk, jsonutils.NewString(err.Error()))
		}
	}
}

func (self *DiskDeleteTask) OnMasterStorageDeleteDiskComplete(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	self.SetStage("OnGuestDiskDeleteComplete", nil)
	storage := models.StorageManager.FetchStorageById(disk.BackupStorageId)
	host := storage.GetMasterHost()
	if host == nil {
		self.OnGuestDiskDeleteCompleteFailed(ctx, disk, jsonutils.NewString(fmt.Sprintf("backup storage %s fail to find master host", disk.BackupStorageId)))
	} else if err := host.GetHostDriver().RequestDeallocateDiskOnHost(ctx, host, storage, disk, self); err != nil {
		self.OnGuestDiskDeleteCompleteFailed(ctx, disk, jsonutils.NewString(err.Error()))
	}
}

func (self *DiskDeleteTask) OnMasterStorageDeleteDiskCompleteFailed(ctx context.Context, disk *models.SDisk, resion jsonutils.JSONObject) {
	self.OnGuestDiskDeleteCompleteFailed(ctx, disk, resion)
}

func (self *DiskDeleteTask) startPendingDeleteDisk(ctx context.Context, disk *models.SDisk) {
	disk.DoPendingDelete(ctx, self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *DiskDeleteTask) OnGuestDiskDeleteComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	if obj == nil {
		self.SetStageComplete(ctx, nil)
		return
	}
	disk := obj.(*models.SDisk)
	self.CleanHostSchedCache(disk)
	db.OpsLog.LogEvent(disk, db.ACT_DELOCATE, disk.GetShortDesc(ctx), self.UserCred)
	if len(disk.SnapshotId) > 0 && disk.GetMetadata("merge_snapshot", nil) == "true" {
		models.SnapshotManager.AddRefCount(disk.SnapshotId, -1)
	}
	disk.RealDelete(ctx, self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *DiskDeleteTask) OnGuestDiskDeleteCompleteFailed(ctx context.Context, disk *models.SDisk, resion jsonutils.JSONObject) {
	disk.SetStatus(self.GetUserCred(), models.DISK_DEALLOC_FAILED, resion.String())
	self.SetStageFailed(ctx, resion.String())
	db.OpsLog.LogEvent(disk, db.ACT_DELOCATE_FAIL, disk.GetShortDesc(ctx), self.GetUserCred())
}
