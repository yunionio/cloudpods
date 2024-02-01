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
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DiskBatchCreateTask struct {
	SSchedTask
}

func init() {
	taskman.RegisterTask(DiskBatchCreateTask{})
}

func (task *DiskBatchCreateTask) getNeedScheduleDisks(objs []db.IStandaloneModel) []db.IStandaloneModel {
	toSchedDisks := make([]db.IStandaloneModel, 0)
	for _, obj := range objs {
		disk := obj.(*models.SDisk)
		if disk.StorageId == "" {
			toSchedDisks = append(toSchedDisks, disk)
		}
	}
	return toSchedDisks
}

func (task *DiskBatchCreateTask) clearPendingUsage(ctx context.Context, disk *models.SDisk) {
	ClearTaskPendingUsage(ctx, task)
	ClearTaskPendingRegionUsage(ctx, task)
}

func (task *DiskBatchCreateTask) OnInit(ctx context.Context, objs []db.IStandaloneModel, body jsonutils.JSONObject) {
	toSchedDisks := task.getNeedScheduleDisks(objs)
	if len(toSchedDisks) == 0 {
		task.SetStage("OnScheduleComplete", nil)
		// create not need schedule disks directly
		for _, disk := range objs {
			task.startCreateDisk(ctx, disk.(*models.SDisk))
		}
		return
	}
	StartScheduleObjects(ctx, task, toSchedDisks)
}

func (task *DiskBatchCreateTask) GetCreateInput(data jsonutils.JSONObject) (*api.DiskCreateInput, error) {
	input := new(api.DiskCreateInput)
	err := data.Unmarshal(input)
	return input, err
}

func (task *DiskBatchCreateTask) GetSchedParams() (*schedapi.ScheduleInput, error) {
	data := getBatchParamsAtIndex(task, 0)
	return task.getSchedParamsInternal(data)
}

func (task *DiskBatchCreateTask) getSchedParamsInternal(data jsonutils.JSONObject) (*schedapi.ScheduleInput, error) {
	input, err := task.GetCreateInput(data)
	if err != nil {
		return nil, err
	}
	ret := new(schedapi.ScheduleInput)
	srvInput := input.ToServerCreateInput()
	err = srvInput.JSON(srvInput).Unmarshal(ret)
	return ret, err
}

func (task *DiskBatchCreateTask) GetDisks(data jsonutils.JSONObject) ([]*api.DiskConfig, error) {
	input, err := task.getSchedParamsInternal(data)
	if err != nil {
		return nil, err
	}
	return input.Disks, nil
}

func (task *DiskBatchCreateTask) GetFirstDisk(data jsonutils.JSONObject) (*api.DiskConfig, error) {
	disks, err := task.GetDisks(data)
	if err != nil {
		return nil, err
	}
	if len(disks) == 0 {
		return nil, fmt.Errorf("Empty disks to schedule")
	}
	return disks[0], nil
}

func (task *DiskBatchCreateTask) OnScheduleFailCallback(ctx context.Context, obj IScheduleModel, reason jsonutils.JSONObject, index int) {
	task.SSchedTask.OnScheduleFailCallback(ctx, obj, reason, index)
	disk := obj.(*models.SDisk)
	log.Errorf("Schedule disk %s failed", disk.Name)
	task.clearPendingUsage(ctx, disk)
}

func (task *DiskBatchCreateTask) SaveScheduleResult(ctx context.Context, obj IScheduleModel, candidate *schedapi.CandidateResource, index int) {
	var err error
	disk := obj.(*models.SDisk)
	// pendingUsage := models.SQuota{}
	// err = task.GetPendingUsage(&pendingUsage, 0)
	// if err != nil {
	// 	log.Errorf("GetPendingUsage fail %s", err)
	// }

	// input, _ := task.GetCreateInput()
	// quotaPlatform := models.GetQuotaPlatformID(input.Hypervisor)

	// quotaStorage := models.SQuota{Storage: disk.DiskSize}

	onError := func(err error) {
		task.clearPendingUsage(ctx, disk)
		disk.SetStatus(ctx, task.UserCred, api.DISK_ALLOC_FAILED, "")
		logclient.AddActionLogWithStartable(task, disk, logclient.ACT_ALLOCATE, err, task.UserCred, false)
		task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		db.OpsLog.LogEvent(disk, db.ACT_ALLOCATE_FAIL, err, task.UserCred)
		notifyclient.NotifySystemErrorWithCtx(ctx, disk.Id, disk.Name, api.DISK_ALLOC_FAILED, err.Error())
	}

	data := getBatchParamsAtIndex(task, index)

	diskConfig, err := task.GetFirstDisk(data)
	if err != nil {
		onError(err)
		return
	}

	storageIds := []string{}
	var hostId string
	if candidate != nil && len(candidate.Disks) != 0 {
		hostId = candidate.HostId
		storageIds = candidate.Disks[0].StorageIds
	}
	err = disk.SetStorageByHost(hostId, diskConfig, storageIds)
	if err != nil {
		onError(err)
		return
	}

	task.startCreateDisk(ctx, disk)
}

func (task *DiskBatchCreateTask) startCreateDisk(ctx context.Context, disk *models.SDisk) {
	pendingUsage := models.SQuota{}
	err := task.GetPendingUsage(&pendingUsage, 0)
	if err != nil {
		log.Warningf("GetPendingUsage fail %s", err)
	}

	quotaStorage := models.SQuota{Storage: disk.DiskSize}
	keys, err := disk.GetQuotaKeys()
	if err != nil {
		log.Warningf("disk.GetQuotaKeys fail %s", err)
	}
	quotaStorage.SetKeys(keys)
	quotas.CancelPendingUsage(ctx, task.UserCred, &pendingUsage, &quotaStorage, true) // success
	task.SetPendingUsage(&pendingUsage, 0)

	disk.StartDiskCreateTask(ctx, task.GetUserCred(), false, disk.SnapshotId, task.GetTaskId())
}

func (task *DiskBatchCreateTask) OnScheduleComplete(ctx context.Context, items []db.IStandaloneModel, data *jsonutils.JSONDict) {
	task.SetStageComplete(ctx, nil)
}
