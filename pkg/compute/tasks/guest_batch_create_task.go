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

type GuestBatchCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(GuestBatchCreateTask{})
}

func (self *GuestBatchCreateTask) OnInit(ctx context.Context, objs []db.IStandaloneModel, body jsonutils.JSONObject) {
	StartScheduleObjects(ctx, self, objs)
}

func (self *GuestBatchCreateTask) OnScheduleFailCallback(obj IScheduleModel) {
	guest := obj.(*models.SGuest)
	if guest.DisableDelete.IsTrue() {
		guest.SetDisableDelete(false)
	}
}

func (self *GuestBatchCreateTask) SaveScheduleResult(ctx context.Context, obj IScheduleModel, hostId string) {
	var err error
	guest := obj.(*models.SGuest)
	pendingUsage := models.SQuota{}
	err = self.GetPendingUsage(&pendingUsage)
	if err != nil {
		log.Errorf("GetPendingUsage fail %s", err)
	}
	guest.SetHostId(hostId)
	quotaCpuMem := models.SQuota{Cpu: int(guest.VcpuCount), Memory: guest.VmemSize}
	err = models.QuotaManager.CancelPendingUsage(ctx, self.UserCred, guest.ProjectId, &pendingUsage, &quotaCpuMem)
	self.SetPendingUsage(&pendingUsage)

	host := guest.GetHost()

	err = guest.CreateNetworksOnHost(ctx, self.UserCred, host, self.Params, &pendingUsage)
	self.SetPendingUsage(&pendingUsage)

	if err != nil {
		log.Errorf("Network failed: %s", err)
		guest.SetStatus(self.UserCred, models.VM_NETWORK_FAILED, err.Error())
		self.SetStageFailed(ctx, err.Error())
		db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
		notifyclient.NotifySystemError(guest.Id, guest.Name, models.VM_NETWORK_FAILED, err.Error())
		return
	}

	guest.GetDriver().PrepareDiskRaidConfig(host, self.Params)
	err = guest.CreateDisksOnHost(ctx, self.UserCred, host, self.Params, &pendingUsage)
	self.SetPendingUsage(&pendingUsage)

	if err != nil {
		log.Errorf("Disk create failed: %s", err)
		guest.SetStatus(self.UserCred, models.VM_DISK_FAILED, err.Error())
		self.SetStageFailed(ctx, err.Error())
		db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
		notifyclient.NotifySystemError(guest.Id, guest.Name, models.VM_DISK_FAILED, err.Error())
		return
	}

	err = guest.CreateIsolatedDeviceOnHost(ctx, self.UserCred, host, self.Params, &pendingUsage)
	self.SetPendingUsage(&pendingUsage)

	if err != nil {
		log.Errorf("IsolatedDevices create failed: %s", err)
		guest.SetStatus(self.UserCred, models.VM_DEVICE_FAILED, err.Error())
		self.SetStageFailed(ctx, err.Error())
		db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
		notifyclient.NotifySystemError(guest.Id, guest.Name, models.VM_DEVICE_FAILED, err.Error())
		return
	}

	guest.JoinGroups(self.UserCred, self.Params)

	err = guest.StartGuestCreateTask(ctx, self.UserCred, self.Params, nil, self.GetId())
	if err != nil {
		log.Errorf("start guest create task fail %s", err)
		guest.SetStatus(self.UserCred, models.VM_CREATE_FAILED, err.Error())
		self.SetStageFailed(ctx, err.Error())
		db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
		notifyclient.NotifySystemError(guest.Id, guest.Name, models.VM_DEVICE_FAILED, err.Error())
	}
}

func (self *GuestBatchCreateTask) OnScheduleComplete(ctx context.Context, items []db.IStandaloneModel, data *jsonutils.JSONDict) {
	self.SetStageComplete(ctx, nil)
}
