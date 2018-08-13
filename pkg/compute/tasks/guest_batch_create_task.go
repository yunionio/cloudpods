package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type GuestBatchCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(GuestBatchCreateTask{})
}

func (self *GuestBatchCreateTask) OnInit(ctx context.Context, objs []db.IStandaloneModel, body jsonutils.JSONObject) {
	guests := make([]*models.SGuest, 0)
	for _, obj := range objs {
		guest := obj.(*models.SGuest)
		guests = append(guests, guest)
		db.OpsLog.LogEvent(guest, db.ACT_ALLOCATING, nil, self.UserCred)
		guest.SetStatus(self.UserCred, models.VM_SCHEDULE, "")
	}
	self.startScheduleGuests(ctx, guests)
}

func (self *GuestBatchCreateTask) startScheduleGuests(ctx context.Context, guests []*models.SGuest) {
	self.SetStage("on_guest_schedule_complete", nil)

	s := auth.GetAdminSession(options.Options.Region, "")
	results, err := modules.SchedManager.DoSchedule(s, self.Params, len(guests))
	if err != nil {
		self.onSchedulerRequestFail(ctx, guests, fmt.Sprintf("Scheduler fail: %s", err))
		return
	} else {
		self.onSchedulerResults(ctx, guests, results)
	}
}

func (self *GuestBatchCreateTask) cancelPendingUsage(ctx context.Context) {
	pendingUsage := models.SQuota{}
	err := self.GetPendingUsage(&pendingUsage)
	if err != nil {
		log.Errorf("Taks GetPendingUsage fail %s", err)
	} else {
		ownerProjectId, _ := self.Params.GetString("owner_tenant_id")
		err := models.QuotaManager.CancelPendingUsage(ctx, self.UserCred, ownerProjectId, &pendingUsage, &pendingUsage)
		if err != nil {
			log.Errorf("cancelpendingusage error %s", err)
		}
	}
}

func (self *GuestBatchCreateTask) onSchedulerRequestFail(ctx context.Context, guests []*models.SGuest, reason string) {
	for _, guest := range guests {
		self.onScheduleFail(ctx, guest, reason)
	}
	self.SetStageFailed(ctx, fmt.Sprintf("Schedule failed: %s", reason))
	self.cancelPendingUsage(ctx)
}

func (self *GuestBatchCreateTask) onScheduleFail(ctx context.Context, guest *models.SGuest, msg string) {
	lockman.LockObject(ctx, guest)
	defer lockman.ReleaseObject(ctx, guest)
	reason := "No matching resources"
	if len(msg) > 0 {
		reason = fmt.Sprintf("%s: %s", reason, msg)
	}
	guest.SetStatus(self.UserCred, models.VM_SCHEDULE_FAILED, reason)
	db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE_FAIL, reason, self.UserCred)
	notifyclient.NotifySystemError(guest.Id, guest.Name, models.VM_SCHEDULE_FAILED, reason)
}

func (self *GuestBatchCreateTask) onSchedulerResults(ctx context.Context, guests []*models.SGuest, results []jsonutils.JSONObject) {
	succCount := 0
	for idx := 0; idx < len(guests); idx += 1 {
		guest := guests[idx]
		result := results[idx]
		if result.Contains("candidate") {
			hostId, _ := result.GetString("candidate", "id")
			self.onScheduleSucc(ctx, guest, hostId)
			succCount += 1
		} else if result.Contains("error") {
			msg, _ := result.Get("error")
			self.onScheduleFail(ctx, guest, fmt.Sprintf("%s", msg))
		} else {
			msg := fmt.Sprintf("Unknown scheduler result %s", result)
			self.onScheduleFail(ctx, guest, msg)
			return
		}
	}
	if succCount == 0 {
		self.SetStageFailed(ctx, "Schedule failed")
	}
	self.cancelPendingUsage(ctx)
}

func (self *GuestBatchCreateTask) onScheduleSucc(ctx context.Context, guest *models.SGuest, hostId string) {
	lockman.LockObject(ctx, guest)
	defer lockman.ReleaseObject(ctx, guest)

	self.saveScheduleResult(ctx, guest, hostId)
	models.HostManager.ClearSchedDescCache(hostId)
}

func (self *GuestBatchCreateTask) saveScheduleResult(ctx context.Context, guest *models.SGuest, hostId string) {
	var err error

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

func (self *GuestBatchCreateTask) OnGuestScheduleComplete(ctx context.Context, items []db.IStandaloneModel, data *jsonutils.JSONDict) {
	self.SetStageComplete(ctx, nil)
}
