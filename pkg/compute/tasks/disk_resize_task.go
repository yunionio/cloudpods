package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
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
		for _, guest := range disk.GetGuests() {
			host = guest.GetHost()
		}
	}
	reason := "Cannot find host for disk"
	if host == nil || host.HostStatus != models.HOST_ONLINE {
		disk.SetDiskReady(ctx, self.GetUserCred(), reason)
		self.SetStageFailed(ctx, reason)
		db.OpsLog.LogEvent(disk, db.ACT_RESIZE_FAIL, reason, self.GetUserCred())
		logclient.AddActionLog(disk, logclient.ACT_RESIZE, reason, self.UserCred, false)
	} else {
		disk.SetStatus(self.GetUserCred(), models.DISK_START_RESIZE, "")
		for _, guest := range disk.GetGuests() {
			guest.SetStatus(self.GetUserCred(), models.VM_RESIZE_DISK, "")
		}
		self.StartResizeDisk(ctx, host, storage, disk, online)
	}
}

func (self *DiskResizeTask) StartResizeDisk(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, online bool) {
	log.Infof("Resizing disk on host %s ...", host.GetName())
	self.SetStage("on_disk_resize_complete", nil)
	sizeMb, _ := self.GetParams().Int("size")
	proc := host.GetHostDriver().RequestResizeDiskOnHost
	if online {
		proc = host.GetHostDriver().RequestResizeDiskOnHostOnline
	}
	if err := proc(ctx, host, storage, disk, sizeMb, self); err != nil {
		log.Errorf("request_resize_disk_on_host: %v", err)
		self.OnStartResizeDiskFailed(ctx, disk, err)
		return
	}
	self.OnStartResizeDiskSucc(ctx, disk)
}

func (self *DiskResizeTask) OnStartResizeDiskSucc(ctx context.Context, disk *models.SDisk) {
	disk.SetStatus(self.GetUserCred(), models.DISK_RESIZING, "")
}

func (self *DiskResizeTask) OnStartResizeDiskFailed(ctx context.Context, disk *models.SDisk, reason error) {
	disk.SetDiskReady(ctx, self.GetUserCred(), reason.Error())
	self.SetStageFailed(ctx, reason.Error())
	db.OpsLog.LogEvent(disk, db.ACT_RESIZE_FAIL, reason.Error(), self.GetUserCred())
	logclient.AddActionLog(disk, logclient.ACT_RESIZE, reason.Error(), self.UserCred, false)
}

func (self *DiskResizeTask) OnDiskResizeComplete(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	jSize, err := data.Get("disk_size")
	if err != nil {
		log.Errorf("OnDiskResizeComplete error: %s", err.Error())
		self.OnStartResizeDiskFailed(ctx, disk, err)
		return
	}
	sizeMb, err := jSize.Int()
	if err != nil {
		log.Errorf("OnDiskResizeComplete error: %s", err.Error())
		self.OnStartResizeDiskFailed(ctx, disk, err)
		return
	}
	oldStatus := disk.Status
	_, err = disk.GetModelManager().TableSpec().Update(disk, func() error {
		disk.Status = models.DISK_READY
		disk.DiskSize = int(sizeMb)
		return nil
	})
	if err != nil {
		log.Errorf("OnDiskResizeComplete error: %s", err.Error())
		self.OnStartResizeDiskFailed(ctx, disk, err)
		return
	}
	disk.SetDiskReady(ctx, self.GetUserCred(), "")
	notes := fmt.Sprintf("%s=>%s", oldStatus, disk.Status)
	db.OpsLog.LogEvent(disk, db.ACT_UPDATE_STATUS, notes, self.UserCred)
	self.CleanHostSchedCache(disk)
	db.OpsLog.LogEvent(disk, db.ACT_RESIZE, disk.GetShortDesc(), self.UserCred)
	logclient.AddActionLog(disk, logclient.ACT_RESIZE, nil, self.UserCred, true)
	self.SetStageComplete(ctx, disk.GetShortDesc())
	self.finalReleasePendingUsage(ctx)
}

func (self *DiskResizeTask) OnDiskResizeCompleteFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	disk.SetDiskReady(ctx, self.GetUserCred(), data.String())
	db.OpsLog.LogEvent(disk, db.ACT_RESIZE_FAIL, disk.GetShortDesc(), self.UserCred)
	logclient.AddActionLog(disk, logclient.ACT_RESIZE, data.String(), self.UserCred, false)
	self.SetStageFailed(ctx, data.String())
}
