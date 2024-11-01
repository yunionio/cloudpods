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
	"database/sql"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/cgrouputils/cpuset"
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

func (task *GuestMigrateTask) isLiveMigrate() bool {
	guestStatus, _ := task.Params.GetString("guest_status")
	if !task.isRescueMode() && (guestStatus == api.VM_RUNNING || guestStatus == api.VM_SUSPEND) {
		return true
	}
	return false
}

func (task *GuestMigrateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	StartScheduleObjects(ctx, task, []db.IStandaloneModel{obj})
}

func (task *GuestMigrateTask) GetSchedParams() (*schedapi.ScheduleInput, error) {
	obj := task.GetObject()
	guest := obj.(*models.SGuest)
	input := new(api.ServerMigrateForecastInput)
	if task.Params.Contains("prefer_host_id") {
		preferHostId, _ := task.Params.GetString("prefer_host_id")
		input.PreferHostId = preferHostId
	}
	if jsonutils.QueryBoolean(task.Params, "reset_cpu_numa_pin", false) {
		input.ResetCpuNumaPin = true
	}

	if task.isLiveMigrate() {
		input.LiveMigrate = true
		skipCpuCheck := jsonutils.QueryBoolean(task.Params, "skip_cpu_check", false)
		skipKernelCheck := jsonutils.QueryBoolean(task.Params, "skip_kernel_check", false)
		input.SkipCpuCheck = skipCpuCheck
		input.SkipKernelCheck = skipKernelCheck
	}
	res := guest.GetSchedMigrateParams(task.GetUserCred(), input)

	if devs, _ := guest.GetIsolatedDevices(); len(devs) > 0 {
		preferNumaNodesSet := cpuset.NewBuilder()
		for i := range devs {
			if devs[i].NumaNode >= 0 {
				preferNumaNodesSet.Add(int(devs[i].NumaNode))
			}
		}
		res.PreferNumaNodes = preferNumaNodesSet.Result().ToSlice()
	}
	return res, nil
}

func (task *GuestMigrateTask) OnStartSchedule(obj IScheduleModel) {
	guest := obj.(*models.SGuest)
	guestStatus, _ := task.Params.GetString("guest_status")
	if guestStatus != api.VM_RUNNING && guestStatus != api.VM_SUSPEND {
		guest.SetStatus(context.Background(), task.UserCred, api.VM_MIGRATING, "")
	}

	db.OpsLog.LogEvent(guest, db.ACT_MIGRATING, "", task.UserCred)
}

func (task *GuestMigrateTask) OnScheduleFailCallback(ctx context.Context, obj IScheduleModel, reason jsonutils.JSONObject, index int) {
	// do nothing
}

func (task *GuestMigrateTask) OnScheduleFailed(ctx context.Context, reason jsonutils.JSONObject) {
	obj := task.GetObject()
	guest := obj.(*models.SGuest)
	task.TaskFailed(ctx, guest, reason)
}

func (task *GuestMigrateTask) SaveScheduleResult(ctx context.Context, obj IScheduleModel, target *schedapi.CandidateResource, index int) {
	guest := obj.(*models.SGuest)
	if jsonutils.QueryBoolean(task.Params, "reset_cpu_numa_pin", false) {
		guest.SetCpuNumaPin(ctx, task.UserCred, target.CpuNumaPin, nil)
		db.OpsLog.LogEvent(guest, db.ACT_RESET_CPU_NUMA_PIN, fmt.Sprintf("reset cpu numa pin %s", jsonutils.Marshal(target.CpuNumaPin)), task.UserCred)
		task.SetStageComplete(ctx, nil)
		return
	}

	targetHostId := target.HostId
	targetHost := models.HostManager.FetchHostById(targetHostId)
	if targetHost == nil {
		task.TaskFailed(ctx, guest, jsonutils.NewString("target host not found?"))
		return
	}

	body := jsonutils.NewDict()
	body.Set("target_host_id", jsonutils.NewString(targetHostId))
	if len(target.CpuNumaPin) > 0 {
		body.Set("target_cpu_numa_pin", jsonutils.Marshal(target.CpuNumaPin))
	}

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

	// prepare disk for migration
	if len(disk.TemplateId) > 0 && isLocalStorage {
		templates := []string{}
		if sourceGuestId := guest.GetMetadata(ctx, api.SERVER_META_CONVERT_FROM_ESXI, task.UserCred); len(sourceGuestId) > 0 {
			// skip cache images
		} else if sourceGuestId := guest.GetMetadata(ctx, api.SERVER_META_CONVERT_FROM_CLOUDPODS, task.UserCred); len(sourceGuestId) > 0 {
			// skip cache images
		} else {
			guestdisks, _ := guest.GetDisks()
			for i := range guestdisks {
				if guestdisks[i].TemplateId != "" {
					templates = append(templates, guestdisks[i].TemplateId)
				}
			}
		}

		if len(templates) > 0 {
			body.Set("cache_templates", jsonutils.NewStringArray(templates))
		}
	}

	db.OpsLog.LogEvent(guest, db.ACT_MIGRATING, fmt.Sprintf("guest start migrate from host %s to %s", guest.HostId, targetHostId), task.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_MIGRATING,
		fmt.Sprintf("guest start migrate from host %s to %s(%s)", guest.HostId, targetHostId, targetHost.GetName()), task.UserCred, true)

	task.SetStage("OnStartCacheImages", body)
	task.OnStartCacheImages(ctx, guest, nil)
}

func (task *GuestMigrateTask) tryRecoverImageCache(ctx context.Context, guest *models.SGuest, input *api.CacheImageInput) error {
	if _, err := models.CachedimageManager.FetchById(input.ImageId); err != nil {
		if err != sql.ErrNoRows {
			return err
		}
		if _, err := models.CachedimageManager.RecoverCachedImage(ctx, task.UserCred, input.ImageId); err != nil {
			log.Errorf("failed recache image %s: %s", input.ImageId, err)
		}

		srcHost, err := guest.GetHost()
		if err != nil {
			return err
		}
		srcStorageCache := srcHost.GetLocalStoragecache()
		if scImg := models.StoragecachedimageManager.GetStoragecachedimage(srcStorageCache.Id, input.ImageId); scImg == nil {
			_, err = models.StoragecachedimageManager.RecoverStoragecachedImage(ctx, task.UserCred, srcStorageCache.Id, input.ImageId)
			if err != nil {
				log.Errorf("failed RecoverStoragecachedImage %s:%s %s", srcStorageCache.Id, input.ImageId, err)
			}
		}
	}
	return nil
}

func (task *GuestMigrateTask) OnStartCacheImages(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	templates, _ := task.Params.GetArray("cache_templates")
	if len(templates) == 0 {
		task.OnCachedImageComplete(ctx, guest, nil)
		return
	}

	templateId, _ := templates[0].GetString()
	task.Params.Set("cache_templates", jsonutils.NewArray(templates[1:]...))
	task.SetStage("OnStartCacheImages", nil)

	targetHostId, _ := task.Params.GetString("target_host_id")
	targetHost := models.HostManager.FetchHostById(targetHostId)
	targetStorageCache := targetHost.GetLocalStoragecache()
	if targetStorageCache != nil {
		input := api.CacheImageInput{
			ImageId:      templateId,
			IsForce:      false,
			SourceHostId: guest.HostId,
			ParentTaskId: task.GetTaskId(),
		}
		if err := task.tryRecoverImageCache(ctx, guest, &input); err != nil {
			task.TaskFailed(ctx, guest, jsonutils.NewString(err.Error()))
			return
		}

		err := targetStorageCache.StartImageCacheTask(ctx, task.UserCred, input)
		if err != nil {
			task.TaskFailed(ctx, guest, jsonutils.NewString(err.Error()))
			return
		}
	} else {
		task.OnStartCacheImages(ctx, guest, nil)
	}
}

func (task *GuestMigrateTask) OnStartCacheImagesFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.TaskFailed(ctx, guest, data)
}

// For local storage get disk info
func (task *GuestMigrateTask) OnCachedImageComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.SetStage("OnCachedCdromComplete", nil)
	isLocalStorage, _ := task.Params.Bool("is_local_storage")
	if cdrom := guest.GetCdrom(); cdrom != nil && len(cdrom.ImageId) > 0 && isLocalStorage {
		targetHostId, _ := task.Params.GetString("target_host_id")
		targetHost := models.HostManager.FetchHostById(targetHostId)
		targetStorageCache := targetHost.GetLocalStoragecache()
		if targetStorageCache != nil {
			input := api.CacheImageInput{
				ImageId:      cdrom.ImageId,
				Format:       "iso",
				IsForce:      false,
				ParentTaskId: task.GetTaskId(),
				SourceHostId: guest.HostId,
			}
			if err := task.tryRecoverImageCache(ctx, guest, &input); err != nil {
				task.TaskFailed(ctx, guest, jsonutils.NewString(err.Error()))
				return
			}
			err := targetStorageCache.StartImageCacheTask(ctx, task.UserCred, input)
			if err != nil {
				task.TaskFailed(ctx, guest, jsonutils.NewString(err.Error()))
				return
			}
		}
	} else {
		task.OnCachedCdromComplete(ctx, guest, nil)
	}
}

func (task *GuestMigrateTask) OnCachedCdromComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	header := task.GetTaskRequestHeader()
	body := jsonutils.NewDict()
	if task.isLiveMigrate() {
		body.Set("live_migrate", jsonutils.JSONTrue)
		body.Set("enable_tls", jsonutils.NewBool(jsonutils.QueryBoolean(task.GetParams(), "enable_tls", false)))
	}

	if !task.isRescueMode() {
		host, _ := guest.GetHost()
		url := fmt.Sprintf("%s/servers/%s/src-prepare-migrate", host.ManagerUri, guest.Id)
		task.SetStage("OnSrcPrepareComplete", body)
		_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST",
			url, header, body, false)
		if err != nil {
			task.TaskFailed(ctx, guest, jsonutils.NewString(err.Error()))
			return
		}
	} else {
		task.OnSrcPrepareComplete(ctx, guest, nil)
	}
}

func (task *GuestMigrateTask) OnCachedCdromCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.TaskFailed(ctx, guest, data)
}

func (task *GuestMigrateTask) OnCachedImageCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.TaskFailed(ctx, guest, data)
}

func (task *GuestMigrateTask) OnSrcPrepareCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.TaskFailed(ctx, guest, data)
}

func (task *GuestMigrateTask) OnSrcPrepareComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	targetHostId, _ := task.Params.GetString("target_host_id")
	targetHost := models.HostManager.FetchHostById(targetHostId)
	var body *jsonutils.JSONDict
	var err error
	if jsonutils.QueryBoolean(task.Params, "is_local_storage", false) {
		body, err = task.localStorageMigrateConf(ctx, guest, targetHost, data)
	} else {
		body, err = task.sharedStorageMigrateConf(ctx, guest, targetHost)
	}
	if err != nil {
		task.TaskFailed(ctx, guest, jsonutils.NewString(errors.Wrap(err, "get storage migrate conf").Error()))
		return
	}

	if task.isLiveMigrate() {
		srcDesc, err := data.Get("src_desc")
		if err != nil {
			task.TaskFailed(ctx, guest, jsonutils.NewString(errors.Wrap(err, "get src_desc from data").Error()))
			return
		}
		body.Set("src_desc", srcDesc)
		body.Set("live_migrate", jsonutils.JSONTrue)
	}
	if jsonutils.QueryBoolean(task.GetParams(), "enable_tls", false) {
		body.Set("enable_tls", jsonutils.JSONTrue)
		certsObj, err := data.Get("migrate_certs")
		if err != nil {
			task.TaskFailed(ctx, guest, jsonutils.NewString(errors.Wrap(err, "get migrate_certs from data").Error()))
			return
		}
		body.Set("migrate_certs", certsObj)
	}
	if err != nil {
		task.TaskFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}

	headers := task.GetTaskRequestHeader()
	url := fmt.Sprintf("%s/servers/%s/dest-prepare-migrate", targetHost.ManagerUri, guest.Id)
	task.SetStage("OnMigrateConfAndDiskComplete", nil)
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(),
		ctx, "POST", url, headers, body, false)
	if err != nil {
		task.TaskFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (task *GuestMigrateTask) OnMigrateConfAndDiskCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	targetHostId, _ := task.Params.GetString("target_host_id")
	err := jsonutils.NewDict()
	err.Set("MigrateConfAndDiskFailedReason", data)
	task.SetStage("OnUndeployTargetGuestSucc", err)
	guest.StartUndeployGuestTask(ctx, task.UserCred, task.GetTaskId(), targetHostId)
	task.TaskFailed(ctx, guest, data)
}

func (task *GuestMigrateTask) OnUndeployTargetGuestSucc(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	err, _ := task.Params.Get("MigrateConfAndDiskFailedReason")
	task.TaskFailed(ctx, guest, err)
}

func (task *GuestMigrateTask) OnUndeployTargetGuestSuccFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	prevErr, _ := task.Params.Get("MigrateConfAndDiskFailedReason")
	err := jsonutils.NewDict()
	err.Set("MigrateConfAndDiskFailedReason", prevErr)
	err.Set("UndeployTargetGuestFailedReason", data)
	task.TaskFailed(ctx, guest, err)
}

func (task *GuestMigrateTask) OnMigrateConfAndDiskComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if data.Contains("dest_prepared_memory_snapshots") {
		msData, _ := data.Get("dest_prepared_memory_snapshots")
		task.Params.Set("dest_prepared_memory_snapshots", msData)
	}
	if task.isLiveMigrate() {
		// Live migrate
		task.SetStage("OnStartDestComplete", nil)
	} else {
		// Normal migrate
		task.OnNormalMigrateComplete(ctx, guest, data)
	}
}

func (task *GuestMigrateTask) OnNormalMigrateComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	oldHostId := guest.HostId
	task.setGuest(ctx, guest)
	guestStatus, _ := task.Params.GetString("guest_status")
	guest.SetStatus(ctx, task.UserCred, guestStatus, "")
	if task.isRescueMode() {
		guest.StartGueststartTask(ctx, task.UserCred, nil, "")
		task.TaskComplete(ctx, guest)
	} else {
		task.SetStage("OnUndeployOldHostSucc", nil)
		guest.StartUndeployGuestTask(ctx, task.UserCred, task.GetTaskId(), oldHostId)
	}
}

// Server migrate complete
func (task *GuestMigrateTask) OnUndeployOldHostSucc(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if jsonutils.QueryBoolean(task.Params, "auto_start", false) {
		task.SetStage("OnGuestStartSucc", nil)
		guest.StartGueststartTask(ctx, task.UserCred, nil, task.GetId())
	} else {
		task.TaskComplete(ctx, guest)
	}
}

func (task *GuestMigrateTask) OnUndeployOldHostSuccFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.TaskFailed(ctx, guest, data)
}

func (task *GuestMigrateTask) OnGuestStartSucc(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.TaskComplete(ctx, guest)
}

func (task *GuestMigrateTask) OnGuestStartSuccFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.TaskFailed(ctx, guest, data)
}

func (task *GuestMigrateTask) isRescueMode() bool {
	return jsonutils.QueryBoolean(task.Params, "is_rescue_mode", false)
}

func (task *GuestMigrateTask) getInstanceSnapShotsWithMemory(guest *models.SGuest) ([]*models.SInstanceSnapshot, error) {
	isps, err := guest.GetInstanceSnapshots()
	if err != nil {
		return nil, errors.Wrap(err, "GetInstanceSnapshots")
	}
	ret := make([]*models.SInstanceSnapshot, 0)
	for idx := range isps {
		if isps[idx].WithMemory {
			if task.isRescueMode() {
				// do not copy memory snapshot in rescure mode, as it is not accessible
				// remove memory flag, because the memory snapshot will be lost after migration
				db.Update(&isps[idx], func() error {
					isps[idx].WithMemory = false
					return nil
				})
			} else {
				ret = append(ret, &isps[idx])
			}
		}
	}
	return ret, nil
}

func (task *GuestMigrateTask) getInstanceSnapShotIdsWithMemory(guest *models.SGuest) (*jsonutils.JSONArray, error) {
	isps, err := task.getInstanceSnapShotsWithMemory(guest)
	if err != nil {
		return nil, errors.Wrap(err, "getInstanceSnapshotsWithMemory")
	}
	ret := []string{}
	for _, isp := range isps {
		ret = append(ret, isp.GetId())
	}
	return jsonutils.Marshal(ret).(*jsonutils.JSONArray), nil
}

func (task *GuestMigrateTask) setBodyMemorySnapshotParams(guest *models.SGuest, srcHost *models.SHost, body *jsonutils.JSONDict) error {
	isps, err := task.getInstanceSnapShotIdsWithMemory(guest)
	if err != nil {
		return errors.Wrap(err, "getInstanceSnapShotsWithMemory")
	}
	memSnapshotUri := fmt.Sprintf("%s/download/memory_snapshots", srcHost.ManagerUri)
	body.Set("memory_snapshots_uri", jsonutils.NewString(memSnapshotUri))
	body.Set("src_memory_snapshots", isps)
	return nil
}

func (task *GuestMigrateTask) sharedStorageMigrateConf(ctx context.Context, guest *models.SGuest, targetHost *models.SHost) (*jsonutils.JSONDict, error) {
	body := jsonutils.NewDict()
	body.Set("is_local_storage", jsonutils.JSONFalse)
	body.Set("qemu_version", jsonutils.NewString(guest.GetQemuVersion(task.UserCred)))
	targetDesc := guest.GetJsonDescAtHypervisor(ctx, targetHost)
	if task.Params.Contains("target_cpu_numa_pin") {
		if err := task.setCpuNumaPin(targetDesc); err != nil {
			return nil, errors.Wrap(err, "setCpuNumaPin")
		}
	}

	body.Set("desc", jsonutils.Marshal(targetDesc))

	sourceHost, _ := guest.GetHost()
	if err := task.setBodyMemorySnapshotParams(guest, sourceHost, body); err != nil {
		return nil, errors.Wrap(err, "setBodyMemorySnapshotParams")
	}
	return body, nil
}

func (task *GuestMigrateTask) localStorageMigrateConf(ctx context.Context,
	guest *models.SGuest, targetHost *models.SHost, data jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	body := jsonutils.NewDict()
	if data != nil {
		body.Update(data.(*jsonutils.JSONDict))
	}
	params := jsonutils.NewDict()
	disks, _ := guest.GetGuestDisks()
	for i := 0; i < len(disks); i++ {
		snapChain := []string{}
		if body.Contains("disk_snaps_chain", disks[i].DiskId) {
			err := body.Unmarshal(&snapChain, "disk_snaps_chain", disks[i].DiskId)
			if err != nil {
				return nil, errors.Wrap(err, "unmarshal snap chain")
			}
		}
		snapshots := models.SnapshotManager.GetDiskSnapshots(disks[i].DiskId)
		outChainSnapshotIds := jsonutils.NewArray()
		for j := 0; j < len(snapshots); j++ {
			if !utils.IsInStringArray(snapshots[j].Id, snapChain) {
				outChainSnapshotIds.Add(jsonutils.NewString(snapshots[j].Id))
			}
		}
		params.Set(disks[i].DiskId, outChainSnapshotIds)
	}

	sourceHost, _ := guest.GetHost()
	snapshotsUri := fmt.Sprintf("%s/download/snapshots/", sourceHost.ManagerUri)
	disksUri := fmt.Sprintf("%s/download/disks/", sourceHost.ManagerUri)
	serverUrl := fmt.Sprintf("%s/download/servers/%s", sourceHost.ManagerUri, guest.Id)

	body.Set("out_chain_snapshots", params)
	body.Set("snapshots_uri", jsonutils.NewString(snapshotsUri))
	body.Set("disks_uri", jsonutils.NewString(disksUri))
	body.Set("server_url", jsonutils.NewString(serverUrl))
	body.Set("qemu_version", jsonutils.NewString(guest.GetQemuVersion(task.UserCred)))

	if err := task.setBodyMemorySnapshotParams(guest, sourceHost, body); err != nil {
		return nil, errors.Wrap(err, "setBodyMemorySnapshotParams")
	}

	targetDesc := guest.GetJsonDescAtHypervisor(ctx, targetHost)
	if len(targetDesc.Disks) == 0 {
		return nil, errors.Errorf("Get disksDesc error")
	}
	if task.Params.Contains("target_cpu_numa_pin") {
		if err := task.setCpuNumaPin(targetDesc); err != nil {
			return nil, errors.Wrap(err, "setCpuNumaPin")
		}
	}

	targetStorages, _ := task.Params.GetArray("target_storages")
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

func (task *GuestMigrateTask) setCpuNumaPin(targetDesc *api.GuestJsonDesc) error {
	cpuNumaPin := make([]schedapi.SCpuNumaPin, 0)
	if err := task.Params.Unmarshal(&cpuNumaPin, "cpu_numa_pin"); err != nil {
		return errors.Wrap(err, "unmarshal cpu_numa_pin")
	}
	for i := range targetDesc.CpuNumaPin {
		for j := range targetDesc.CpuNumaPin[i].VcpuPin {
			targetDesc.CpuNumaPin[i].VcpuPin[j].Pcpu = cpuNumaPin[i].CpuPin[j]
		}
	}
	task.Params.Set("target_vcpu_numa_pin", jsonutils.Marshal(targetDesc.CpuNumaPin))
	return nil
}

func (task *GuestLiveMigrateTask) OnStartDestComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	liveMigrateDestPort, err := data.Get("live_migrate_dest_port")
	if err != nil {
		task.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("Get migrate port error: %s", err)))
		return
	}

	var body = jsonutils.NewDict()
	var nbdServerPort jsonutils.JSONObject
	if !jsonutils.QueryBoolean(data, "nbd_server_disabled", false) {
		nbdServerPort, err = data.Get("nbd_server_port")
		if err != nil {
			task.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("Get nbd server port error: %s", err)))
			return
		}
		body.Set("nbd_server_port", nbdServerPort)
	}

	targetHostId, _ := task.Params.GetString("target_host_id")
	targetHost := models.HostManager.FetchHostById(targetHostId)
	isLocalStorage, _ := task.Params.Get("is_local_storage")
	body.Set("is_local_storage", isLocalStorage)
	body.Set("live_migrate_dest_port", liveMigrateDestPort)
	body.Set("dest_ip", jsonutils.NewString(targetHost.AccessIp))
	body.Set("enable_tls", jsonutils.NewBool(jsonutils.QueryBoolean(task.GetParams(), "enable_tls", false)))
	body.Set("quickly_finish", jsonutils.NewBool(jsonutils.QueryBoolean(task.GetParams(), "quickly_finish", false)))
	if task.Params.Contains("max_bandwidth_mb") {
		maxBandwidthMb, _ := task.Params.Get("max_bandwidth_mb")
		body.Set("max_bandwidth_mb", maxBandwidthMb)
	}

	headers := task.GetTaskRequestHeader()
	host, _ := guest.GetHost()
	url := fmt.Sprintf("%s/servers/%s/live-migrate", host.ManagerUri, guest.Id)
	task.SetStage("OnLiveMigrateComplete", nil)
	guest.SetStatus(ctx, task.UserCred, api.VM_LIVE_MIGRATING, "")
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(),
		ctx, "POST", url, headers, body, false)
	if err != nil {
		task.OnLiveMigrateCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (task *GuestLiveMigrateTask) OnStartDestCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if !jsonutils.QueryBoolean(task.Params, "keep_dest_guest_on_failed", false) {
		targetHostId, _ := task.Params.GetString("target_host_id")
		guest.StartUndeployGuestTask(ctx, task.UserCred, "", targetHostId)
	}
	task.TaskFailed(ctx, guest, data)
}

func (task *GuestMigrateTask) setGuest(ctx context.Context, guest *models.SGuest) error {
	targetHostId, _ := task.Params.GetString("target_host_id")
	if jsonutils.QueryBoolean(task.Params, "is_local_storage", false) {
		targetStorages, _ := task.Params.GetArray("target_storages")
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

	if task.Params.Contains("target_cpu_numa_pin") {
		var cpuNumaPinSrc []schedapi.SCpuNumaPin = nil
		var cpuNumaPin []api.SCpuNumaPin = nil

		val, _ := task.Params.Get("target_cpu_numa_pin")
		if !val.Equals(jsonutils.JSONNull) {
			cpuNumaPinSrc = make([]schedapi.SCpuNumaPin, 0)
			if err := task.Params.Unmarshal(&cpuNumaPinSrc, "target_cpu_numa_pin"); err != nil {
				return errors.Wrap(err, "unmarshal target_cpu_numa_pin")
			}

			cpuNumaPin = make([]api.SCpuNumaPin, 0)
			if err := task.Params.Unmarshal(&cpuNumaPin, "target_vcpu_numa_pin"); err != nil {
				return errors.Wrap(err, "unmarshal target_vcpu_numa_pin")
			}
		}

		if err := guest.SetCpuNumaPin(ctx, task.UserCred, cpuNumaPinSrc, cpuNumaPin); err != nil {
			return errors.Wrap(err, "SetCpuNumaPin")
		}
	}

	oldHost, _ := guest.GetHost()
	oldHost.ClearSchedDescCache()
	err := guest.OnScheduleToHost(ctx, task.UserCred, targetHostId)
	if err != nil {
		return err
	}
	return nil
}

func (task *GuestLiveMigrateTask) OnLiveMigrateCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if reason, _ := data.GetString("__reason__"); reason == "cancelled" {
		task.Params.Set("migrate_cancelled", jsonutils.JSONTrue)
	}

	if !jsonutils.QueryBoolean(task.Params, "keep_dest_guest_on_failed", false) {
		targetHostId, _ := task.Params.GetString("target_host_id")
		task.SetStage("OnGuestUndeployed", nil)
		guest.StartUndeployGuestTask(ctx, task.UserCred, task.Id, targetHostId)
	} else {
		task.OnGuestUndeployed(ctx, guest, data)
	}
}

func (task *GuestLiveMigrateTask) OnGuestUndeployed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.TaskFailed(ctx, guest, data)
	if jsonutils.QueryBoolean(task.Params, "migrate_cancelled", false) {
		guest.StartSyncstatus(ctx, task.UserCred, "")
	}
}

func (task *GuestLiveMigrateTask) OnGuestUndeployedFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.TaskFailed(ctx, guest, data)
}

func (task *GuestLiveMigrateTask) OnLiveMigrateComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if migInfo, err := data.Get("migration_info"); err != nil {
		db.OpsLog.LogEvent(guest, db.ACT_MIGRATE, migInfo, task.UserCred)
		logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_MIGRATE, migInfo, task.UserCred, true)
	}

	headers := task.GetTaskRequestHeader()
	body := jsonutils.NewDict()
	body.Set("live_migrate", jsonutils.JSONTrue)
	body.Set("clean_tls", jsonutils.NewBool(jsonutils.QueryBoolean(task.GetParams(), "enable_tls", false)))
	targetHostId, _ := task.Params.GetString("target_host_id")

	task.SetStage("OnResumeDestGuestComplete", nil)
	targetHost := models.HostManager.FetchHostById(targetHostId)
	url := fmt.Sprintf("%s/servers/%s/resume", targetHost.ManagerUri, guest.Id)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(),
		ctx, "POST", url, headers, body, false)
	if err != nil {
		task.OnResumeDestGuestCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (task *GuestLiveMigrateTask) OnResumeDestGuestCompleteFailed(ctx context.Context,
	guest *models.SGuest, data jsonutils.JSONObject) {

	task.markFailed(ctx, guest, data)
	if !jsonutils.QueryBoolean(task.Params, "keep_dest_guest_on_failed", false) {
		targetHostId, _ := task.Params.GetString("target_host_id")
		guest.StartUndeployGuestTask(ctx, task.UserCred, "", targetHostId)
	}

	task.SetStage("OnResumeSourceGuestComplete", nil)
	sourceHost := models.HostManager.FetchHostById(guest.HostId)
	headers := task.GetTaskRequestHeader()
	body := jsonutils.NewDict()
	url := fmt.Sprintf("%s/servers/%s/resume", sourceHost.ManagerUri, guest.Id)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(),
		ctx, "POST", url, headers, body, false)
	if err != nil {
		task.OnResumeSourceGuestCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (task *GuestLiveMigrateTask) OnResumeSourceGuestCompleteFailed(ctx context.Context,
	guest *models.SGuest, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_RESUME_FAIL, data, task.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_VM_RESUME, data, task.UserCred, false)
	task.TaskFailed(ctx, guest, data)
}

func (task *GuestLiveMigrateTask) OnResumeSourceGuestComplete(ctx context.Context,
	guest *models.SGuest, data jsonutils.JSONObject) {
	task.TaskFailed(ctx, guest, data)
}

func (task *GuestLiveMigrateTask) OnResumeDestGuestComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	oldHostId := guest.HostId
	err := task.setGuest(ctx, guest)
	if err != nil {
		task.TaskFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
	task.SetStage("OnUndeploySrcGuestComplete", nil)
	err = guest.StartUndeployGuestTask(ctx, task.UserCred, task.GetTaskId(), oldHostId)
	if err != nil {
		task.TaskFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (task *GuestLiveMigrateTask) OnUndeploySrcGuestComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	db.OpsLog.LogEvent(guest, db.ACT_MIGRATE, "OnUndeploySrcGuestComplete", task.UserCred)
	status, _ := task.Params.GetString("guest_status")
	if status != guest.Status {
		task.SetStage("OnGuestSyncStatus", nil)
		guest.StartSyncstatus(ctx, task.UserCred, task.GetTaskId())
	} else {
		task.OnGuestSyncStatus(ctx, guest, nil)
	}
}

// Server live migrate complete
func (task *GuestLiveMigrateTask) OnGuestSyncStatus(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.TaskComplete(ctx, guest)
}

func (task *GuestMigrateTask) updateInstanceSnapshotMemory(ctx context.Context, guest *models.SGuest) error {
	if !task.Params.Contains("dest_prepared_memory_snapshots") {
		return nil
	}
	ms, err := task.Params.Get("dest_prepared_memory_snapshots")
	if err != nil {
		return errors.Wrap(err, "get dest_prepared_memory_snapshots from params")
	}
	isps, err := task.getInstanceSnapShotsWithMemory(guest)
	if err != nil {
		return errors.Wrap(err, "getInstanceSnapShotsWithMemory")
	}
	for _, isp := range isps {
		msPath, err := ms.GetString(isp.GetId())
		if err != nil {
			return errors.Wrapf(err, "get instance snapshot %s memory path from dest prepared", isp.GetId())
		}
		if _, err := db.Update(isp, func() error {
			isp.MemoryFilePath = msPath
			isp.MemoryFileHostId = guest.HostId
			return nil
		}); err != nil {
			return errors.Wrapf(err, "update instance snapshot %q memory_filie_path", isp.GetId())
		}
	}
	return nil
}

func (task *GuestMigrateTask) TaskComplete(ctx context.Context, guest *models.SGuest) {
	if err := task.updateInstanceSnapshotMemory(ctx, guest); err != nil {
		task.TaskFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	task.SetStageComplete(ctx, nil)
	db.OpsLog.LogEvent(guest, db.ACT_MIGRATE, "Migrate success", task.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_MIGRATE, task.Params, task.UserCred, true)
}

func (task *GuestMigrateTask) TaskFailed(ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject) {
	task.markFailed(ctx, guest, reason)
	task.SetStageFailed(ctx, reason)
}

func (task *GuestMigrateTask) markFailed(ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject) {
	guest.SetStatus(ctx, task.UserCred, api.VM_MIGRATE_FAILED, reason.String())
	db.OpsLog.LogEvent(guest, db.ACT_MIGRATE_FAIL, reason, task.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_MIGRATE, reason, task.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, guest.Id, guest.Name, api.VM_MIGRATE_FAILED, reason.String())
	notifyclient.EventNotify(ctx, task.GetUserCred(), notifyclient.SEventNotifyParam{
		Obj:    guest,
		Action: notifyclient.ActionMigrate,
		IsFail: true,
	})
}

// ManagedGuestMigrateTask
func (task *ManagedGuestMigrateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_MIGRATING, nil, task.UserCred)
	task.MigrateStart(ctx, guest, data)
}

func (task *ManagedGuestMigrateTask) MigrateStart(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.SetStage("OnMigrateComplete", nil)
	guest.SetStatus(ctx, task.UserCred, api.VM_MIGRATING, "")
	input := api.GuestMigrateInput{}
	task.GetParams().Unmarshal(&input)
	drv, err := guest.GetDriver()
	if err != nil {
		task.OnMigrateCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	if err := drv.RequestMigrate(ctx, guest, task.UserCred, input, task); err != nil {
		task.OnMigrateCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (task *ManagedGuestMigrateTask) OnMigrateComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_MIGRATE, task.Params, task.UserCred, true)
	db.OpsLog.LogEvent(guest, db.ACT_MIGRATE, guest.GetShortDesc(ctx), task.UserCred)
	if jsonutils.QueryBoolean(task.Params, "auto_start", false) {
		task.SetStage("OnGuestStartSucc", nil)
		guest.StartGueststartTask(ctx, task.UserCred, nil, task.GetId())
	} else {
		task.SetStage("OnGuestSyncStatus", nil)
		guest.StartSyncstatus(ctx, task.UserCred, task.GetTaskId())
	}
}

func (task *ManagedGuestMigrateTask) OnGuestStartSucc(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.SetStageComplete(ctx, nil)
}

func (task *ManagedGuestMigrateTask) OnGuestSyncStatus(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.SetStageComplete(ctx, nil)
}

func (task *ManagedGuestMigrateTask) OnGuestSyncStatusFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.SetStageFailed(ctx, data)
}

func (task *ManagedGuestMigrateTask) OnGuestStartSuccFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.SetStageFailed(ctx, data)
}

func (task *ManagedGuestMigrateTask) OnMigrateCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetStatus(ctx, task.UserCred, api.VM_MIGRATE_FAILED, "")
	db.OpsLog.LogEvent(guest, db.ACT_MIGRATE_FAIL, data, task.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_MIGRATE, data, task.UserCred, false)
	task.SetStageFailed(ctx, data)
	notifyclient.NotifySystemErrorWithCtx(ctx, guest.Id, guest.Name, api.VM_MIGRATE_FAILED, data.String())
}

// ManagedGuestLiveMigrateTask
func (task *ManagedGuestLiveMigrateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_MIGRATING, nil, task.UserCred)
	task.MigrateStart(ctx, guest, data)
}

func (task *ManagedGuestLiveMigrateTask) MigrateStart(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.SetStage("OnMigrateComplete", nil)
	guest.SetStatus(ctx, task.UserCred, api.VM_MIGRATING, "")
	input := api.GuestLiveMigrateInput{}
	task.GetParams().Unmarshal(&input)
	drv, err := guest.GetDriver()
	if err != nil {
		task.OnMigrateCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	if err := drv.RequestLiveMigrate(ctx, guest, task.UserCred, input, task); err != nil {
		task.OnMigrateCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (task *ManagedGuestLiveMigrateTask) OnMigrateComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.SetStage("OnGuestSyncStatus", nil)
	db.OpsLog.LogEvent(guest, db.ACT_MIGRATE, guest.GetShortDesc(ctx), task.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_MIGRATE, task.Params, task.UserCred, true)
	guest.StartSyncstatus(ctx, task.UserCred, task.GetTaskId())
}

func (task *ManagedGuestLiveMigrateTask) OnGuestSyncStatus(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.SetStageComplete(ctx, nil)
}

func (task *ManagedGuestLiveMigrateTask) OnGuestSyncStatusFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.SetStageFailed(ctx, data)
}

func (task *ManagedGuestLiveMigrateTask) OnMigrateCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetStatus(ctx, task.UserCred, api.VM_MIGRATE_FAILED, "")
	db.OpsLog.LogEvent(guest, db.ACT_MIGRATE_FAIL, data, task.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_MIGRATE, data, task.UserCred, false)
	task.SetStageFailed(ctx, data)
	notifyclient.NotifySystemErrorWithCtx(ctx, guest.Id, guest.Name, api.VM_MIGRATE_FAILED, data.String())
}
