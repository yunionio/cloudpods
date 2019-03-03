package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type DiskBatchCreateTask struct {
	SSchedTask
}

func init() {
	taskman.RegisterTask(DiskBatchCreateTask{})
}

func (self *DiskBatchCreateTask) getNeedScheduleDisks(objs []db.IStandaloneModel) []db.IStandaloneModel {
	toSchedDisks := make([]db.IStandaloneModel, 0)
	for _, obj := range objs {
		disk := obj.(*models.SDisk)
		if disk.StorageId == "" {
			toSchedDisks = append(toSchedDisks, disk)
		}
	}
	return toSchedDisks
}

func (self *DiskBatchCreateTask) OnInit(ctx context.Context, objs []db.IStandaloneModel, body jsonutils.JSONObject) {
	toSchedDisks := self.getNeedScheduleDisks(objs)
	if len(toSchedDisks) == 0 {
		self.SetStage("OnScheduleComplete", nil)
		// create not need schedule disks directly
		for _, disk := range objs {
			self.startCreateDisk(ctx, disk.(*models.SDisk))
		}
		return
	}
	StartScheduleObjects(ctx, self, toSchedDisks)
}

func (self *DiskBatchCreateTask) OnScheduleFailCallback(ctx context.Context, obj IScheduleModel, reason string) {
	self.SSchedTask.OnScheduleFailCallback(ctx, obj, reason)
	disk := obj.(*models.SDisk)
	log.Errorf("Schedule disk %s failed", disk.Name)
}

func (self *DiskBatchCreateTask) SaveScheduleResult(ctx context.Context, obj IScheduleModel, hostId string) {
	var err error
	disk := obj.(*models.SDisk)
	pendingUsage := models.SQuota{}
	err = self.GetPendingUsage(&pendingUsage)
	if err != nil {
		log.Errorf("GetPendingUsage fail %s", err)
	}
	diskConfig := models.SDiskConfig{}
	self.GetParams().Unmarshal(&diskConfig, "disk.0")
	quotaStorage := models.SQuota{Storage: disk.DiskSize}
	err = disk.SetStorageByHost(hostId, &diskConfig)
	if err != nil {
		models.QuotaManager.CancelPendingUsage(ctx, self.UserCred, disk.ProjectId, &pendingUsage, &quotaStorage)
		disk.SetStatus(self.UserCred, models.DISK_ALLOC_FAILED, err.Error())
		self.SetStageFailed(ctx, err.Error())
		db.OpsLog.LogEvent(disk, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
		notifyclient.NotifySystemError(disk.Id, disk.Name, models.DISK_ALLOC_FAILED, err.Error())
		return
	}

	self.startCreateDisk(ctx, disk)
}

func (self *DiskBatchCreateTask) startCreateDisk(ctx context.Context, disk *models.SDisk) {
	pendingUsage := models.SQuota{}
	err := self.GetPendingUsage(&pendingUsage)
	if err != nil {
		log.Errorf("GetPendingUsage fail %s", err)
	}
	quotaStorage := models.SQuota{Storage: disk.DiskSize}
	models.QuotaManager.CancelPendingUsage(ctx, self.UserCred, disk.ProjectId, &pendingUsage, &quotaStorage)
	self.SetPendingUsage(&pendingUsage)

	disk.StartDiskCreateTask(ctx, self.GetUserCred(), false, "", self.GetTaskId())
}

func (self *DiskBatchCreateTask) OnScheduleComplete(ctx context.Context, items []db.IStandaloneModel, data *jsonutils.JSONDict) {
	self.SetStageComplete(ctx, nil)
}
