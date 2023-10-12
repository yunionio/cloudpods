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
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestChangeConfigTask struct {
	SSchedTask
}

func init() {
	taskman.RegisterTask(GuestChangeConfigTask{})
}

func (task *GuestChangeConfigTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	StartScheduleObjects(ctx, task, nil)
}

func (task *GuestChangeConfigTask) GetSchedParams() (*schedapi.ScheduleInput, error) {
	schedInput := new(schedapi.ScheduleInput)
	err := task.Params.Unmarshal(schedInput, "sched_desc")
	if err != nil {
		return nil, err
	}
	return schedInput, nil
}

func (task *GuestChangeConfigTask) OnStartSchedule(obj IScheduleModel) {
	// do nothing
}

func (task *GuestChangeConfigTask) OnScheduleFailCallback(ctx context.Context, obj IScheduleModel, reason jsonutils.JSONObject, index int) {
	// do nothing
}

func (task *GuestChangeConfigTask) OnScheduleFailed(ctx context.Context, reason jsonutils.JSONObject) {
	obj := task.GetObject()
	guest := obj.(*models.SGuest)
	task.markStageFailed(ctx, guest, reason)
}

func (task *GuestChangeConfigTask) SaveScheduleResult(ctx context.Context, obj IScheduleModel, target *schedapi.CandidateResource, index int) {
	// must get object from task, because of obj is nil
	guest := task.GetObject().(*models.SGuest)
	task.Params.Set("sched_session_id", jsonutils.NewString(target.SessionId))
	if task.Params.Contains("create") {
		disks := make([]*api.DiskConfig, 0)
		err := task.Params.Unmarshal(&disks, "create")
		if err != nil {
			task.markStageFailed(ctx, guest, jsonutils.NewString(err.Error()))
			return
		}
		var resizeDisksCount = 0
		if task.Params.Contains("resize") {
			iResizeDisks, err := task.Params.Get("resize")
			if err != nil {
				task.markStageFailed(ctx, guest, jsonutils.NewString(err.Error()))
				return
			}
			resizeDisksCount = iResizeDisks.(*jsonutils.JSONArray).Length()
		}
		for i := 0; i < len(disks); i++ {
			disks[i].Storage = target.Disks[resizeDisksCount+i].StorageIds[0]
		}
		task.Params.Set("create", jsonutils.Marshal(disks))
	}

	task.SetStage("StartResizeDisks", nil)

	task.StartResizeDisks(ctx, guest, nil)
}

func (task *GuestChangeConfigTask) StartResizeDisks(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	_, err := task.Params.Get("resize")
	if err == nil {
		task.SetStage("OnDisksResizeComplete", nil)
		task.OnDisksResizeComplete(ctx, guest, data)
	} else {
		task.DoCreateDisksTask(ctx, guest)
	}
}

func (task *GuestChangeConfigTask) OnDisksResizeComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	iResizeDisks, err := task.Params.Get("resize")
	if iResizeDisks == nil || err != nil {
		task.markStageFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	resizeDisks := iResizeDisks.(*jsonutils.JSONArray)
	for i := 0; i < resizeDisks.Length(); i++ {
		iResizeSet, err := resizeDisks.GetAt(i)
		if err != nil {
			task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("resizeDisks.GetAt fail %s", err)))
			return
		}
		resizeSet := iResizeSet.(*jsonutils.JSONArray)
		diskId, err := resizeSet.GetAt(0)
		if err != nil {
			task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("resizeSet.GetAt(0) fail %s", err)))
			return
		}
		idStr, err := diskId.GetString()
		if err != nil {
			task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("diskId.GetString fail %s", err)))
			return
		}
		jSize, err := resizeSet.GetAt(1)
		if err != nil {
			task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("resizeSet.GetAt(1) fail %s", err)))
			return
		}
		size, err := jSize.Int()
		if err != nil {
			task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("jSize.Int fail %s", err)))
			return
		}
		iDisk, err := models.DiskManager.FetchById(idStr)
		if err != nil {
			task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("models.DiskManager.FetchById(idStr) fail %s", err)))
			return
		}
		disk := iDisk.(*models.SDisk)
		if disk.DiskSize < int(size) {
			var pendingUsage models.SQuota
			err = task.GetPendingUsage(&pendingUsage, 0)
			if err != nil {
				task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("task.GetPendingUsage(&pendingUsage) fail %s", err)))
				return
			}
			err = guest.StartGuestDiskResizeTask(ctx, task.UserCred, disk.Id, size, task.GetTaskId(), &pendingUsage)
			if err != nil {
				task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("guest.StartGuestDiskResizeTask fail %s", err)))
				return
			}
			return
		}
	}

	task.DoCreateDisksTask(ctx, guest)
}

func (task *GuestChangeConfigTask) DoCreateDisksTask(ctx context.Context, guest *models.SGuest) {
	disks := make([]*api.DiskConfig, 0)
	err := task.Params.Unmarshal(&disks, "create")
	if err != nil || len(disks) == 0 {
		task.OnCreateDisksComplete(ctx, guest, nil)
		return
	}
	host, _ := guest.GetHost()
	err = guest.CreateDisksOnHost(ctx, task.UserCred, host, disks, nil, false, false, nil, nil, false)
	if err != nil {
		task.markStageFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	task.SetStage("OnCreateDisksComplete", nil)
	guest.StartGuestCreateDiskTask(ctx, task.UserCred, disks, task.GetTaskId())
}

func (task *GuestChangeConfigTask) OnCreateDisksCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	task.markStageFailed(ctx, guest, err)
}

func (task *GuestChangeConfigTask) OnCreateDisksComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	if task.Params.Contains("instance_type") || task.Params.Contains("vcpu_count") || task.Params.Contains("vmem_size") {
		task.SetStage("OnGuestChangeCpuMemSpecComplete", nil)
		instanceType, _ := task.Params.GetString("instance_type")
		vcpuCount, _ := task.Params.Int("vcpu_count")
		vmemSize, _ := task.Params.Int("vmem_size")
		if len(instanceType) > 0 {
			provider := guest.GetDriver().GetProvider()
			sku, err := models.ServerSkuManager.FetchSkuByNameAndProvider(instanceType, provider, false)
			if err != nil {
				task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("models.ServerSkuManager.FetchSkuByNameAndProvider error %s", err)))
				return
			}
			vcpuCount = int64(sku.CpuCoreCount)
			vmemSize = int64(sku.MemorySizeMB)
		} else {
			if vcpuCount == 0 {
				vcpuCount = int64(guest.VcpuCount)
			}
			if vmemSize == 0 {
				vmemSize = int64(guest.VmemSize)
			}
		}
		task.startGuestChangeCpuMemSpec(ctx, guest, instanceType, vcpuCount, vmemSize)
	} else {
		task.OnGuestChangeCpuMemSpecComplete(ctx, obj, data)
	}
}

func (task *GuestChangeConfigTask) startGuestChangeCpuMemSpec(ctx context.Context, guest *models.SGuest, instanceType string, vcpuCount int64, vmemSize int64) {
	err := guest.GetDriver().RequestChangeVmConfig(ctx, guest, task, instanceType, vcpuCount, vmemSize)
	if err != nil {
		task.markStageFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
}

func (task *GuestChangeConfigTask) OnGuestChangeCpuMemSpecComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	instanceType, _ := task.Params.GetString("instance_type")
	vcpuCount, _ := task.Params.Int("vcpu_count")
	vmemSize, _ := task.Params.Int("vmem_size")

	if len(instanceType) == 0 {
		skus, err := models.ServerSkuManager.GetSkus(api.CLOUD_PROVIDER_ONECLOUD, int(vcpuCount), int(vmemSize))
		if err == nil && len(skus) > 0 {
			instanceType = skus[0].GetName()
		}
	}

	addCpu := int(vcpuCount - int64(guest.VcpuCount))
	addMem := int(vmemSize - int64(guest.VmemSize))

	_, err := db.Update(guest, func() error {
		if vcpuCount > 0 {
			guest.VcpuCount = int(vcpuCount)
		}
		if vmemSize > 0 {
			guest.VmemSize = int(vmemSize)
		}
		if len(instanceType) > 0 {
			guest.InstanceType = instanceType
		}
		return nil
	})
	if err != nil {
		task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("Update fail %s", err)))
		return
	}
	changeConfigSpec := guest.GetShortDesc(ctx)
	if vcpuCount > 0 && addCpu != 0 {
		changeConfigSpec.Set("add_cpu", jsonutils.NewInt(int64(addCpu)))
	}
	if vmemSize > 0 && addMem != 0 {
		changeConfigSpec.Set("add_mem", jsonutils.NewInt(int64(addMem)))
	}
	if len(instanceType) > 0 {
		changeConfigSpec.Set("instance_type", jsonutils.NewString(instanceType))
	}

	db.OpsLog.LogEvent(guest, db.ACT_CHANGE_FLAVOR, changeConfigSpec.String(), task.UserCred)

	var pendingUsage models.SQuota
	err = task.GetPendingUsage(&pendingUsage, 0)
	if err != nil {
		task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("GetPendingUsage %s", err)))
		return
	}
	var cancelUsage models.SQuota
	var reduceUsage models.SQuota
	if addCpu > 0 {
		cancelUsage.Cpu = addCpu
	} else if addCpu < 0 {
		reduceUsage.Cpu = -addCpu
	}
	if addMem > 0 {
		cancelUsage.Memory = addMem
	} else if addMem < 0 {
		reduceUsage.Memory = -addMem
	}

	keys, err := guest.GetQuotaKeys()
	if err != nil {
		task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("guest.GetQuotaKeys %s", err)))
		return
	}
	cancelUsage.SetKeys(keys)
	reduceUsage.SetKeys(keys)

	lockman.LockClass(ctx, guest.GetModelManager(), guest.ProjectId)
	defer lockman.ReleaseClass(ctx, guest.GetModelManager(), guest.ProjectId)

	if !cancelUsage.IsEmpty() {
		err = quotas.CancelPendingUsage(ctx, task.UserCred, &pendingUsage, &cancelUsage, true) // success
		if err != nil {
			task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("CancelPendingUsage fail %s", err)))
			return
		}
		err = task.SetPendingUsage(&pendingUsage, 0)
		if err != nil {
			task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("SetPendingUsage fail %s", err)))
			return
		}
	}

	if !reduceUsage.IsEmpty() {
		quotas.CancelUsages(ctx, task.UserCred, []db.IUsage{&reduceUsage})
	}

	task.OnGuestChangeCpuMemSpecFinish(ctx, guest)
}

func (task *GuestChangeConfigTask) OnGuestChangeCpuMemSpecCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if err := guest.GetDriver().OnGuestChangeCpuMemFailed(ctx, guest, data.(*jsonutils.JSONDict), task); err != nil {
		log.Errorln(err)
	}
	task.markStageFailed(ctx, guest, data)
}

func (task *GuestChangeConfigTask) OnGuestChangeCpuMemSpecFinish(ctx context.Context, guest *models.SGuest) {
	models.HostManager.ClearSchedDescCache(guest.HostId)
	task.SetStage("OnSyncConfigComplete", nil)
	err := guest.StartSyncTaskWithoutSyncstatus(ctx, task.UserCred, false, task.GetTaskId())
	if err != nil {
		task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("StartSyncstatus fail %s", err)))
		return
	}
}

func (task *GuestChangeConfigTask) OnSyncConfigComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	task.SetStage("OnSyncStatusComplete", nil)
	err := guest.StartSyncstatus(ctx, task.UserCred, task.GetTaskId())
	if err != nil {
		task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("StartSyncstatus fail %s", err)))
		return
	}
}

func (task *GuestChangeConfigTask) OnSyncStatusComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if guest.Status == api.VM_READY && jsonutils.QueryBoolean(task.Params, "auto_start", false) {
		task.SetStage("OnGuestStartComplete", nil)
		guest.StartGueststartTask(ctx, task.UserCred, nil, task.GetTaskId())
	} else {
		dt := jsonutils.NewDict()
		dt.Add(jsonutils.NewString(guest.Id), "id")
		task.SetStageComplete(ctx, dt)
	}
	logclient.AddActionLogWithStartable(task, guest, logclient.ACT_VM_CHANGE_FLAVOR, "", task.UserCred, true)
	guest.EventNotify(ctx, task.UserCred, notifyclient.ActionChangeConfig)
}

func (task *GuestChangeConfigTask) OnGuestStartComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	dt := jsonutils.NewDict()
	dt.Add(jsonutils.NewString(guest.Id), "id")
	task.SetStageComplete(ctx, dt)
}

func (task *GuestChangeConfigTask) OnGuestStartCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	task.SetStageFailed(ctx, data)
}

func (task *GuestChangeConfigTask) markStageFailed(ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject) {
	guest.SetStatus(task.UserCred, api.VM_CHANGE_FLAVOR_FAIL, reason.String())
	db.OpsLog.LogEvent(guest, db.ACT_CHANGE_FLAVOR_FAIL, reason, task.UserCred)
	logclient.AddActionLogWithStartable(task, guest, logclient.ACT_VM_CHANGE_FLAVOR, reason, task.UserCred, false)
	notifyclient.EventNotify(ctx, task.GetUserCred(), notifyclient.SEventNotifyParam{
		Obj:    guest,
		Action: notifyclient.ActionChangeConfig,
		IsFail: true,
	})
	task.SetStageFailed(ctx, reason)
}

func (task *GuestChangeConfigTask) SetStageFailed(ctx context.Context, reason jsonutils.JSONObject) {
	guest := task.GetObject().(*models.SGuest)
	hostId := guest.HostId
	sessionId, _ := task.Params.GetString("sched_session_id")
	lockman.LockRawObject(ctx, models.HostManager.KeywordPlural(), hostId)
	defer lockman.ReleaseRawObject(ctx, models.HostManager.KeywordPlural(), hostId)
	models.HostManager.ClearSchedDescSessionCache(hostId, sessionId)

	task.SSchedTask.SetStageFailed(ctx, reason)
}

func (task *GuestChangeConfigTask) SetStageComplete(ctx context.Context, data *jsonutils.JSONDict) {
	guest := task.GetObject().(*models.SGuest)
	hostId := guest.HostId
	sessionId, _ := task.Params.GetString("sched_session_id")
	lockman.LockRawObject(ctx, models.HostManager.KeywordPlural(), hostId)
	defer lockman.ReleaseRawObject(ctx, models.HostManager.KeywordPlural(), hostId)
	models.HostManager.ClearSchedDescSessionCache(hostId, sessionId)

	task.SSchedTask.SetStageComplete(ctx, data)
}
