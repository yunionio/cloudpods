package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DiskResizeTask struct {
	SDiskBaseTask
}

func init() {
	taskman.RegisterTask(DiskResizeTask{})
}

func (self *DiskResizeTask) SetDiskReady(ctx context.Context, disk *models.SDisk, userCred mcclient.TokenCredential, reason string) {
	// 此函数主要避免虚机更改配置时，虚机可能出现中间状态
	if self.HasParentTask() {
		// 若是子任务，磁盘关联的虚拟机状态由父任务恢复，仅恢复磁盘自身状态即可
		disk.SetStatus(userCred, models.DISK_READY, reason)
	} else {
		// 若不是子任务，由于扩容时设置了关联的虚机状态，虚机的状态也由自己恢复
		disk.SetDiskReady(ctx, userCred, reason)
	}
}

func (self *DiskResizeTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)

	guestId, _ := self.Params.GetString("guest_id")
	var masterGuest *models.SGuest
	if len(guestId) > 0 {
		masterGuest = models.GuestManager.FetchGuestById(guestId)
	}

	storage := disk.GetStorage()
	host := storage.GetMasterHost()

	if masterGuest != nil {
		host = masterGuest.GetHost()
	}

	reason := "Cannot find host for disk"
	if host == nil || host.HostStatus != models.HOST_ONLINE {
		self.SetDiskReady(ctx, disk, self.GetUserCred(), reason)
		self.SetStageFailed(ctx, reason)
		db.OpsLog.LogEvent(disk, db.ACT_RESIZE_FAIL, reason, self.GetUserCred())
		logclient.AddActionLogWithStartable(self, disk, logclient.ACT_RESIZE, reason, self.UserCred, false)
		return
	}

	disk.SetStatus(self.GetUserCred(), models.DISK_START_RESIZE, "")
	if masterGuest != nil {
		masterGuest.SetStatus(self.GetUserCred(), models.VM_RESIZE_DISK, "")
	}
	self.StartResizeDisk(ctx, host, storage, disk, masterGuest)
}

func (self *DiskResizeTask) StartResizeDisk(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, guest *models.SGuest) {
	log.Infof("Resizing disk on host %s ...", host.GetName())
	self.SetStage("OnDiskResizeComplete", nil)
	sizeMb, _ := self.GetParams().Int("size")
	if err := host.GetHostDriver().RequestResizeDiskOnHost(ctx, host, storage, disk, guest, sizeMb, self); err != nil {
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
	self.SetDiskReady(ctx, disk, self.GetUserCred(), reason.Error())
	self.SetStageFailed(ctx, reason.Error())
	db.OpsLog.LogEvent(disk, db.ACT_RESIZE_FAIL, reason.Error(), self.GetUserCred())
	logclient.AddActionLogWithStartable(self, disk, logclient.ACT_RESIZE, reason.Error(), self.UserCred, false)
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
	self.SetDiskReady(ctx, disk, self.GetUserCred(), "")
	notes := fmt.Sprintf("%s=>%s", oldStatus, disk.Status)
	db.OpsLog.LogEvent(disk, db.ACT_UPDATE_STATUS, notes, self.UserCred)
	self.CleanHostSchedCache(disk)
	db.OpsLog.LogEvent(disk, db.ACT_RESIZE, disk.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, disk, logclient.ACT_RESIZE, nil, self.UserCred, true)
	self.OnDiskResized(ctx, disk)
}

func (self *DiskResizeTask) OnDiskResized(ctx context.Context, disk *models.SDisk) {
	guestId, _ := self.Params.GetString("guest_id")
	if len(guestId) > 0 {
		self.SetStage("TaskComplete", nil)
		masterGuest := models.GuestManager.FetchGuestById(guestId)
		if self.HasParentTask() {
			masterGuest.StartSyncTaskWithoutSyncstatus(ctx, self.UserCred, false, self.GetId())
		} else {
			masterGuest.StartSyncTask(ctx, self.UserCred, false, self.GetId())
		}
	} else {
		self.TaskComplete(ctx, disk, nil)
	}
}

func (self *DiskResizeTask) TaskComplete(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, disk.GetShortDesc(ctx))
	self.finalReleasePendingUsage(ctx)
}

func (self *DiskResizeTask) TaskCompleteFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data.String())
}

func (self *DiskResizeTask) OnDiskResizeCompleteFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	self.SetDiskReady(ctx, disk, self.GetUserCred(), data.String())
	db.OpsLog.LogEvent(disk, db.ACT_RESIZE_FAIL, disk.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, disk, logclient.ACT_RESIZE, data.String(), self.UserCred, false)
	guestId, _ := self.Params.GetString("guest_id")
	if len(guestId) > 0 {
		masterGuest := models.GuestManager.FetchGuestById(guestId)
		masterGuest.SetStatus(self.UserCred, models.VM_RESIZE_DISK_FAILED, data.String())
	}
	self.SetStageFailed(ctx, data.String())
}
