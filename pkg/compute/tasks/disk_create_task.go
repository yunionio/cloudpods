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

type DiskCreateTask struct {
	SDiskBaseTask
}

func (self *DiskCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)

	storagecache := disk.GetStorage().GetStoragecache()
	imageId := disk.GetTemplateId()
	if len(imageId) > 0 {
		if len(disk.BackupStorageId) > 0 {
			self.SetStage("OnMasterStorageCacheImageComplete", nil)
		} else {
			self.SetStage("OnStorageCacheImageComplete", nil)
		}
		storagecache.StartImageCacheTask(ctx, self.UserCred, imageId, false, self.GetTaskId())
	} else {
		self.OnStorageCacheImageComplete(ctx, disk, nil)
	}
}

func (self *DiskCreateTask) OnMasterStorageCacheImageComplete(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	storage := models.StorageManager.FetchStorageById(disk.BackupStorageId)
	storagecache := storage.GetStoragecache()
	imageId := disk.GetTemplateId()
	self.SetStage("OnStorageCacheImageComplete", nil)
	storagecache.StartImageCacheTask(ctx, self.UserCred, imageId, false, self.GetTaskId())
}

func (self *DiskCreateTask) OnStorageCacheImageComplete(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	rebuild, _ := self.GetParams().Bool("rebuild")
	snapshot, _ := self.GetParams().GetString("snapshot")
	if rebuild {
		db.OpsLog.LogEvent(disk, db.ACT_DELOCATE, disk.GetShortDesc(), self.GetUserCred())
	}
	storage := disk.GetStorage()
	host := storage.GetMasterHost()
	db.OpsLog.LogEvent(disk, db.ACT_ALLOCATING, disk.GetShortDesc(), self.GetUserCred())
	disk.SetStatus(self.GetUserCred(), models.DISK_STARTALLOC, "")
	if len(disk.BackupStorageId) > 0 {
		self.SetStage("OnMasterStorageCreateDiskComplete", nil)
	} else {
		self.SetStage("OnDiskReady", nil)
	}
	if err := disk.StartAllocate(ctx, host, storage, self.GetTaskId(), self.GetUserCred(), rebuild, snapshot, self); err != nil {
		self.OnStartAllocateFailed(ctx, disk, jsonutils.NewString(err.Error()))
	}
}

func (self *DiskCreateTask) OnMasterStorageCreateDiskComplete(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	rebuild, _ := self.GetParams().Bool("rebuild")
	snapshot, _ := self.GetParams().GetString("snapshot")
	storage := models.StorageManager.FetchStorageById(disk.BackupStorageId)
	host := storage.GetMasterHost()
	db.OpsLog.LogEvent(disk, db.ACT_BACKUP_ALLOCATING, disk.GetShortDesc(), self.GetUserCred())
	disk.SetStatus(self.UserCred, models.DISK_BACKUP_STARTALLOC, "")
	self.SetStage("OnDiskReady", nil)
	if err := disk.StartAllocate(ctx, host, storage, self.GetTaskId(), self.GetUserCred(), rebuild, snapshot, self); err != nil {
		self.OnBackupAllocateFailed(ctx, disk, jsonutils.NewString(fmt.Sprintf("Backup disk alloctate failed: %s", err.Error())))
	}
}

func (self *DiskCreateTask) OnBackupAllocateFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	disk.SetStatus(self.UserCred, models.DISK_BACKUP_ALLOC_FAILED, data.String())
	self.SetStageFailed(ctx, data.String())
}

func (self *DiskCreateTask) OnStartAllocateFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	disk.SetStatus(self.UserCred, models.DISK_ALLOC_FAILED, data.String())
	self.SetStageFailed(ctx, data.String())
}

func (self *DiskCreateTask) OnDiskReady(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	diskSize, _ := data.Int("disk_size")
	if _, err := disk.GetModelManager().TableSpec().Update(disk, func() error {
		disk.DiskSize = int(diskSize)
		disk.DiskFormat, _ = data.GetString("disk_format")
		disk.AccessPath, _ = data.GetString("disk_path")
		return nil
	}); err != nil {
		log.Errorf("update disk info error: %v", err)
	}

	disk.SetStatus(self.UserCred, models.DISK_READY, "")
	self.CleanHostSchedCache(disk)
	db.OpsLog.LogEvent(disk, db.ACT_ALLOCATE, disk.GetShortDesc(), self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *DiskCreateTask) OnDiskReadyFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	disk.SetStatus(self.UserCred, models.DISK_ALLOC_FAILED, data.String())
	self.SetStageFailed(ctx, data.String())
}

type DiskCreateBackupTask struct {
	DiskCreateTask
}

func (self *DiskCreateBackupTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)
	storage := models.StorageManager.FetchStorageById(disk.BackupStorageId)
	storagecache := storage.GetStoragecache()
	imageId := disk.GetTemplateId()
	if len(imageId) > 0 {
		self.SetStage("OnMasterStorageCreateDiskComplete", nil)
		storagecache.StartImageCacheTask(ctx, self.UserCred, imageId, false, self.GetTaskId())
	} else {
		self.OnMasterStorageCreateDiskComplete(ctx, disk, nil)
	}
}

func (self *DiskCreateBackupTask) OnDiskReady(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	bkStorage := models.StorageManager.FetchStorageById(disk.BackupStorageId)
	bkStorage.ClearSchedDescCache()
	disk.SetStatus(self.UserCred, models.DISK_READY, "")
	db.OpsLog.LogEvent(disk, db.ACT_BACKUP_ALLOCATE, disk.GetShortDesc(), self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func init() {
	taskman.RegisterTask(DiskCreateTask{})
	taskman.RegisterTask(DiskCreateBackupTask{})
}
