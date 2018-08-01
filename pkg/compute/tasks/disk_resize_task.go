package tasks

import (
	"context"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/compute/models"
)

type DiskResizeTask struct {
	SDiskBaseTask
}

func init() {
	taskman.RegisterTask(DiskResizeTask{})
}

func (self *DiskResizeTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)
	storage := disk.GetStorage()
	host := storage.GetMasterHost()
	online := disk.GetRuningGuestCount() > 0
	if online {
		for _, guest := range disk.GetGuest() {
			host = guest.GetHost()
		}
	}
	resion := "Cannot find host for disk"
	if host == nil || host.HostStatus != models.HOST_ONLINE {
		disk.SetStatus(self.GetUserCred(), models.DISK_READY, resion)
		self.SetStageFailed(ctx, resion)
		db.OpsLog.LogEvent(disk, db.ACT_RESIZE_FAIL, resion, self.GetUserCred())
	} else {
		disk.SetStatus(self.GetUserCred(), models.DISK_START_RESIZE, "")
		for _, guest := range disk.GetGuest() {
			guest.SetStatus(self.GetUserCred(), models.VM_RESIZE_DISK, "")
		}
		self.StartResizeDisk(ctx, host, storage, disk, online)
	}
}

func (self *DiskResizeTask) StartResizeDisk(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, online bool) {
	log.Infof("Resizing disk on host %s ...", host.GetName())
	self.SetStage("on_disk_resize_complete", nil)
	size, _ := self.GetParams().Int("size")
	proc := host.GetHostDriver().RequestResizeDiskOnHost
	if online {
		proc = host.GetHostDriver().RequestResizeDiskOnHostOnline
	}
	if err := proc(host, storage, disk, size, self); err != nil {
		log.Errorf("request_resize_disk_on_host: %s", err.Error())
		self.OnStartResizeDiskFailed(ctx, err)
	}
	self.OnStartResizeDiskSucc(ctx, disk)
}

func (self *DiskResizeTask) OnStartResizeDiskSucc(ctx context.Context, disk *models.SDisk) {
	disk.SetStatus(self.GetUserCred(), models.DISK_RESIZING, "")
}

func (self *DiskResizeTask) OnStartResizeDiskFailed(ctx context.Context, resion error) {
	disk := self.getDisk()
	disk.SetStatus(self.GetUserCred(), models.DISK_READY, resion.Error())
	self.SetStageFailed(ctx, resion.Error())
	db.OpsLog.LogEvent(disk, db.ACT_RESIZE_FAIL, resion.Error(), self.GetUserCred())
}

func (self *DiskResizeTask) OnDiskResizeComplete(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	disk.SetStatus(self.UserCred, models.DISK_READY, "")
	self.CleanHostSchedCache(disk)
	db.OpsLog.LogEvent(disk, db.ACT_RESIZE, disk.GetShortDesc(), self.UserCred)
	self.SetStageComplete(ctx, disk.GetShortDesc())
	self.finalReleasePendingUsage(ctx)
}

func (self *DiskResizeTask) OnDiskResizeCompleteFailed(ctx context.Context, disk *models.SDisk, resion error) {
	disk.SetStatus(self.UserCred, models.DISK_READY, resion.Error())
	db.OpsLog.LogEvent(disk, db.ACT_RESIZE_FAIL, disk.GetShortDesc(), self.UserCred)
}
