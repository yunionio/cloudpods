package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestBatchCreateTask struct {
	SSchedTask
}

func init() {
	taskman.RegisterTask(GuestBatchCreateTask{})
}

func (self *GuestBatchCreateTask) GetCreateInput() (*api.ServerCreateInput, error) {
	input := new(api.ServerCreateInput)
	err := self.GetParams().Unmarshal(input)
	return input, err
}

func (self *GuestBatchCreateTask) OnInit(ctx context.Context, objs []db.IStandaloneModel, body jsonutils.JSONObject) {
	StartScheduleObjects(ctx, self, objs)
}

func (self *GuestBatchCreateTask) OnScheduleFailCallback(ctx context.Context, obj IScheduleModel, reason string) {
	self.SSchedTask.OnScheduleFailCallback(ctx, obj, reason)
	guest := obj.(*models.SGuest)
	if guest.DisableDelete.IsTrue() {
		guest.SetDisableDelete(self.UserCred, false)
	}
}

func (self *GuestBatchCreateTask) SaveScheduleResultWithBackup(ctx context.Context, obj IScheduleModel, master, slave *schedapi.CandidateResource) {
	guest := obj.(*models.SGuest)
	guest.SetHostIdWithBackup(self.UserCred, master.HostId, slave.HostId)
	self.SaveScheduleResult(ctx, obj, master)
}

func (self *GuestBatchCreateTask) allocateGuestOnHost(ctx context.Context, guest *models.SGuest, candidate *schedapi.CandidateResource) error {
	pendingUsage := models.SQuota{}
	err := self.GetPendingUsage(&pendingUsage)
	if err != nil {
		log.Errorf("GetPendingUsage fail %s", err)
	}
	quotaCpuMem := models.SQuota{Cpu: int(guest.VcpuCount), Memory: guest.VmemSize}
	err = models.QuotaManager.CancelPendingUsage(ctx, self.UserCred, guest.ProjectId, &pendingUsage, &quotaCpuMem)
	self.SetPendingUsage(&pendingUsage)

	host := guest.GetHost()

	input, err := self.GetCreateInput()

	if host.IsPrepaidRecycle() {
		input, err = host.SetGuestCreateNetworkAndDiskParams(ctx, self.UserCred, input)
		if err != nil {
			log.Errorf("host.SetGuestCreateNetworkAndDiskParams fail %s", err)
			guest.SetStatus(self.UserCred, models.VM_CREATE_FAILED, err.Error())
			return err
		}
		params := input.JSON(input)
		self.SaveParams(params)
	}

	input, err = self.GetCreateInput()
	if err != nil {
		guest.SetStatus(self.UserCred, models.VM_CREATE_FAILED, err.Error())
		return err
	}
	err = guest.CreateNetworksOnHost(ctx, self.UserCred, host, input.Networks, &pendingUsage)
	self.SetPendingUsage(&pendingUsage)

	if err != nil {
		log.Errorf("Network failed: %s", err)
		guest.SetStatus(self.UserCred, models.VM_NETWORK_FAILED, err.Error())
		return err
	}

	guest.GetDriver().PrepareDiskRaidConfig(self.UserCred, host, input.BaremetalDiskConfigs)
	err = guest.CreateDisksOnHost(ctx, self.UserCred, host, input.Disks, &pendingUsage, true, true, candidate)
	self.SetPendingUsage(&pendingUsage)

	if err != nil {
		log.Errorf("Disk create failed: %s", err)
		guest.SetStatus(self.UserCred, models.VM_DISK_FAILED, err.Error())
		return err
	}

	err = guest.CreateIsolatedDeviceOnHost(ctx, self.UserCred, host, input.IsolatedDevices, &pendingUsage)
	self.SetPendingUsage(&pendingUsage)

	if err != nil {
		log.Errorf("IsolatedDevices create failed: %s", err)
		guest.SetStatus(self.UserCred, models.VM_DEVICE_FAILED, err.Error())
		return err
	}

	guest.JoinGroups(self.UserCred, self.Params)

	if guest.IsPrepaidRecycle() {
		err := host.RebuildRecycledGuest(ctx, self.UserCred, guest)
		if err != nil {
			log.Errorf("start guest create task fail %s", err)
			guest.SetStatus(self.UserCred, models.VM_CREATE_FAILED, err.Error())
			return err
		}

		autoStart := input.AutoStart
		resetPassword := true
		if input.ResetPassword != nil {
			resetPassword = *input.ResetPassword
		}
		passwd := input.Password
		err = guest.StartRebuildRootTask(ctx, self.UserCred, "", false, autoStart, passwd, resetPassword, true)
		if err != nil {
			log.Errorf("start guest create task fail %s", err)
			guest.SetStatus(self.UserCred, models.VM_CREATE_FAILED, err.Error())
			return err
		}
		return nil
	}

	err = guest.StartGuestCreateTask(ctx, self.UserCred, input, nil, self.GetId())
	if err != nil {
		log.Errorf("start guest create task fail %s", err)
		guest.SetStatus(self.UserCred, models.VM_CREATE_FAILED, err.Error())
		return err
	}
	return nil
}

func (self *GuestBatchCreateTask) SaveScheduleResult(ctx context.Context, obj IScheduleModel, candidate *schedapi.CandidateResource) {
	var err error
	hostId := candidate.HostId
	guest := obj.(*models.SGuest)
	if len(guest.HostId) == 0 {
		guest.OnScheduleToHost(ctx, self.UserCred, hostId)
	}

	err = self.allocateGuestOnHost(ctx, guest, candidate)
	if err != nil {
		db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
		logclient.AddActionLogWithStartable(self, obj, logclient.ACT_ALLOCATE, err.Error(), self.GetUserCred(), false)
		notifyclient.NotifySystemError(guest.Id, guest.Name, models.VM_CREATE_FAILED, err.Error())
		self.SetStageFailed(ctx, err.Error())
	}
}

func (self *GuestBatchCreateTask) OnScheduleComplete(ctx context.Context, items []db.IStandaloneModel, data *jsonutils.JSONDict) {
	self.SetStageComplete(ctx, nil)
}
