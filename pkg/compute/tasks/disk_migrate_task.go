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

	"yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DiskMigrateTask struct {
	SSchedTask
}

func init() {
	taskman.RegisterTask(DiskMigrateTask{})
}

func (task *DiskMigrateTask) TaskComplete(ctx context.Context, disk *models.SDisk) {
	task.SetStageComplete(ctx, nil)
	db.OpsLog.LogEvent(disk, db.ACT_MIGRATE, "Migrate success", task.UserCred)
	logclient.AddActionLogWithContext(ctx, disk, logclient.ACT_MIGRATE, task.Params, task.UserCred, true)
}

func (task *DiskMigrateTask) markFailed(ctx context.Context, disk *models.SDisk, reason jsonutils.JSONObject) {
	disk.SetStatus(ctx, task.UserCred, compute.DISK_MIGRATE_FAIL, reason.String())
	db.OpsLog.LogEvent(disk, db.ACT_MIGRATE_FAIL, reason, task.UserCred)
	logclient.AddActionLogWithContext(ctx, disk, logclient.ACT_MIGRATE, reason, task.UserCred, false)
	notifyclient.NotifySystemErrorWithCtx(ctx, disk.Id, disk.Name, compute.DISK_MIGRATE_FAIL, reason.String())
	notifyclient.EventNotify(ctx, task.GetUserCred(), notifyclient.SEventNotifyParam{
		Obj:    disk,
		Action: notifyclient.ActionMigrate,
		IsFail: true,
	})
}

func (task *DiskMigrateTask) TaskFailed(ctx context.Context, disk *models.SDisk, reason jsonutils.JSONObject) {
	task.markFailed(ctx, disk, reason)
	task.SetStageFailed(ctx, reason)
}

func (task *DiskMigrateTask) GetSchedParams() (*schedapi.ScheduleInput, error) {
	obj := task.GetObject()
	disk := obj.(*models.SDisk)

	targetStorageId, _ := task.Params.GetString("target_storage_id")
	return disk.GetSchedMigrateParams(targetStorageId)
}

func (task *DiskMigrateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	StartScheduleObjects(ctx, task, []db.IStandaloneModel{obj})
}

func (task *DiskMigrateTask) OnScheduleFailCallback(ctx context.Context, obj IScheduleModel, reason jsonutils.JSONObject, index int) {
	// do nothing
}

func (task *DiskMigrateTask) OnScheduleFailed(ctx context.Context, reason jsonutils.JSONObject) {
	obj := task.GetObject()
	disk := obj.(*models.SDisk)
	task.TaskFailed(ctx, disk, reason)
}

func (task *DiskMigrateTask) SaveScheduleResult(ctx context.Context, obj IScheduleModel, candidate *schedapi.CandidateResource, index int) {
	disk := obj.(*models.SDisk)
	targetHostId := candidate.HostId
	storageIds := candidate.Disks[0].StorageIds
	targetHost := models.HostManager.FetchHostById(targetHostId)
	if targetHost == nil {
		task.TaskFailed(ctx, disk, jsonutils.NewString("target host not found?"))
		return
	}
	if len(storageIds) == 0 {
		task.TaskFailed(ctx, disk, jsonutils.NewString("no target storage found?"))
		return
	}
	var storageId string
	for i := range storageIds {
		if storageIds[i] != disk.StorageId {
			storageId = storageIds[i]
			break
		}
	}

	storage := models.StorageManager.FetchStorageById(storageId)
	if storage == nil {
		task.TaskFailed(ctx, disk, jsonutils.NewString("target storage not found?"))
		return
	}

	task.Params.Set("target_host_id", jsonutils.NewString(targetHostId))
	task.Params.Set("target_storage_id", jsonutils.NewString(storage.Id))
	task.SetStage("OnStorageCacheImage", nil)

	if disk.TemplateId != "" {
		format, err := disk.GetCacheImageFormat(ctx)
		if err != nil {
			task.TaskFailed(ctx, disk, jsonutils.NewString(fmt.Sprintf("disk get cache image format failed %s", err)))
			return
		}

		input := compute.CacheImageInput{
			ImageId:      disk.GetTemplateId(),
			Format:       format,
			ParentTaskId: task.GetTaskId(),
		}
		disk.SetStatus(ctx, task.UserCred, compute.DISK_IMAGE_CACHING, "On disk migrate save schedule result")
		storagecache := storage.GetStoragecache()
		storagecache.StartImageCacheTask(ctx, task.UserCred, input)
	} else {
		task.OnStorageCacheImage(ctx, disk, nil)
	}
}

func (task *DiskMigrateTask) OnStorageCacheImage(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	disk.SetStatus(ctx, task.UserCred, compute.DISK_MIGRATING, "On disk migrate start migrate")

	storage, err := disk.GetStorage()
	if err != nil {
		task.TaskFailed(ctx, disk, jsonutils.NewString(err.Error()))
		return
	}

	sourceHost, err := storage.GetMasterHost()
	if err != nil {
		task.TaskFailed(ctx, disk, jsonutils.NewString(err.Error()))
		return
	}
	driver, err := sourceHost.GetHostDriver()
	if err != nil {
		task.TaskFailed(ctx, disk, jsonutils.NewString(err.Error()))
		return
	}
	ret, err := driver.RequestDiskSrcMigratePrepare(ctx, sourceHost, disk, task)
	if err != nil {
		task.TaskFailed(ctx, disk, jsonutils.NewString(err.Error()))
		return
	}
	snapshotsUri := fmt.Sprintf("%s/download/snapshots/", sourceHost.ManagerUri)
	diskUri := fmt.Sprintf("%s/download/disks/", sourceHost.ManagerUri)

	body := jsonutils.NewDict()
	if ret != nil {
		body.Update(ret.(*jsonutils.JSONDict))
	}

	snapChain := []string{}
	if ret.Contains("disk_snaps_chain") {
		err = ret.Unmarshal(&snapChain, "disk_snaps_chain")
		if err != nil {
			task.TaskFailed(ctx, disk, jsonutils.NewString(errors.Wrap(err, "unmarshal snap chain").Error()))
			return
		}
	}

	outChainSnapshotIds := jsonutils.NewArray()
	snapshots := models.SnapshotManager.GetDiskSnapshots(disk.Id)
	for j := 0; j < len(snapshots); j++ {
		if !utils.IsInStringArray(snapshots[j].Id, snapChain) {
			outChainSnapshotIds.Add(jsonutils.NewString(snapshots[j].Id))
		}
	}

	body.Set("out_chain_snapshots", outChainSnapshotIds)
	body.Set("snapshots_uri", jsonutils.NewString(snapshotsUri))
	body.Set("disk_uri", jsonutils.NewString(diskUri))
	body.Set("src_storage_id", jsonutils.NewString(disk.StorageId))
	if disk.TemplateId != "" {
		body.Set("template_id", jsonutils.NewString(disk.TemplateId))
	}

	targetHostId, _ := task.Params.GetString("target_host_id")
	targetHost := models.HostManager.FetchHostById(targetHostId)
	targetStorageId, _ := task.Params.GetString("target_storage_id")
	targetStorage := models.StorageManager.FetchStorageById(targetStorageId)

	task.SetStage("OnDiskMigrate", nil)
	targetDriver, err := targetHost.GetHostDriver()
	if err != nil {
		task.TaskFailed(ctx, disk, jsonutils.NewString(errors.Wrap(err, "GetHostDriver").Error()))
		return
	}
	if err = targetDriver.RequestDiskMigrate(ctx, targetHost, targetStorage, disk, task, body); err != nil {
		task.TaskFailed(ctx, disk, jsonutils.NewString(fmt.Sprintf("failed request disk migrate %s", err)))
		return
	}
}

func (task *DiskMigrateTask) OnDiskMigrate(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	srcStorage, _ := disk.GetStorage()
	srcHost, _ := srcStorage.GetMasterHost()

	diskPath, _ := data.GetString("disk_path")
	targetStorageId, _ := task.Params.GetString("target_storage_id")
	_, err := db.Update(disk, func() error {
		//disk.Status = compute.DISK_READY
		disk.StorageId = targetStorageId
		if diskPath != "" {
			disk.AccessPath = diskPath
		}
		return nil
	})
	if err != nil {
		task.TaskFailed(ctx, disk, jsonutils.NewString(fmt.Sprintf("db failed update disk %s", err)))
		return
	}
	snapshots := models.SnapshotManager.GetDiskSnapshots(disk.Id)
	for _, snapshot := range snapshots {
		_, err := db.Update(&snapshot, func() error {
			snapshot.StorageId = targetStorageId
			return nil
		})
		if err != nil {
			task.TaskFailed(ctx, disk, jsonutils.NewString(fmt.Sprintf("db failed update disk snapshot %s %s", snapshot.Id, err)))
			return
		}
	}

	task.SetStage("OnDeallocateSourceDisk", nil)
	driver, err := srcHost.GetHostDriver()
	if err != nil {
		task.TaskFailed(ctx, disk, jsonutils.NewString(fmt.Sprintf("GetHostDriver: %v", err)))
		return
	}
	err = driver.RequestDeallocateDiskOnHost(ctx, srcHost, srcStorage, disk, true, task)
	if err != nil {
		task.TaskFailed(ctx, disk, jsonutils.NewString(fmt.Sprintf("failed deallocate disk on src storage %s", err)))
		return
	}
}

func (task *DiskMigrateTask) OnDiskMigrateFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	task.TaskFailed(ctx, disk, data)
}

func (task *DiskMigrateTask) OnDeallocateSourceDisk(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	db.Update(disk, func() error {
		disk.Status = compute.DISK_READY
		return nil
	})
	task.SetStageComplete(ctx, nil)
	db.OpsLog.LogEvent(disk, db.ACT_MIGRATE, "OnDeallocateSourceDisk", task.UserCred)
	logclient.AddActionLogWithContext(ctx, disk, logclient.ACT_MIGRATE, task.Params, task.UserCred, true)
}

func (task *DiskMigrateTask) OnDeallocateSourceDiskFailed(ctx context.Context, disk *models.SDisk, data jsonutils.JSONObject) {
	task.TaskFailed(ctx, disk, data)
}
