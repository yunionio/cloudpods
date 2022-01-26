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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestMigrateTask struct {
	SSchedTask
}

type GuestLiveMigrateTask struct {
	GuestMigrateTask
}
type ManagedGuestMigrateTask struct {
	SGuestBaseTask
}

type ManagedGuestLiveMigrateTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestLiveMigrateTask{})
	taskman.RegisterTask(GuestMigrateTask{})
	taskman.RegisterTask(ManagedGuestMigrateTask{})
	taskman.RegisterTask(ManagedGuestLiveMigrateTask{})
}

func (self *GuestMigrateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	StartScheduleObjects(ctx, self, []db.IStandaloneModel{obj})
}

func (self *GuestMigrateTask) GetSchedParams() (*schedapi.ScheduleInput, error) {
	obj := self.GetObject()
	guest := obj.(*models.SGuest)
	input := new(api.ServerMigrateForecastInput)
	if self.Params.Contains("prefer_host_id") {
		preferHostId, _ := self.Params.GetString("prefer_host_id")
		input.PreferHostId = preferHostId
	}
	guestStatus, _ := self.Params.GetString("guest_status")
	if !jsonutils.QueryBoolean(self.Params, "is_rescue_mode", false) && (guestStatus == api.VM_RUNNING || guestStatus == api.VM_SUSPEND) {
		input.LiveMigrate = true
		skipCpuCheck := jsonutils.QueryBoolean(self.Params, "skip_cpu_check", false)
		input.SkipCpuCheck = skipCpuCheck
	}
	return guest.GetSchedMigrateParams(self.GetUserCred(), input), nil
}

func (self *GuestMigrateTask) OnStartSchedule(obj IScheduleModel) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.UserCred, api.VM_MIGRATING, "")
	db.OpsLog.LogEvent(guest, db.ACT_MIGRATING, "", self.UserCred)
}

func (self *GuestMigrateTask) OnScheduleFailCallback(ctx context.Context, obj IScheduleModel, reason jsonutils.JSONObject) {
	// do nothing
}

func (self *GuestMigrateTask) OnScheduleFailed(ctx context.Context, reason jsonutils.JSONObject) {
	obj := self.GetObject()
	guest := obj.(*models.SGuest)
	self.TaskFailed(ctx, guest, reason)
}

func (self *GuestMigrateTask) SaveScheduleResult(ctx context.Context, obj IScheduleModel, target *schedapi.CandidateResource) {
	targetHostId := target.HostId
	guest := obj.(*models.SGuest)
	targetHost := models.HostManager.FetchHostById(targetHostId)
	if targetHost == nil {
		self.TaskFailed(ctx, guest, jsonutils.NewString("target host not found?"))
		return
	}
	db.OpsLog.LogEvent(guest, db.ACT_MIGRATING, fmt.Sprintf("guest start migrate from host %s to %s", guest.HostId, targetHostId), self.UserCred)

	body := jsonutils.NewDict()
	body.Set("target_host_id", jsonutils.NewString(targetHostId))
	// for params notes
	body.Set("target_host_name", jsonutils.NewString(targetHost.Name))
	srcHost := models.HostManager.FetchHostById(guest.HostId)
	body.Set("source_host_name", jsonutils.NewString(srcHost.Name))
	body.Set("source_host_id", jsonutils.NewString(srcHost.Id))

	disks, _ := guest.GetGuestDisks()
	disk := disks[0].GetDisk()
	storage, _ := disk.GetStorage()
	isLocalStorage := utils.IsInStringArray(storage.StorageType,
		api.STORAGE_LOCAL_TYPES)
	if isLocalStorage {
		targetStorages := jsonutils.NewArray()
		for i := 0; i < len(disks); i++ {
			var targetStroage string
			if len(target.Disks[i].StorageIds) == 0 {
				targetStroage = targetHost.GetLeastUsedStorage(storage.StorageType).Id
			} else {
				targetStroage = target.Disks[i].StorageIds[0]
			}
			targetStorages.Add(jsonutils.NewString(targetStroage))
		}
		body.Set("target_storages", targetStorages)
		body.Set("is_local_storage", jsonutils.JSONTrue)
	} else {
		body.Set("is_local_storage", jsonutils.JSONFalse)
	}

	self.SetStage("OnCachedImageComplete", body)
	// prepare disk for migration
	if len(disk.TemplateId) > 0 && isLocalStorage {
		targetStorageCache := targetHost.GetLocalStoragecache()
		if targetStorageCache != nil {
			input := api.CacheImageInput{
				ImageId:      disk.TemplateId,
				Format:       disk.DiskFormat,
				IsForce:      false,
				SourceHostId: guest.HostId,
				ParentTaskId: self.GetTaskId(),
			}
			err := targetStorageCache.StartImageCacheTask(ctx, self.UserCred, input)
			if err != nil {
				self.TaskFailed(ctx, guest, jsonutils.NewString(err.Error()))
			}
			return
		}
	}
	self.OnCachedImageComplete(ctx, guest, nil)
}

// For local storage get disk info
func (self *GuestMigrateTask) OnCachedImageComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStage("OnCachedCdromComplete", nil)
	isLocalStorage, _ := self.Params.Bool("is_local_storage")
	if cdrom := guest.GetCdrom(); cdrom != nil && len(cdrom.ImageId) > 0 && isLocalStorage {
		targetHostId, _ := self.Params.GetString("target_host_id")
		targetHost := models.HostManager.FetchHostById(targetHostId)
		targetStorageCache := targetHost.GetLocalStoragecache()
		if targetStorageCache != nil {
			input := api.CacheImageInput{
				ImageId:      cdrom.ImageId,
				Format:       "iso",
				IsForce:      false,
				ParentTaskId: self.GetTaskId(),
			}
			err := targetStorageCache.StartImageCacheTask(ctx, self.UserCred, input)
			if err != nil {
				self.TaskFailed(ctx, guest, jsonutils.NewString(err.Error()))
			}
			return
		}
	}
	self.OnCachedCdromComplete(ctx, guest, nil)
}

func (self *GuestMigrateTask) OnCachedCdromComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	header := self.GetTaskRequestHeader()
	body := jsonutils.NewDict()
	guestStatus, _ := self.Params.GetString("guest_status")
	if !jsonutils.QueryBoolean(self.Params, "is_rescue_mode", false) && (guestStatus == api.VM_RUNNING || guestStatus == api.VM_SUSPEND) {
		body.Set("live_migrate", jsonutils.JSONTrue)
		body.Set("enable_tls", jsonutils.NewBool(jsonutils.QueryBoolean(self.GetParams(), "enable_tls", false)))
	}

	if !jsonutils.QueryBoolean(self.Params, "is_rescue_mode", false) {
		host, _ := guest.GetHost()
		url := fmt.Sprintf("%s/servers/%s/src-prepare-migrate", host.ManagerUri, guest.Id)
		self.SetStage("OnSrcPrepareComplete", body)
		_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST",
			url, header, body, false)
		if err != nil {
			self.TaskFailed(ctx, guest, jsonutils.NewString(err.Error()))
			return
		}
	} else {
		self.OnSrcPrepareComplete(ctx, guest, nil)
	}
}

func (self *GuestMigrateTask) OnCachedCdromCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, guest, data)
}

func (self *GuestMigrateTask) OnCachedImageCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, guest, data)
}

func (self *GuestMigrateTask) OnSrcPrepareCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, guest, data)
}

func (self *GuestMigrateTask) OnSrcPrepareComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	targetHostId, _ := self.Params.GetString("target_host_id")
	targetHost := models.HostManager.FetchHostById(targetHostId)
	var body *jsonutils.JSONDict
	var err error
	if jsonutils.QueryBoolean(self.Params, "is_local_storage", false) {
		body, err = self.localStorageMigrateConf(ctx, guest, targetHost, data)
	} else {
		body, err = self.sharedStorageMigrateConf(ctx, guest, targetHost)
	}
	if jsonutils.QueryBoolean(self.GetParams(), "enable_tls", false) {
		body.Set("enable_tls", jsonutils.JSONTrue)
		certsObj, err := data.Get("migrate_certs")
		if err != nil {
			self.TaskFailed(ctx, guest, jsonutils.NewString(errors.Wrap(err, "get migrate_certs from data").Error()))
			return
		}
		body.Set("migrate_certs", certsObj)
	}
	if err != nil {
		self.TaskFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	guestStatus, _ := self.Params.GetString("guest_status")
	if !jsonutils.QueryBoolean(self.Params, "is_rescue_mode", false) && (guestStatus == api.VM_RUNNING || guestStatus == api.VM_SUSPEND) {
		body.Set("live_migrate", jsonutils.JSONTrue)
	}

	headers := self.GetTaskRequestHeader()

	url := fmt.Sprintf("%s/servers/%s/dest-prepare-migrate", targetHost.ManagerUri, guest.Id)
	self.SetStage("OnMigrateConfAndDiskComplete", body)
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(),
		ctx, "POST", url, headers, body, false)
	if err != nil {
		self.TaskFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (self *GuestMigrateTask) OnMigrateConfAndDiskCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	targetHostId, _ := self.Params.GetString("target_host_id")
	err := jsonutils.NewDict()
	err.Set("MigrateConfAndDiskFailedReason", data)
	self.SetStage("OnUndeployTargetGuestSucc", err)
	guest.StartUndeployGuestTask(ctx, self.UserCred, self.GetTaskId(), targetHostId)
	self.TaskFailed(ctx, guest, data)
}

func (self *GuestMigrateTask) OnUndeployTargetGuestSucc(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	err, _ := self.Params.Get("MigrateConfAndDiskFailedReason")
	self.TaskFailed(ctx, guest, err)
}

func (self *GuestMigrateTask) OnUndeployTargetGuestSuccFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	prevErr, _ := self.Params.Get("MigrateConfAndDiskFailedReason")
	err := jsonutils.NewDict()
	err.Set("MigrateConfAndDiskFailedReason", prevErr)
	err.Set("UndeployTargetGuestFailedReason", data)
	self.TaskFailed(ctx, guest, err)
}

func (self *GuestMigrateTask) OnMigrateConfAndDiskComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guestStatus, _ := self.Params.GetString("guest_status")
	if !jsonutils.QueryBoolean(self.Params, "is_rescue_mode", false) && (guestStatus == api.VM_RUNNING || guestStatus == api.VM_SUSPEND) {
		// Live migrate
		self.SetStage("OnStartDestComplete", nil)
	} else {
		// Normal migrate
		self.OnNormalMigrateComplete(ctx, guest, data)
	}
}

func (self *GuestMigrateTask) OnNormalMigrateComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	oldHostId := guest.HostId
	self.setGuest(ctx, guest)
	guestStatus, _ := self.Params.GetString("guest_status")
	guest.SetStatus(self.UserCred, guestStatus, "")
	if jsonutils.QueryBoolean(self.Params, "is_rescue_mode", false) {
		guest.StartGueststartTask(ctx, self.UserCred, nil, "")
		self.TaskComplete(ctx, guest)
	} else {
		self.SetStage("OnUndeployOldHostSucc", nil)
		guest.StartUndeployGuestTask(ctx, self.UserCred, self.GetTaskId(), oldHostId)
	}
}

// Server migrate complete
func (self *GuestMigrateTask) OnUndeployOldHostSucc(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if jsonutils.QueryBoolean(self.Params, "auto_start", false) {
		self.SetStage("OnGuestStartSucc", nil)
		guest.StartGueststartTask(ctx, self.UserCred, nil, self.GetId())
	} else {
		self.TaskComplete(ctx, guest)
	}
}

func (self *GuestMigrateTask) OnUndeployOldHostSuccFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, guest, data)
}

func (self *GuestMigrateTask) OnGuestStartSucc(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.TaskComplete(ctx, guest)
}

func (self *GuestMigrateTask) OnGuestStartSuccFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, guest, data)
}

func (self *GuestMigrateTask) sharedStorageMigrateConf(ctx context.Context, guest *models.SGuest, targetHost *models.SHost) (*jsonutils.JSONDict, error) {
	body := jsonutils.NewDict()
	body.Set("is_local_storage", jsonutils.JSONFalse)
	body.Set("qemu_version", jsonutils.NewString(guest.GetQemuVersion(self.UserCred)))
	body.Set("qemu_cmdline", jsonutils.NewString(guest.GetQemuCmdline(self.UserCred)))
	targetDesc := guest.GetJsonDescAtHypervisor(ctx, targetHost)
	body.Set("desc", jsonutils.Marshal(targetDesc))
	return body, nil
}

func (self *GuestMigrateTask) localStorageMigrateConf(ctx context.Context,
	guest *models.SGuest, targetHost *models.SHost, data jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	body := jsonutils.NewDict()
	if data != nil {
		body.Update(data.(*jsonutils.JSONDict))
	}
	params := jsonutils.NewDict()
	disks, _ := guest.GetGuestDisks()
	for i := 0; i < len(disks); i++ {
		snapshots := models.SnapshotManager.GetDiskSnapshots(disks[i].DiskId)
		snapshotIds := jsonutils.NewArray()
		for j := 0; j < len(snapshots); j++ {
			snapshotIds.Add(jsonutils.NewString(snapshots[j].Id))
		}
		params.Set(disks[i].DiskId, snapshotIds)
	}

	sourceHost, _ := guest.GetHost()
	snapshotsUri := fmt.Sprintf("%s/download/snapshots/", sourceHost.ManagerUri)
	disksUri := fmt.Sprintf("%s/download/disks/", sourceHost.ManagerUri)
	serverUrl := fmt.Sprintf("%s/download/servers/%s", sourceHost.ManagerUri, guest.Id)

	body.Set("src_snapshots", params)
	body.Set("snapshots_uri", jsonutils.NewString(snapshotsUri))
	body.Set("disks_uri", jsonutils.NewString(disksUri))
	body.Set("server_url", jsonutils.NewString(serverUrl))
	body.Set("qemu_version", jsonutils.NewString(guest.GetQemuVersion(self.UserCred)))
	body.Set("qemu_cmdline", jsonutils.NewString(guest.GetQemuCmdline(self.UserCred)))
	targetDesc := guest.GetJsonDescAtHypervisor(ctx, targetHost)
	if len(targetDesc.Disks) == 0 {
		return nil, errors.Errorf("Get disksDesc error")
	}
	targetStorages, _ := self.Params.GetArray("target_storages")
	for i := 0; i < len(disks); i++ {
		targetStorageId, err := targetStorages[i].GetString()
		if err != nil {
			return nil, errors.Wrapf(err, "Get disk %d target storage id", i)
		}
		targetDesc.Disks[i].TargetStorageId = targetStorageId
	}

	body.Set("desc", jsonutils.Marshal(targetDesc))
	body.Set("rebase_disks", jsonutils.JSONTrue)
	body.Set("is_local_storage", jsonutils.JSONTrue)
	return body, nil
}

func (self *GuestLiveMigrateTask) OnStartDestComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	liveMigrateDestPort, err := data.Get("live_migrate_dest_port")
	if err != nil {
		self.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("Get migrate port error: %s", err)))
		return
	}

	targetHostId, _ := self.Params.GetString("target_host_id")
	targetHost := models.HostManager.FetchHostById(targetHostId)

	body := jsonutils.NewDict()
	isLocalStorage, _ := self.Params.Get("is_local_storage")
	body.Set("is_local_storage", isLocalStorage)
	body.Set("live_migrate_dest_port", liveMigrateDestPort)
	body.Set("dest_ip", jsonutils.NewString(targetHost.AccessIp))
	body.Set("enable_tls", jsonutils.NewBool(jsonutils.QueryBoolean(self.GetParams(), "enable_tls", false)))

	headers := self.GetTaskRequestHeader()

	host, _ := guest.GetHost()
	url := fmt.Sprintf("%s/servers/%s/live-migrate", host.ManagerUri, guest.Id)
	self.SetStage("OnLiveMigrateComplete", nil)
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(),
		ctx, "POST", url, headers, body, false)
	if err != nil {
		self.OnLiveMigrateCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (self *GuestLiveMigrateTask) OnStartDestCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	targetHostId, _ := self.Params.GetString("target_host_id")
	guest.StartUndeployGuestTask(ctx, self.UserCred, "", targetHostId)
	self.TaskFailed(ctx, guest, data)
}

func (self *GuestMigrateTask) setGuest(ctx context.Context, guest *models.SGuest) error {
	targetHostId, _ := self.Params.GetString("target_host_id")
	if jsonutils.QueryBoolean(self.Params, "is_local_storage", false) {
		targetStorages, _ := self.Params.GetArray("target_storages")
		disks, _ := guest.GetDisks()
		for i := 0; i < len(disks); i++ {
			disk := &disks[i]
			db.Update(disk, func() error {
				disk.Status = api.DISK_READY
				disk.StorageId, _ = targetStorages[i].GetString()
				return nil
			})
			snapshots := models.SnapshotManager.GetDiskSnapshots(disk.Id)
			for _, snapshot := range snapshots {
				db.Update(&snapshot, func() error {
					snapshot.StorageId, _ = targetStorages[i].GetString()
					return nil
				})
			}
		}
	}
	oldHost, _ := guest.GetHost()
	oldHost.ClearSchedDescCache()
	err := guest.OnScheduleToHost(ctx, self.UserCred, targetHostId)
	if err != nil {
		return err
	}
	return nil
}

func (self *GuestLiveMigrateTask) OnLiveMigrateCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	targetHostId, _ := self.Params.GetString("target_host_id")
	guest.StartUndeployGuestTask(ctx, self.UserCred, "", targetHostId)
	self.TaskFailed(ctx, guest, data)
}

func (self *GuestLiveMigrateTask) OnLiveMigrateComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	headers := self.GetTaskRequestHeader()
	body := jsonutils.NewDict()
	body.Set("live_migrate", jsonutils.JSONTrue)
	body.Set("clean_tls", jsonutils.NewBool(jsonutils.QueryBoolean(self.GetParams(), "enable_tls", false)))
	targetHostId, _ := self.Params.GetString("target_host_id")

	self.SetStage("OnResumeDestGuestComplete", nil)
	targetHost := models.HostManager.FetchHostById(targetHostId)
	url := fmt.Sprintf("%s/servers/%s/resume", targetHost.ManagerUri, guest.Id)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(),
		ctx, "POST", url, headers, body, false)
	if err != nil {
		self.OnResumeDestGuestCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (self *GuestLiveMigrateTask) OnResumeDestGuestCompleteFailed(ctx context.Context,
	guest *models.SGuest, data jsonutils.JSONObject) {
	targetHostId, _ := self.Params.GetString("target_host_id")

	self.markFailed(ctx, guest, data)

	guest.StartUndeployGuestTask(ctx, self.UserCred, "", targetHostId)

	self.SetStage("OnResumeSourceGuestComplete", nil)
	sourceHost := models.HostManager.FetchHostById(guest.HostId)
	headers := self.GetTaskRequestHeader()
	body := jsonutils.NewDict()
	url := fmt.Sprintf("%s/servers/%s/resume", sourceHost.ManagerUri, guest.Id)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(),
		ctx, "POST", url, headers, body, false)
	if err != nil {
		self.OnResumeSourceGuestCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (self *GuestLiveMigrateTask) OnResumeSourceGuestCompleteFailed(ctx context.Context,
	guest *models.SGuest, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_RESUME_FAIL, data, self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_VM_RESUME, data, self.UserCred, false)
	self.TaskFailed(ctx, guest, data)
}

func (self *GuestLiveMigrateTask) OnResumeSourceGuestComplete(ctx context.Context,
	guest *models.SGuest, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, guest, data)
}

func (self *GuestLiveMigrateTask) OnResumeDestGuestComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	oldHostId := guest.HostId
	err := self.setGuest(ctx, guest)
	if err != nil {
		self.TaskFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
	self.SetStage("OnUndeploySrcGuestComplete", nil)
	err = guest.StartUndeployGuestTask(ctx, self.UserCred, self.GetTaskId(), oldHostId)
	if err != nil {
		self.TaskFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (self *GuestLiveMigrateTask) OnUndeploySrcGuestComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_MIGRATE, "OnUndeploySrcGuestComplete", self.UserCred)
	status, _ := self.Params.GetString("guest_status")
	if status != guest.Status {
		self.SetStage("OnGuestSyncStatus", nil)
		guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
	} else {
		self.OnGuestSyncStatus(ctx, guest, nil)
	}
}

// Server live migrate complete
func (self *GuestLiveMigrateTask) OnGuestSyncStatus(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.TaskComplete(ctx, guest)
}

func (self *GuestMigrateTask) TaskComplete(ctx context.Context, guest *models.SGuest) {
	self.SetStageComplete(ctx, nil)
	db.OpsLog.LogEvent(guest, db.ACT_MIGRATE, "Migrate success", self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_MIGRATE, self.Params, self.UserCred, true)
}

func (self *GuestMigrateTask) TaskFailed(ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject) {
	self.markFailed(ctx, guest, reason)
	self.SetStageFailed(ctx, reason)
}

func (self *GuestMigrateTask) markFailed(ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject) {
	guest.SetStatus(self.UserCred, api.VM_MIGRATE_FAILED, reason.String())
	db.OpsLog.LogEvent(guest, db.ACT_MIGRATE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_MIGRATE, reason, self.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, guest.Id, guest.Name, api.VM_MIGRATE_FAILED, reason.String())
	notifyclient.EventNotify(ctx, self.GetUserCred(), notifyclient.SEventNotifyParam{
		Obj:    guest,
		Action: notifyclient.ActionMigrate,
		IsFail: true,
	})
}

//ManagedGuestMigrateTask
func (self *ManagedGuestMigrateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_MIGRATING, nil, self.UserCred)
	self.MigrateStart(ctx, guest, data)
}

func (self *ManagedGuestMigrateTask) MigrateStart(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStage("OnMigrateComplete", nil)
	guest.SetStatus(self.UserCred, api.VM_MIGRATING, "")
	if err := guest.GetDriver().RequestMigrate(ctx, guest, self.UserCred, self.GetParams(), self); err != nil {
		self.OnMigrateCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (self *ManagedGuestMigrateTask) OnMigrateComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_MIGRATE, self.Params, self.UserCred, true)
	db.OpsLog.LogEvent(guest, db.ACT_MIGRATE, guest.GetShortDesc(ctx), self.UserCred)
	if jsonutils.QueryBoolean(self.Params, "auto_start", false) {
		self.SetStage("OnGuestStartSucc", nil)
		guest.StartGueststartTask(ctx, self.UserCred, nil, self.GetId())
	} else {
		self.SetStage("OnGuestSyncStatus", nil)
		guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
	}
}

func (self *ManagedGuestMigrateTask) OnGuestStartSucc(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *ManagedGuestMigrateTask) OnGuestSyncStatus(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *ManagedGuestMigrateTask) OnGuestSyncStatusFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}

func (self *ManagedGuestMigrateTask) OnGuestStartSuccFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}

func (self *ManagedGuestMigrateTask) OnMigrateCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetStatus(self.UserCred, api.VM_MIGRATE_FAILED, "")
	db.OpsLog.LogEvent(guest, db.ACT_MIGRATE_FAIL, data, self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_MIGRATE, data, self.UserCred, false)
	self.SetStageFailed(ctx, data)
	notifyclient.NotifySystemErrorWithCtx(ctx, guest.Id, guest.Name, api.VM_MIGRATE_FAILED, data.String())
}

//ManagedGuestLiveMigrateTask
func (self *ManagedGuestLiveMigrateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_MIGRATING, nil, self.UserCred)
	self.MigrateStart(ctx, guest, data)
}

func (self *ManagedGuestLiveMigrateTask) MigrateStart(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStage("OnMigrateComplete", nil)
	guest.SetStatus(self.UserCred, api.VM_MIGRATING, "")
	if err := guest.GetDriver().RequestLiveMigrate(ctx, guest, self.UserCred, self.GetParams(), self); err != nil {
		self.OnMigrateCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (self *ManagedGuestLiveMigrateTask) OnMigrateComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStage("OnGuestSyncStatus", nil)
	db.OpsLog.LogEvent(guest, db.ACT_MIGRATE, guest.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_MIGRATE, self.Params, self.UserCred, true)
	guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
}

func (self *ManagedGuestLiveMigrateTask) OnGuestSyncStatus(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *ManagedGuestLiveMigrateTask) OnGuestSyncStatusFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}

func (self *ManagedGuestLiveMigrateTask) OnMigrateCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetStatus(self.UserCred, api.VM_MIGRATE_FAILED, "")
	db.OpsLog.LogEvent(guest, db.ACT_MIGRATE_FAIL, data, self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_MIGRATE, data, self.UserCred, false)
	self.SetStageFailed(ctx, data)
	notifyclient.NotifySystemErrorWithCtx(ctx, guest.Id, guest.Name, api.VM_MIGRATE_FAILED, data.String())
}
