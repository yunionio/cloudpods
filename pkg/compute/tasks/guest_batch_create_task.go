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
	SSchedTask
}

func init() {
	taskman.RegisterTask(GuestBatchCreateTask{})
}

func (self *GuestBatchCreateTask) OnInit(ctx context.Context, objs []db.IStandaloneModel, body jsonutils.JSONObject) {
	StartScheduleObjects(ctx, self, objs)
}

func (self *GuestBatchCreateTask) OnScheduleFailCallback(obj IScheduleModel, reason string) {
	self.SSchedTask.OnScheduleFailCallback(obj, reason)
	guest := obj.(*models.SGuest)
	if guest.DisableDelete.IsTrue() {
		guest.SetDisableDelete(false)
	}
}

func (self *GuestBatchCreateTask) SaveScheduleResultWithBackup(ctx context.Context, obj IScheduleModel, master, slave string) {
	guest := obj.(*models.SGuest)
	guest.SetHostIdWithBackup(master, slave)
	self.SaveScheduleResult(ctx, obj, master)
}

func (self *GuestBatchCreateTask) SaveScheduleResult(ctx context.Context, obj IScheduleModel, hostId string) {
	var err error
	guest := obj.(*models.SGuest)
	pendingUsage := models.SQuota{}
	err = self.GetPendingUsage(&pendingUsage)
	if err != nil {
		log.Errorf("GetPendingUsage fail %s", err)
	}
	if len(guest.HostId) == 0 {
		guest.OnScheduleToHost(ctx, self.UserCred, hostId)
	}

	quotaCpuMem := models.SQuota{Cpu: int(guest.VcpuCount), Memory: guest.VmemSize}
	err = models.QuotaManager.CancelPendingUsage(ctx, self.UserCred, guest.ProjectId, &pendingUsage, &quotaCpuMem)
	self.SetPendingUsage(&pendingUsage)

	host := guest.GetHost()

	if host.IsPrepaidRecycle() {
		self.Params, err = host.SetGuestCreateNetworkAndDiskParams(ctx, self.UserCred, self.Params)
		if err != nil {
			log.Errorf("host.SetGuestCreateNetworkAndDiskParams fail %s", err)
			guest.SetStatus(self.UserCred, models.VM_CREATE_FAILED, err.Error())
			self.SetStageFailed(ctx, err.Error())
			db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
			notifyclient.NotifySystemError(guest.Id, guest.Name, models.VM_CREATE_FAILED, err.Error())
			return
		}
		self.SaveParams(self.Params)
	}

	log.Debugf("%s", self.Params)
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
	err = guest.CreateDisksOnHost(ctx, self.UserCred, host, self.Params, &pendingUsage, true)
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

	if guest.IsPrepaidRecycle() {
		err := host.RebuildRecycledGuest(ctx, self.UserCred, guest)
		if err != nil {
			log.Errorf("start guest create task fail %s", err)
			guest.SetStatus(self.UserCred, models.VM_CREATE_FAILED, err.Error())
			self.SetStageFailed(ctx, err.Error())
			db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
			notifyclient.NotifySystemError(guest.Id, guest.Name, models.VM_CREATE_FAILED, err.Error())
			return
		}

		autoStart := jsonutils.QueryBoolean(self.Params, "auto_start", false)
		resetPassword := jsonutils.QueryBoolean(self.Params, "reset_password", true)
		passwd, _ := self.Params.GetString("password")
		err = guest.StartRebuildRootTask(ctx, self.UserCred, "", false, autoStart, passwd, resetPassword, true)
		if err != nil {
			log.Errorf("start guest create task fail %s", err)
			guest.SetStatus(self.UserCred, models.VM_CREATE_FAILED, err.Error())
			self.SetStageFailed(ctx, err.Error())
			db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
			notifyclient.NotifySystemError(guest.Id, guest.Name, models.VM_CREATE_FAILED, err.Error())
			return
		}
		return
	}
	err = guest.StartGuestCreateTask(ctx, self.UserCred, self.Params, nil, self.GetId())
	if err != nil {
		log.Errorf("start guest create task fail %s", err)
		guest.SetStatus(self.UserCred, models.VM_CREATE_FAILED, err.Error())
		self.SetStageFailed(ctx, err.Error())
		db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
		notifyclient.NotifySystemError(guest.Id, guest.Name, models.VM_CREATE_FAILED, err.Error())
	}
}

func (self *GuestBatchCreateTask) OnScheduleComplete(ctx context.Context, items []db.IStandaloneModel, data *jsonutils.JSONDict) {
	self.SetStageComplete(ctx, nil)
}
