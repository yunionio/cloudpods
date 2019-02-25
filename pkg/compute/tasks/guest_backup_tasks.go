package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/pkg/utils"
)

type GuestSwitchToBackupTask struct {
	SGuestBaseTask
}

/*
0. ensure master guest stopped
1. stop backup guest
2. switch guest master host to backup host
3. start guest with new master
*/
func (self *GuestSwitchToBackupTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	host := guest.GetHost()
	self.Params.Set("is_force", jsonutils.JSONTrue)
	self.SetStage("OnEnsureMasterGuestStoped", nil)
	err := guest.GetDriver().RequestStopOnHost(ctx, guest, host, self)
	if err != nil {
		// In case of master host crash
		self.OnEnsureMasterGuestStoped(ctx, guest, nil)
	}
}

func (self *GuestSwitchToBackupTask) OnEnsureMasterGuestStoped(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	backupHost := models.HostManager.FetchHostById(guest.BackupHostId)
	self.Params.Set("is_force", jsonutils.JSONTrue)
	self.SetStage("OnBackupGuestStoped", nil)
	err := guest.GetDriver().RequestStopOnHost(ctx, guest, backupHost, self)
	if err != nil {
		self.OnFail(ctx, guest, fmt.Sprintf("Stop backup guest error: %s", err))
	}
}

func (self *GuestSwitchToBackupTask) OnBackupGuestStoped(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	disks := guest.GetDisks()
	for i := 0; i < len(disks); i++ {
		disk := disks[i].GetDisk()
		err := disk.SwitchToBackup()
		if err != nil {
			if i > 0 {
				for j := 0; j < i; j++ {
					disk = disks[j].GetDisk()
					disk.SwitchToBackup()
				}
			}
			self.OnFail(ctx, guest, fmt.Sprintf("Switch to backup disk error: %s", err))
			return
		}
	}
	err := guest.SwitchToBackup()
	if err != nil {
		self.OnFail(ctx, guest, fmt.Sprintf("Switch to backup guest error: %s", err))
		return
	}
	db.OpsLog.LogEvent(guest, db.ACT_SWITCHED, "Switch to backup", self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_SWITCH_TO_BACKUP, "Switch to backup", self.UserCred, true)

	self.SetStage("OnSwitched", nil)
	if jsonutils.QueryBoolean(self.Params, "purge_backup", false) {
		guest.StartGuestDeleteOnHostTask(ctx, self.UserCred, guest.BackupHostId, true, self.GetTaskId())
	} else if jsonutils.QueryBoolean(self.Params, "delete_backup", false) {
		guest.StartGuestDeleteOnHostTask(ctx, self.UserCred, guest.BackupHostId, false, self.GetTaskId())
	} else {
		self.OnSwitched(ctx, guest, nil)
	}
}

func (self *GuestSwitchToBackupTask) OnNewMasterStarted(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestSwitchToBackupTask) OnFail(ctx context.Context, guest *models.SGuest, reason string) {
	guest.SetStatus(self.UserCred, models.VM_SWITCH_TO_BACKUP_FAILED, reason)
	db.OpsLog.LogEvent(guest, db.ACT_SWITCH_FAILED, reason, self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_SWITCH_TO_BACKUP, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *GuestSwitchToBackupTask) OnSwitched(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	oldStatus, _ := self.Params.GetString("old_status")
	if utils.IsInStringArray(oldStatus, models.VM_RUNNING_STATUS) {
		self.SetStage("OnNewMasterStarted", nil)
		guest.StartGueststartTask(ctx, self.UserCred, nil, self.GetTaskId())
	} else {
		self.SetStageComplete(ctx, nil)
	}
}

func (self *GuestSwitchToBackupTask) OnSwitchedFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.OnFail(ctx, guest, data.String())
}

/********************* GuestStartAndSyncToBackupTask *********************/

type GuestStartAndSyncToBackupTask struct {
	SGuestBaseTask
}

func (self *GuestStartAndSyncToBackupTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.SetStage("OnCheckTemplete", nil)
	self.checkTemplete(ctx, guest)
}

func (self *GuestStartAndSyncToBackupTask) checkTemplete(ctx context.Context, guest *models.SGuest) {
	diskCat := guest.CategorizeDisks()
	if diskCat.Root != nil && len(diskCat.Root.GetTemplateId()) > 0 {
		err := guest.GetDriver().CheckDiskTemplateOnStorage(ctx, self.UserCred, diskCat.Root.GetTemplateId(), diskCat.Root.DiskFormat,
			diskCat.Root.BackupStorageId, self)
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
		}
	} else {
		self.OnCheckTemplete(ctx, guest, nil)
	}
}

func (self *GuestStartAndSyncToBackupTask) OnCheckTemplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStage("OnStartBackupGuest", nil)
	host := models.HostManager.FetchHostById(guest.BackupHostId)
	if _, err := guest.GetDriver().RequestStartOnHost(ctx, guest, host, self.UserCred, self); err != nil {
		self.SetStageFailed(ctx, err.Error())
	}
}

func (self *GuestStartAndSyncToBackupTask) OnStartBackupGuest(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	nbdServerPort, err := data.Int("nbd_server_port")
	if err != nil {
		self.SetStageFailed(ctx, "Start Backup Guest Missing Nbd Port")
		return
	}
	backupHost := models.HostManager.FetchHostById(guest.BackupHostId)
	nbdServerUri := fmt.Sprintf("nbd:%s:%d", backupHost.AccessIp, nbdServerPort)
	guest.SetMetadata(ctx, "backup_nbd_server_uri", nbdServerUri, self.UserCred)

	db.OpsLog.LogEvent(guest, db.ACT_BACKUP_START, "", self.UserCred)
	if utils.IsInStringArray(guest.Status, models.VM_RUNNING_STATUS) {
		self.SetStage("OnRequestSyncToBackup", nil)
		err := guest.GetDriver().RequestSyncToBackup(ctx, guest, self)
		if err != nil {
			self.SetStageFailed(ctx, fmt.Sprintf("Guest Request Sync to backup failed %s", err))
		}
	} else {
		self.SetStageComplete(ctx, nil)
	}
}

func (self *GuestStartAndSyncToBackupTask) OnStartBackupGuestFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_BACKUP_START_FAILED, "", self.UserCred)
	self.SetStageFailed(ctx, data.String())
}

func (self *GuestStartAndSyncToBackupTask) OnRequestSyncToBackup(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetStatus(self.UserCred, models.VM_BLOCK_STREAM, "OnSyncToBackup")
	self.SetStageComplete(ctx, nil)
}

type GuestCreateBackupTask struct {
	SSchedTask
}

func (self *GuestCreateBackupTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	StartScheduleObjects(ctx, self, []db.IStandaloneModel{obj})
}

func (self *GuestCreateBackupTask) OnStartSchedule(obj IScheduleModel) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, models.VM_BACKUP_CREATING, "")
	db.OpsLog.LogEvent(guest, db.ACT_START_CREATE_BACKUP, "", self.UserCred)
}

func (self *GuestCreateBackupTask) GetSchedParams() *jsonutils.JSONDict {
	obj := self.GetObject()
	guest := obj.(*models.SGuest)
	schedDesc := guest.ToSchedDesc()
	if self.Params.Contains("prefer_host_id") {
		preferHostId, _ := self.Params.Get("prefer_host_id")
		schedDesc.Set("prefer_host_id", preferHostId)
	}
	return schedDesc
}

func (self *GuestCreateBackupTask) OnScheduleFailCallback(ctx context.Context, obj IScheduleModel, reason string) {
	// do nothing
}

func (self *GuestCreateBackupTask) OnScheduleFailed(ctx context.Context, reason string) {
	obj := self.GetObject()
	guest := obj.(*models.SGuest)
	self.TaskFailed(ctx, guest, reason)
}

func (self *GuestCreateBackupTask) SaveScheduleResult(ctx context.Context, obj IScheduleModel, targetHostId string) {
	guest := obj.(*models.SGuest)
	targetHost := models.HostManager.FetchHostById(targetHostId)
	if targetHost == nil {
		self.TaskFailed(ctx, guest, "target host not found?")
		return
	}
	guest.SetHostIdWithBackup(guest.HostId, targetHostId)
	db.OpsLog.LogEvent(guest, db.ACT_CREATE_BACKUP, fmt.Sprintf("guest backup start create on host %s", targetHostId), self.UserCred)

	// backup disk only support disk backend local
	storage := guest.GetDriver().ChooseHostStorage(targetHost, models.STORAGE_LOCAL)
	if storage == nil {
		self.TaskFailed(ctx, guest, "Get backup storage error")
		return
	}
	self.StartCreateBackupDisks(ctx, guest, storage.Id)
}

func (self *GuestCreateBackupTask) StartCreateBackupDisks(ctx context.Context, guest *models.SGuest, storageId string) {
	guestDisks := guest.GetDisks()
	for i := 0; i < len(guestDisks); i++ {
		disk := guestDisks[i].GetDisk()
		disk.GetModelManager().TableSpec().Update(disk, func() error {
			disk.BackupStorageId = storageId
			return nil
		})
	}
	self.SetStage("OnCreateBackupDisks", nil)
	err := guest.CreateBackupDisks(ctx, self.UserCred, self.GetTaskId())
	if err != nil {
		self.TaskFailed(ctx, guest, err.Error())
	}
}

func (self *GuestCreateBackupTask) OnCreateBackupDisks(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStage("OnCreateBackup", nil)
	guest.StartCreateBackup(ctx, self.UserCred, self.GetTaskId(), nil)
}

func (self *GuestCreateBackupTask) OnCreateBackupDisksFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, guest, fmt.Sprintf("Create Backup Disks failed %s", data.String()))
}

func (self *GuestCreateBackupTask) OnCreateBackupFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, guest, fmt.Sprintf("Deploy Backup failed %s", data.String()))
}

func (self *GuestCreateBackupTask) OnCreateBackup(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guestStatus, _ := self.Params.GetString("guest_status")
	guest.SetStatus(self.UserCred, guestStatus, "")
	if utils.IsInStringArray(guestStatus, models.VM_RUNNING_STATUS) {
		self.OnGuestStart(ctx, guest, nil)
	} else {
		self.TaskCompleted(ctx, guest, "")
	}
}

func (self *GuestCreateBackupTask) OnGuestStart(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStage("OnSyncToBackup", nil)
	err := guest.GuestStartAndSyncToBackup(ctx, self.UserCred, nil, self.GetTaskId())
	if err != nil {
		self.TaskFailed(ctx, guest, fmt.Sprintf("Guest sycn to backup error %s", err.Error()))
	}
}

func (self *GuestCreateBackupTask) OnSyncToBackup(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.TaskCompleted(ctx, guest, "")
}

func (self *GuestCreateBackupTask) TaskCompleted(ctx context.Context, guest *models.SGuest, reason string) {
	db.OpsLog.LogEvent(guest, db.ACT_CREATE_BACKUP, reason, self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_CREATE_BACKUP, reason, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *GuestCreateBackupTask) TaskFailed(ctx context.Context, guest *models.SGuest, reason string) {
	guest.SetStatus(self.UserCred, models.VM_BACKUP_CREATE_FAILED, reason)
	db.OpsLog.LogEvent(guest, db.ACT_CREATE_BACKUP_FAILED, reason, self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_CREATE_BACKUP, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func init() {
	taskman.RegisterTask(GuestSwitchToBackupTask{})
	taskman.RegisterTask(GuestStartAndSyncToBackupTask{})
	taskman.RegisterTask(GuestCreateBackupTask{})
}
