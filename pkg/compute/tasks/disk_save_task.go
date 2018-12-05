package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	mc "yunion.io/x/onecloud/pkg/mcclient/modules"
)

type DiskSaveTask struct {
	SDiskBaseTask
}

func init() {
	taskman.RegisterTask(DiskSaveTask{})
}

func (self *DiskSaveTask) GetMasterHost(disk *models.SDisk) *models.SHost {
	if guests := disk.GetGuests(); len(guests) == 1 {
		if host := guests[0].GetHost(); host == nil {
			if storage := disk.GetStorage(); storage != nil {
				return storage.GetMasterHost()
			}
		} else {
			return host
		}
	}
	return nil
}

func (self *DiskSaveTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)
	if host := self.GetMasterHost(disk); host == nil {
		resion := "Cannot find host for disk"
		disk.SetDiskReady(ctx, self.GetUserCred(), resion)
		self.TaskFailed(ctx, resion)
		db.OpsLog.LogEvent(disk, db.ACT_SAVE_FAIL, resion, self.GetUserCred())
	} else {
		disk.SetStatus(self.GetUserCred(), models.DISK_START_SAVE, "")
		for _, guest := range disk.GetGuests() {
			guest.SetStatus(self.GetUserCred(), models.VM_SAVE_DISK, "")
		}
		self.StartBackupDisk(ctx, disk, host)
	}
}

func (self *DiskSaveTask) StartBackupDisk(ctx context.Context, disk *models.SDisk, host *models.SHost) {
	self.SetStage("on_disk_backup_complete", nil)
	disk.SetStatus(self.GetUserCred(), models.DISK_SAVING, "")
	imageId, _ := self.GetParams().GetString("image_id")
	if err := host.GetHostDriver().RequestPrepareSaveDiskOnHost(ctx, host, disk, imageId, self); err != nil {
		log.Errorf("Backup failed: %v", err)
		disk.SetDiskReady(ctx, self.GetUserCred(), err.Error())
		self.TaskFailed(ctx, err.Error())
		db.OpsLog.LogEvent(disk, db.ACT_SAVE_FAIL, err.Error(), self.GetUserCred())
	}
}

func (self *DiskSaveTask) OnDiskBackupCompleteFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	disk.SetDiskReady(ctx, self.GetUserCred(), data.String())
	db.OpsLog.LogEvent(disk, db.ACT_SAVE_FAIL, data.String(), self.GetUserCred())
	self.SetStageFailed(ctx, data.String())
}

func (self *DiskSaveTask) OnDiskBackupComplete(ctx context.Context, disk *models.SDisk, data *jsonutils.JSONDict) {
	disk.SetDiskReady(ctx, self.GetUserCred(), "")
	db.OpsLog.LogEvent(disk, db.ACT_SAVE, disk.GetShortDesc(), self.GetUserCred())
	self.SetStageComplete(ctx, nil)
	imageId, _ := self.GetParams().GetString("image_id")
	if host := self.GetMasterHost(disk); host == nil {
		log.Errorf("Saved disk Host mast not be nil")
		self.TaskFailed(ctx, "Saved disk Host mast not be nil")
	} else {
		if self.Params.Contains("format") {
			format, _ := self.Params.Get("format")
			data.Add(format, "format")
		}
		if err := self.UploadDisk(ctx, host, disk, imageId, data); err != nil {
			log.Errorf("UploadDisk failed: %v", err)
			self.TaskFailed(ctx, err.Error())
		}
		self.RefreshImageCache(ctx, imageId)
	}
}

func (self *DiskSaveTask) RefreshImageCache(ctx context.Context, imageId string) {
	models.CachedimageManager.GetImageById(ctx, self.GetUserCred(), imageId, true)
}

func (self *DiskSaveTask) UploadDisk(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, data *jsonutils.JSONDict) error {
	return host.GetHostDriver().RequestSaveUploadImageOnHost(ctx, host, disk, imageId, self, jsonutils.Marshal(data))
}

func (self *DiskSaveTask) TaskFailed(ctx context.Context, resion string) {
	self.SetStageFailed(ctx, resion)
	if imageId, err := self.GetParams().GetString("image_id"); err != nil && len(imageId) > 0 {
		log.Errorf("save disk task failed, set image %s killed", imageId)
		s := auth.GetAdminSession(options.Options.Region, "")
		mc.Images.Update(s, imageId, jsonutils.Marshal(map[string]string{"status": "killed"}))
	}
}
