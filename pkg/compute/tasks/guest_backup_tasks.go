// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
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
	host, _ := guest.GetHost()
	self.Params.Set("is_force", jsonutils.JSONTrue)
	self.SetStage("OnEnsureMasterGuestStoped", nil)
	err := guest.GetDriver().RequestStopOnHost(ctx, guest, host, self, true)
	if err != nil {
		// In case of master host crash
		self.OnEnsureMasterGuestStoped(ctx, guest, nil)
	}
}

func (self *GuestSwitchToBackupTask) OnEnsureMasterGuestStoped(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	backupHost := models.HostManager.FetchHostById(guest.BackupHostId)
	self.Params.Set("is_force", jsonutils.JSONTrue)
	self.SetStage("OnBackupGuestStoped", nil)
	err := guest.GetDriver().RequestStopOnHost(ctx, guest, backupHost, self, true)
	if err != nil {
		self.OnFail(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (self *GuestSwitchToBackupTask) OnBackupGuestStoped(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	disks, err := guest.GetDisks()
	if err != nil {
		self.OnFail(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	for i := 0; i < len(disks); i++ {
		disk := disks[i]
		err := disk.SwitchToBackup(self.UserCred)
		if err != nil {
			if i > 0 {
				for j := 0; j < i; j++ {
					disk = disks[j]
					disk.SwitchToBackup(self.UserCred)
				}
			}
			self.OnFail(ctx, guest, jsonutils.NewString(fmt.Sprintf("Switch to backup disk error: %s", err)))
			return
		}
	}
	err = guest.SwitchToBackup(self.UserCred)
	if err != nil {
		self.OnFail(ctx, guest, jsonutils.NewString(fmt.Sprintf("Switch to backup guest error: %s", err)))
		return
	}
	if err := guest.SetGuestBackupMirrorJobNotReady(ctx, self.UserCred); err != nil {
		self.OnFail(ctx, guest, jsonutils.NewString("guest set metadata failed"))
		return
	}
	db.OpsLog.LogEvent(guest, db.ACT_SWITCHED, "Switch to backup", self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_SWITCH_TO_BACKUP, "Switch to backup", self.UserCred, true)
	oldStatus, _ := self.Params.GetString("old_status")
	autoStart := jsonutils.QueryBoolean(self.Params, "auto_start", false) ||
		utils.IsInStringArray(oldStatus, api.VM_RUNNING_STATUS)
	if autoStart {
		self.SetStage("OnGuestStartCompleted", nil)
		if err := guest.StartGueststartTask(ctx, self.UserCred, nil, self.GetId()); err != nil {
			self.OnGuestStartCompletedFailed(ctx, guest,
				jsonutils.NewString(fmt.Sprintf("start guest start task: %s", err)))
		}
	} else {
		self.OnComplete(ctx, guest, nil)
	}
}

func (self *GuestSwitchToBackupTask) OnFail(ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject) {
	guest.SetStatus(self.UserCred, api.VM_SWITCH_TO_BACKUP_FAILED, reason.String())
	db.OpsLog.LogEvent(guest, db.ACT_SWITCH_FAILED, reason, self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_SWITCH_TO_BACKUP, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *GuestSwitchToBackupTask) OnGuestStartCompleted(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestSwitchToBackupTask) OnGuestStartCompletedFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}

func (self *GuestSwitchToBackupTask) OnComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

/********************* GuestStartAndSyncToBackupTask *********************/

type GuestStartAndSyncToBackupTask struct {
	SGuestBaseTask
}

func (self *GuestStartAndSyncToBackupTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, api.VM_BACKUP_STARTING, "GuestStartAndSyncToBackupTask")
	self.SetStage("OnCheckTemplete", nil)
	self.checkTemplete(ctx, guest)
}

func (self *GuestStartAndSyncToBackupTask) checkTemplete(ctx context.Context, guest *models.SGuest) {
	diskCat := guest.CategorizeDisks()
	if diskCat.Root != nil && len(diskCat.Root.GetTemplateId()) > 0 {
		err := guest.GetDriver().CheckDiskTemplateOnStorage(ctx, self.UserCred, diskCat.Root.GetTemplateId(), diskCat.Root.DiskFormat,
			diskCat.Root.BackupStorageId, self)
		if err != nil {
			self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		}
	} else {
		self.OnCheckTemplete(ctx, guest, nil)
	}
}

func (self *GuestStartAndSyncToBackupTask) OnCheckTemplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStage("OnStartBackupGuest", nil)
	host := models.HostManager.FetchHostById(guest.BackupHostId)
	err := guest.GetDriver().RequestStartOnHost(ctx, guest, host, self.UserCred, self)
	if err != nil {
		self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
	}
}

func (self *GuestStartAndSyncToBackupTask) OnStartBackupGuest(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetBackupGuestStatus(self.UserCred, api.VM_RUNNING, "on start backup guest")
	nbdServerPort, err := data.Int("nbd_server_port")
	if err != nil {
		self.SetStageFailed(ctx, jsonutils.NewString("Start Backup Guest Missing Nbd Port"))
		return
	}
	backupHost := models.HostManager.FetchHostById(guest.BackupHostId)
	nbdServerUri := fmt.Sprintf("nbd:%s:%d", backupHost.AccessIp, nbdServerPort)
	guest.SetMetadata(ctx, "backup_nbd_server_uri", nbdServerUri, self.UserCred)
	db.OpsLog.LogEvent(guest, db.ACT_BACKUP_START, "", self.UserCred)

	// try get origin guest status
	guestStatus, err := self.Params.GetString("guest_status")
	if err != nil {
		guestStatus = guest.Status
	}

	if utils.IsInStringArray(guestStatus, api.VM_RUNNING_STATUS) {
		self.SetStage("OnRequestSyncToBackup", nil)
		err = guest.GetDriver().RequestSyncToBackup(ctx, guest, self)
		if err != nil {
			self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		}
	} else {
		self.SetStageComplete(ctx, nil)
	}
}

func (self *GuestStartAndSyncToBackupTask) OnStartBackupGuestFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetGuestBackupMirrorJobFailed(ctx, self.UserCred)
	db.OpsLog.LogEvent(guest, db.ACT_BACKUP_START_FAILED, data.String(), self.UserCred)
	self.SetStageFailed(ctx, data)
}

func (self *GuestStartAndSyncToBackupTask) OnRequestSyncToBackup(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetGuestBackupMirrorJobInProgress(ctx, self.UserCred)
	guest.SetStatus(self.UserCred, api.VM_BLOCK_STREAM, "OnSyncToBackup")
	self.SetStageComplete(ctx, nil)
}

func (self *GuestStartAndSyncToBackupTask) OnRequestSyncToBackupFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetGuestBackupMirrorJobFailed(ctx, self.UserCred)
	guest.SetStatus(self.UserCred, api.VM_BLOCK_STREAM_FAIL, "OnSyncToBackup")
	self.SetStageFailed(ctx, data)
}

type GuestCreateBackupTask struct {
	SSchedTask
}

func (self *GuestCreateBackupTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	StartScheduleObjects(ctx, self, []db.IStandaloneModel{obj})
}

func (self *GuestCreateBackupTask) OnStartSchedule(obj IScheduleModel) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_START_CREATE_BACKUP, "", self.UserCred)
}

func (self *GuestCreateBackupTask) GetSchedParams() (*schedapi.ScheduleInput, error) {
	obj := self.GetObject()
	guest := obj.(*models.SGuest)
	schedDesc := guest.ToSchedDesc()
	if self.Params.Contains("prefer_host_id") {
		preferHostId, _ := self.Params.GetString("prefer_host_id")
		schedDesc.ServerConfig.PreferHost = preferHostId
	}
	schedDesc.ReuseNetwork = true
	return schedDesc, nil
}

func (self *GuestCreateBackupTask) OnScheduleFailCallback(ctx context.Context, obj IScheduleModel, reason jsonutils.JSONObject) {
	// do nothing
}

func (self *GuestCreateBackupTask) OnScheduleFailed(ctx context.Context, reason jsonutils.JSONObject) {
	obj := self.GetObject()
	guest := obj.(*models.SGuest)
	self.TaskFailed(ctx, guest, reason)
}

func (self *GuestCreateBackupTask) SaveScheduleResult(ctx context.Context, obj IScheduleModel, candidate *schedapi.CandidateResource) {
	guest := obj.(*models.SGuest)
	targetHostId := candidate.HostId
	targetHost := models.HostManager.FetchHostById(candidate.HostId)
	if targetHost == nil {
		self.TaskFailed(ctx, guest, jsonutils.NewString("target host not found?"))
		return
	}
	guest.SetHostIdWithBackup(self.UserCred, guest.HostId, targetHostId)
	db.OpsLog.LogEvent(guest, db.ACT_CREATE_BACKUP, fmt.Sprintf("guest backup start create on host %s", targetHostId), self.UserCred)

	self.StartCreateBackupDisks(ctx, guest, targetHost, candidate.Disks)
}

func (self *GuestCreateBackupTask) StartCreateBackupDisks(ctx context.Context, guest *models.SGuest, host *models.SHost, candidateDisks []*schedapi.CandidateDisk) {
	guestDisks, _ := guest.GetGuestDisks()
	for i := 0; i < len(guestDisks); i++ {
		var candidateDisk *schedapi.CandidateDisk
		if len(candidateDisks) >= i {
			candidateDisk = candidateDisks[i]
		}
		diskConfig := &api.DiskConfig{Backend: api.STORAGE_LOCAL}
		storage, err := guest.ChooseHostStorage(host, diskConfig, candidateDisk)
		if err != nil {
			self.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("unable to ChooseHostStorage: %v", err)))
			return
		}
		if storage == nil {
			self.TaskFailed(ctx, guest, jsonutils.NewString("Get backup storage error"))
			return
		}
		disk := guestDisks[i].GetDisk()
		db.Update(disk, func() error {
			disk.BackupStorageId = storage.Id
			return nil
		})
	}
	self.SetStage("OnCreateBackupDisks", nil)
	err := guest.CreateBackupDisks(ctx, self.UserCred, self.GetTaskId())
	if err != nil {
		self.TaskFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (self *GuestCreateBackupTask) StartInsertIso(ctx context.Context, guest *models.SGuest, imageId string) {
	self.SetStage("OnInsertIso", nil)
	guest.StartInsertIsoTask(ctx, 0, imageId, false, nil, guest.BackupHostId, self.UserCred, self.GetTaskId())
}

func (self *GuestCreateBackupTask) OnInsertIso(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStage("OnCreateBackup", nil)
	guest.StartCreateBackup(ctx, self.UserCred, self.GetTaskId(), nil)
}

func (self *GuestCreateBackupTask) OnInsertIsoFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("Backup guest insert ISO failed %s", data.String())))
}

func (self *GuestCreateBackupTask) OnCreateBackupDisks(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if cdrom := guest.GetCdrom(); cdrom != nil && len(cdrom.ImageId) > 0 {
		self.StartInsertIso(ctx, guest, cdrom.ImageId)
	} else {
		self.SetStage("OnCreateBackup", nil)
		guest.StartCreateBackup(ctx, self.UserCred, self.GetTaskId(), nil)
	}
}

func (self *GuestCreateBackupTask) OnCreateBackupDisksFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, guest, data)
}

func (self *GuestCreateBackupTask) OnCreateBackupFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, guest, data)
}

func (self *GuestCreateBackupTask) OnCreateBackup(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetBackupGuestStatus(self.UserCred, api.VM_READY, "on create backup")
	guestStatus, _ := self.Params.GetString("guest_status")
	if utils.IsInStringArray(guestStatus, api.VM_RUNNING_STATUS) {
		self.OnGuestStart(ctx, guest, guestStatus)
	} else if jsonutils.QueryBoolean(self.Params, "auto_start", false) {
		self.RequestStartGuest(ctx, guest)
	} else {
		self.TaskCompleted(ctx, guest, "")
	}
}

func (self *GuestCreateBackupTask) OnGuestStart(ctx context.Context, guest *models.SGuest, guestStatus string) {
	self.SetStage("OnSyncToBackup", nil)
	err := guest.GuestStartAndSyncToBackup(ctx, self.UserCred, self.GetTaskId(), guestStatus)
	if err != nil {
		self.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("Guest sycn to backup error %s", err.Error())))
	}
}

func (self *GuestCreateBackupTask) OnSyncToBackup(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.TaskCompleted(ctx, guest, "")
}

func (self *GuestCreateBackupTask) OnSyncToBackupFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, guest, data)
}

func (self *GuestCreateBackupTask) RequestStartGuest(ctx context.Context, guest *models.SGuest) {
	self.SetStage("OnGuestStartCompleted", nil)
	guest.StartGueststartTask(ctx, self.UserCred, nil, self.GetId())
}

func (self *GuestCreateBackupTask) OnGuestStartCompleted(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.TaskCompleted(ctx, guest, "")
}

func (self *GuestCreateBackupTask) OnGuestStartCompletedFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_CREATE_BACKUP_FAILED, data.String(), self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_CREATE_BACKUP, data.String(), self.UserCred, false)
	self.SetStageFailed(ctx, data)
}

func (self *GuestCreateBackupTask) TaskCompleted(ctx context.Context, guest *models.SGuest, reason string) {
	db.OpsLog.LogEvent(guest, db.ACT_CREATE_BACKUP, reason, self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_CREATE_BACKUP, reason, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
	guest.StartSyncstatus(ctx, self.UserCred, "")
}

func (self *GuestCreateBackupTask) TaskFailed(ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject) {
	guest.SetStatus(self.UserCred, api.VM_BACKUP_CREATE_FAILED, reason.String())
	db.OpsLog.LogEvent(guest, db.ACT_CREATE_BACKUP_FAILED, reason, self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_CREATE_BACKUP, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func init() {
	taskman.RegisterTask(GuestSwitchToBackupTask{})
	taskman.RegisterTask(GuestStartAndSyncToBackupTask{})
	taskman.RegisterTask(GuestCreateBackupTask{})
	taskman.RegisterTask(GuestReSyncToBackup{})
}

type GuestReSyncToBackup struct {
	GuestStartAndSyncToBackupTask
}

func (self *GuestReSyncToBackup) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.StartSyncToBackup(ctx, guest)
}

func (self *GuestReSyncToBackup) StartSyncToBackup(ctx context.Context, guest *models.SGuest) {
	data := jsonutils.NewDict()
	nbdServerPort, _ := self.Params.Int("nbd_server_port")
	data.Set("nbd_server_port", jsonutils.NewInt(nbdServerPort))
	self.OnStartBackupGuest(ctx, guest, data)
}
