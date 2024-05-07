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
func (task *GuestSwitchToBackupTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	host, _ := guest.GetHost()
	task.Params.Set("is_force", jsonutils.JSONTrue)
	task.SetStage("OnEnsureMasterGuestStoped", nil)
	drv, err := guest.GetDriver()
	if err != nil {
		task.OnEnsureMasterGuestStoped(ctx, guest, nil)
	}
	err = drv.RequestStopOnHost(ctx, guest, host, task, true)
	if err != nil {
		// In case of master host crash
		task.OnEnsureMasterGuestStoped(ctx, guest, nil)
	}
}

func (task *GuestSwitchToBackupTask) OnEnsureMasterGuestStoped(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	backupHost := models.HostManager.FetchHostById(guest.BackupHostId)
	task.Params.Set("is_force", jsonutils.JSONTrue)
	task.SetStage("OnBackupGuestStoped", nil)
	drv, err := guest.GetDriver()
	if err != nil {
		task.OnFail(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	err = drv.RequestStopOnHost(ctx, guest, backupHost, task, true)
	if err != nil {
		task.OnFail(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (task *GuestSwitchToBackupTask) OnBackupGuestStoped(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	disks, err := guest.GetDisks()
	if err != nil {
		task.OnFail(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	for i := 0; i < len(disks); i++ {
		disk := disks[i]
		err := disk.SwitchToBackup(task.UserCred)
		if err != nil {
			if i > 0 {
				for j := 0; j < i; j++ {
					disk = disks[j]
					disk.SwitchToBackup(task.UserCred)
				}
			}
			task.OnFail(ctx, guest, jsonutils.NewString(fmt.Sprintf("Switch to backup disk error: %s", err)))
			return
		}
	}
	err = guest.SwitchToBackup(task.UserCred)
	if err != nil {
		task.OnFail(ctx, guest, jsonutils.NewString(fmt.Sprintf("Switch to backup guest error: %s", err)))
		return
	}

	db.OpsLog.LogEvent(guest, db.ACT_SWITCHED, "Switch to backup", task.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_SWITCH_TO_BACKUP, "Switch to backup", task.UserCred, true)
	oldStatus, _ := task.Params.GetString("old_status")
	autoStart := jsonutils.QueryBoolean(task.Params, "auto_start", false) ||
		utils.IsInStringArray(oldStatus, api.VM_RUNNING_STATUS)
	if autoStart {
		task.SetStage("OnGuestStartCompleted", nil)
		if err := guest.StartGueststartTask(ctx, task.UserCred, nil, task.GetId()); err != nil {
			task.OnGuestStartCompletedFailed(ctx, guest,
				jsonutils.NewString(fmt.Sprintf("start guest start task: %s", err)))
		}
	} else {
		task.OnComplete(ctx, guest, nil)
	}
}

func (task *GuestSwitchToBackupTask) OnFail(ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject) {
	guest.SetStatus(ctx, task.UserCred, api.VM_SWITCH_TO_BACKUP_FAILED, reason.String())
	db.OpsLog.LogEvent(guest, db.ACT_SWITCH_FAILED, reason, task.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_SWITCH_TO_BACKUP, reason, task.UserCred, false)
	task.SetStageFailed(ctx, reason)
}

func (task *GuestSwitchToBackupTask) OnGuestStartCompleted(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.SetStageComplete(ctx, nil)
}

func (task *GuestSwitchToBackupTask) OnGuestStartCompletedFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.SetStageFailed(ctx, data)
}

func (task *GuestSwitchToBackupTask) OnComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.SetStageComplete(ctx, nil)
}

/********************* GuestStartAndSyncToBackupTask *********************/

type GuestStartAndSyncToBackupTask struct {
	SGuestBaseTask
}

func (task *GuestStartAndSyncToBackupTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(ctx, task.UserCred, api.VM_BACKUP_STARTING, "GuestStartAndSyncToBackupTask")
	task.SetStage("OnCheckTemplete", nil)
	task.checkTemplete(ctx, guest)
}

func (task *GuestStartAndSyncToBackupTask) checkTemplete(ctx context.Context, guest *models.SGuest) {
	diskCat := guest.CategorizeDisks()
	if diskCat.Root != nil && len(diskCat.Root.GetTemplateId()) > 0 {
		drv, err := guest.GetDriver()
		if err != nil {
			task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
			return
		}
		err = drv.CheckDiskTemplateOnStorage(ctx, task.UserCred, diskCat.Root.GetTemplateId(), diskCat.Root.DiskFormat,
			diskCat.Root.BackupStorageId, task)
		if err != nil {
			task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		}
	} else {
		task.OnCheckTemplete(ctx, guest, nil)
	}
}

func (task *GuestStartAndSyncToBackupTask) OnCheckTemplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.SetStage("OnStartBackupGuest", nil)
	host := models.HostManager.FetchHostById(guest.BackupHostId)

	if !guest.IsGuestBackupMirrorJobReady(ctx, task.UserCred) {
		hostMaster := models.HostManager.FetchHostById(guest.HostId)
		task.Params.Set("block_ready", jsonutils.JSONFalse)
		diskUri := fmt.Sprintf("%s/disks", hostMaster.GetFetchUrl(true))
		task.Params.Set("disk_uri", jsonutils.NewString(diskUri))
	} else {
		task.Params.Set("block_ready", jsonutils.JSONTrue)
	}
	drv, err := guest.GetDriver()
	if err != nil {
		task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
	err = drv.RequestStartOnHost(ctx, guest, host, task.UserCred, task)
	if err != nil {
		task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
	}
}

func (task *GuestStartAndSyncToBackupTask) OnStartBackupGuest(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetBackupGuestStatus(task.UserCred, api.VM_RUNNING, "on start backup guest")
	nbdServerPort, err := data.Int("nbd_server_port")
	if err != nil {
		task.SetStageFailed(ctx, jsonutils.NewString("Start Backup Guest Missing Nbd Port"))
		return
	}
	backupHost := models.HostManager.FetchHostById(guest.BackupHostId)
	nbdServerUri := fmt.Sprintf("nbd:%s:%d", backupHost.AccessIp, nbdServerPort)
	guest.SetMetadata(ctx, "backup_nbd_server_uri", nbdServerUri, task.UserCred)
	db.OpsLog.LogEvent(guest, db.ACT_BACKUP_START, guest.GetShortDesc(ctx), task.UserCred)

	// try get origin guest status
	guestStatus, err := task.Params.GetString("guest_status")
	if err != nil {
		guestStatus = guest.Status
	}

	if utils.IsInStringArray(guestStatus, api.VM_RUNNING_STATUS) {
		task.SetStage("OnRequestSyncToBackup", nil)
		drv, err := guest.GetDriver()
		if err != nil {
			task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
			return
		}
		err = drv.RequestSyncToBackup(ctx, guest, task)
		if err != nil {
			task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		}
	} else {
		task.onComplete(ctx, guest)
	}
}

func (task *GuestStartAndSyncToBackupTask) OnStartBackupGuestFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetGuestBackupMirrorJobFailed(ctx, task.UserCred)
	db.OpsLog.LogEvent(guest, db.ACT_BACKUP_START_FAILED, data.String(), task.UserCred)
	task.SetStageFailed(ctx, data)
}

func (task *GuestStartAndSyncToBackupTask) OnRequestSyncToBackup(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetGuestBackupMirrorJobInProgress(ctx, task.UserCred)
	drv, err := guest.GetDriver()
	if err != nil {
		guest.SetGuestBackupMirrorJobFailed(ctx, task.UserCred)
		guest.SetBackupGuestStatus(task.UserCred, api.VM_BLOCK_STREAM_FAIL, err.Error())
		task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
	err = drv.RequestSlaveBlockStreamDisks(ctx, guest, task)
	if err != nil {
		guest.SetGuestBackupMirrorJobFailed(ctx, task.UserCred)
		guest.SetBackupGuestStatus(task.UserCred, api.VM_BLOCK_STREAM_FAIL, err.Error())
		task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}

	guest.SetGuestBackupMirrorJobInProgress(ctx, task.UserCred)
	guest.SetBackupGuestStatus(task.UserCred, api.VM_BLOCK_STREAM, "OnSyncToBackup")
	task.onComplete(ctx, guest)
}

func (task *GuestStartAndSyncToBackupTask) onComplete(ctx context.Context, guest *models.SGuest) {
	guestStatus, _ := task.Params.GetString("guest_status")
	guest.SetStatus(ctx, task.UserCred, guestStatus, "on GuestStartAndSyncToBackupTask completed")
	task.SetStageComplete(ctx, nil)
}

func (task *GuestStartAndSyncToBackupTask) OnRequestSyncToBackupFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetGuestBackupMirrorJobFailed(ctx, task.UserCred)
	guest.SetStatus(ctx, task.UserCred, api.VM_BLOCK_STREAM_FAIL, "OnSyncToBackup")
	task.SetStageFailed(ctx, data)
}

type GuestCreateBackupTask struct {
	SSchedTask
}

func (task *GuestCreateBackupTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	StartScheduleObjects(ctx, task, []db.IStandaloneModel{obj})
}

func (task *GuestCreateBackupTask) OnStartSchedule(obj IScheduleModel) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_START_CREATE_BACKUP, "", task.UserCred)
}

func (task *GuestCreateBackupTask) GetSchedParams() (*schedapi.ScheduleInput, error) {
	obj := task.GetObject()
	guest := obj.(*models.SGuest)
	schedDesc := guest.ToSchedDesc()
	if task.Params.Contains("prefer_host_id") {
		preferHostId, _ := task.Params.GetString("prefer_host_id")
		schedDesc.ServerConfig.PreferHost = preferHostId
	}
	schedDesc.ReuseNetwork = true
	return schedDesc, nil
}

func (task *GuestCreateBackupTask) OnScheduleFailCallback(ctx context.Context, obj IScheduleModel, reason jsonutils.JSONObject, index int) {
	// do nothing
}

func (task *GuestCreateBackupTask) OnScheduleFailed(ctx context.Context, reason jsonutils.JSONObject) {
	obj := task.GetObject()
	guest := obj.(*models.SGuest)
	task.TaskFailed(ctx, guest, reason)
}

func (task *GuestCreateBackupTask) SaveScheduleResult(ctx context.Context, obj IScheduleModel, candidate *schedapi.CandidateResource, index int) {
	guest := obj.(*models.SGuest)
	targetHostId := candidate.HostId
	targetHost := models.HostManager.FetchHostById(candidate.HostId)
	if targetHost == nil {
		task.TaskFailed(ctx, guest, jsonutils.NewString("target host not found?"))
		return
	}
	guest.SetHostIdWithBackup(task.UserCred, guest.HostId, targetHostId)
	db.OpsLog.LogEvent(guest, db.ACT_CREATE_BACKUP, fmt.Sprintf("guest backup start create on host %s", targetHostId), task.UserCred)

	task.StartCreateBackupDisks(ctx, guest, targetHost, candidate.Disks)
}

func (task *GuestCreateBackupTask) StartCreateBackupDisks(ctx context.Context, guest *models.SGuest, host *models.SHost, candidateDisks []*schedapi.CandidateDisk) {
	guestDisks, _ := guest.GetGuestDisks()
	for i := 0; i < len(guestDisks); i++ {
		var candidateDisk *schedapi.CandidateDisk
		if len(candidateDisks) >= i {
			candidateDisk = candidateDisks[i]
		}
		diskConfig := &api.DiskConfig{Backend: api.STORAGE_LOCAL}
		storage, err := guest.ChooseHostStorage(host, diskConfig, candidateDisk)
		if err != nil {
			task.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("unable to ChooseHostStorage: %v", err)))
			return
		}
		if storage == nil {
			task.TaskFailed(ctx, guest, jsonutils.NewString("Get backup storage error"))
			return
		}
		disk := guestDisks[i].GetDisk()
		db.Update(disk, func() error {
			disk.BackupStorageId = storage.Id
			return nil
		})
	}
	task.SetStage("OnCreateBackupDisks", nil)
	err := guest.CreateBackupDisks(ctx, task.UserCred, task.GetTaskId())
	if err != nil {
		task.TaskFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (task *GuestCreateBackupTask) StartInsertIso(ctx context.Context, guest *models.SGuest, imageId string) {
	task.SetStage("OnInsertIso", nil)
	guest.StartInsertIsoTask(ctx, 0, imageId, false, nil, guest.BackupHostId, task.UserCred, task.GetTaskId())
}

func (task *GuestCreateBackupTask) OnInsertIso(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.SetStage("OnCreateBackup", nil)
	guest.StartCreateBackup(ctx, task.UserCred, task.GetTaskId(), nil)
}

func (task *GuestCreateBackupTask) OnInsertIsoFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("Backup guest insert ISO failed %s", data.String())))
}

func (task *GuestCreateBackupTask) OnCreateBackupDisks(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if cdrom := guest.GetCdrom(); cdrom != nil && len(cdrom.ImageId) > 0 {
		task.StartInsertIso(ctx, guest, cdrom.ImageId)
	} else {
		task.SetStage("OnCreateBackup", nil)
		guest.StartCreateBackup(ctx, task.UserCred, task.GetTaskId(), nil)
	}
}

func (task *GuestCreateBackupTask) OnCreateBackupDisksFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.TaskFailed(ctx, guest, data)
}

func (task *GuestCreateBackupTask) OnCreateBackupFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.TaskFailed(ctx, guest, data)
}

func (task *GuestCreateBackupTask) OnCreateBackup(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetBackupGuestStatus(task.UserCred, api.VM_READY, "on create backup")
	guestStatus, _ := task.Params.GetString("guest_status")
	if utils.IsInStringArray(guestStatus, api.VM_RUNNING_STATUS) {
		task.OnGuestStart(ctx, guest, guestStatus)
	} else if jsonutils.QueryBoolean(task.Params, "auto_start", false) {
		task.RequestStartGuest(ctx, guest)
	} else {
		task.TaskCompleted(ctx, guest, "")
	}
}

func (task *GuestCreateBackupTask) OnGuestStart(ctx context.Context, guest *models.SGuest, guestStatus string) {
	task.SetStage("OnSyncToBackup", nil)
	err := guest.GuestStartAndSyncToBackup(ctx, task.UserCred, task.GetTaskId(), guestStatus)
	if err != nil {
		task.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("Guest sycn to backup error %s", err.Error())))
	}
}

func (task *GuestCreateBackupTask) OnSyncToBackup(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.TaskCompleted(ctx, guest, "")
}

func (task *GuestCreateBackupTask) OnSyncToBackupFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.TaskFailed(ctx, guest, data)
}

func (task *GuestCreateBackupTask) RequestStartGuest(ctx context.Context, guest *models.SGuest) {
	task.SetStage("OnGuestStartCompleted", nil)
	guest.StartGueststartTask(ctx, task.UserCred, nil, task.GetId())
}

func (task *GuestCreateBackupTask) OnGuestStartCompleted(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.TaskCompleted(ctx, guest, "")
}

func (task *GuestCreateBackupTask) OnGuestStartCompletedFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_CREATE_BACKUP_FAILED, data.String(), task.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_CREATE_BACKUP, data.String(), task.UserCred, false)
	task.SetStageFailed(ctx, data)
}

func (task *GuestCreateBackupTask) TaskCompleted(ctx context.Context, guest *models.SGuest, reason string) {
	db.OpsLog.LogEvent(guest, db.ACT_CREATE_BACKUP, reason, task.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_CREATE_BACKUP, reason, task.UserCred, true)
	task.SetStageComplete(ctx, nil)
	guest.StartSyncstatus(ctx, task.UserCred, "")
}

func (task *GuestCreateBackupTask) TaskFailed(ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject) {
	guest.SetStatus(ctx, task.UserCred, api.VM_BACKUP_CREATE_FAILED, reason.String())
	db.OpsLog.LogEvent(guest, db.ACT_CREATE_BACKUP_FAILED, reason, task.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_CREATE_BACKUP, reason, task.UserCred, false)
	task.SetStageFailed(ctx, reason)
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

func (task *GuestReSyncToBackup) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	task.StartSyncToBackup(ctx, guest)
}

func (task *GuestReSyncToBackup) StartSyncToBackup(ctx context.Context, guest *models.SGuest) {
	data := jsonutils.NewDict()
	nbdServerPort, _ := task.Params.Int("nbd_server_port")
	data.Set("nbd_server_port", jsonutils.NewInt(nbdServerPort))
	task.OnStartBackupGuest(ctx, guest, data)
}
