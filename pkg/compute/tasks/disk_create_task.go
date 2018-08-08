package tasks

import (
	"context"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/compute/models"
)

type DiskCreateTask struct {
	SDiskBaseTask
}

func init() {
	taskman.RegisterTask(DiskCreateTask{})
}

func (self *DiskCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)

	storagecache := disk.GetStorage().GetStoragecache()
	imageId := disk.GetTemplateId()
	if len(imageId) > 0 {
		self.SetStage("on_storage_cache_image_complete", nil)
		storagecache.StartImageCacheTask(ctx, self.UserCred, imageId, false, self.GetTaskId())
	} else {
		self.OnStorageCacheImageComplete(ctx, disk)
	}
}

func (self *DiskCreateTask) OnStorageCacheImageComplete(ctx context.Context, disk *models.SDisk) {
	rebuild, _ := self.GetParams().Bool("rebuild")
	snapshot, _ := self.GetParams().GetString("snapshot")
	if rebuild {
		db.OpsLog.LogEvent(disk, db.ACT_DELOCATE, disk.GetShortDesc(), self.GetUserCred())
	}
	storage := disk.GetStorage()
	host := storage.GetMasterHost()
	db.OpsLog.LogEvent(disk, db.ACT_ALLOCATE, disk.GetShortDesc(), self.GetUserCred())
	disk.SetStatus(self.GetUserCred(), models.DISK_STARTALLOC, "")
	self.SetStage("on_disk_ready", nil)
	if err := disk.StartAllocate(host, storage, self.GetTaskId(), self.GetUserCred(), rebuild, snapshot, self); err != nil {
		self.OnStartAllocateFailed(ctx, disk, err)
	}
}

func (self *DiskCreateTask) OnStartAllocateFailed(ctx context.Context, disk *models.SDisk, resion error) {
	disk.SetStatus(self.UserCred, models.DISK_ALLOC_FAILED, resion.Error())
	self.SetStageFailed(ctx, resion.Error())
}

func (self *DiskCreateTask) OnDiskReady(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	diskSize, _ := data.Int("disk_size")
	disk.DiskSize = int(diskSize)
	disk.DiskFormat, _ = data.GetString("disk_format")
	disk.AccessPath, _ = data.GetString("disk_path")
	disk.SetStatus(self.UserCred, models.DISK_READY, "")
	self.CleanHostSchedCache(disk)
	db.OpsLog.LogEvent(disk, db.ACT_ALLOCATE, disk.GetShortDesc(), self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *DiskCreateTask) OnDiskReadyFailed(ctx context.Context, disk *models.SDisk, resion error) {
	disk.SetStatus(self.UserCred, models.DISK_ALLOC_FAILED, resion.Error())
	self.SetStageFailed(ctx, resion.Error())
}
