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
	"yunion.io/x/pkg/errors"

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

func (task *GuestChangeConfigTask) getChangeConfigSetting() (*api.ServerChangeConfigSettings, error) {
	confs := &api.ServerChangeConfigSettings{}
	err := task.Params.Unmarshal(confs)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal ServerChangeConfigSettings")
	}
	return confs, nil
}

func (task *GuestChangeConfigTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	StartScheduleObjects(ctx, task, nil)
}

func (task *GuestChangeConfigTask) GetSchedParams() (*schedapi.ScheduleInput, error) {
	confs, err := task.getChangeConfigSetting()
	if err != nil {
		return nil, errors.Wrap(err, "getChangeConfigSetting")
	}
	schedInput := new(schedapi.ScheduleInput)
	err = confs.SchedDesc.Unmarshal(schedInput)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal sched_desc")
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

	confs, err := task.getChangeConfigSetting()
	if err != nil {
		task.markStageFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	if confs.ExtraCpuChanged() {
		_, err = db.Update(guest, func() error {
			if confs.ExtraCpuCount > 0 {
				guest.ExtraCpuCount = confs.ExtraCpuCount
			}
			return nil
		})
		if err != nil {
			task.markStageFailed(ctx, guest, jsonutils.NewString(err.Error()))
			return
		}
	}

	if len(target.CpuNumaPin) > 0 {
		task.Params.Set("cpu_numa_pin", jsonutils.Marshal(target.CpuNumaPin))
	}

	/*confs, err := task.getChangeConfigSetting()
	if err != nil {
		task.markStageFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	if len(confs.Create) > 0 {
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
	}*/

	task.SetStage("StartResizeDisks", nil)

	task.StartResizeDisks(ctx, guest, nil)
}

func (task *GuestChangeConfigTask) StartResizeDisks(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	confs, err := task.getChangeConfigSetting()
	if err != nil {
		task.markStageFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	if len(confs.Resize) > 0 {
		task.SetStage("OnDisksResizeComplete", nil)
		task.OnDisksResizeComplete(ctx, guest, data)
	} else {
		task.DoCreateDisksTask(ctx, guest)
	}
}

func (task *GuestChangeConfigTask) OnDisksResizeComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	confs, err := task.getChangeConfigSetting()
	if err != nil {
		task.markStageFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}

	for i := 0; i < len(confs.Resize); i++ {
		diskId := confs.Resize[i].DiskId
		diskSizeMb := confs.Resize[i].SizeMb

		iDisk, err := models.DiskManager.FetchById(diskId)
		if err != nil {
			task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("models.DiskManager.FetchById(%s) fail %s", diskId, err)))
			return
		}

		disk := iDisk.(*models.SDisk)
		if disk.DiskSize < diskSizeMb {
			var pendingUsage models.SQuota
			err = task.GetPendingUsage(&pendingUsage, 0)
			if err != nil {
				task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("task.GetPendingUsage(&pendingUsage) fail %s", err)))
				return
			}
			err = guest.StartGuestDiskResizeTask(ctx, task.UserCred, disk.Id, int64(diskSizeMb), task.GetTaskId(), &pendingUsage)
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
	confs, err := task.getChangeConfigSetting()
	if err != nil {
		task.markStageFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}

	disks := confs.Create
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

	confs, err := task.getChangeConfigSetting()
	if err != nil {
		task.markStageFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}

	if confs.CpuChanged() || confs.MemChanged() {
		task.SetStage("OnGuestChangeCpuMemSpecComplete", nil)
		task.startGuestChangeCpuMemSpec(ctx, guest, confs.InstanceType, confs.VcpuCount, confs.CpuSockets, confs.VmemSize)
	} else {
		task.OnGuestChangeCpuMemSpecComplete(ctx, obj, data)
	}
}

func (task *GuestChangeConfigTask) startGuestChangeCpuMemSpec(ctx context.Context, guest *models.SGuest, instanceType string, vcpuCount, cpuSockets int, vmemSize int) {
	drv, err := guest.GetDriver()
	if err != nil {
		task.markStageFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	err = drv.RequestChangeVmConfig(ctx, guest, task, instanceType, int64(vcpuCount), int64(cpuSockets), int64(vmemSize))
	if err != nil {
		task.markStageFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
}

func (task *GuestChangeConfigTask) OnGuestChangeCpuMemSpecComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	confs, err := task.getChangeConfigSetting()
	if err != nil {
		task.markStageFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}

	if len(confs.InstanceType) == 0 {
		skus, err := models.ServerSkuManager.GetSkus(api.CLOUD_PROVIDER_ONECLOUD, confs.VcpuCount, confs.VmemSize)
		if err == nil && len(skus) > 0 {
			confs.InstanceType = skus[0].GetName()
		}
	}

	_, err = db.Update(guest, func() error {
		if confs.VcpuCount > 0 {
			guest.VcpuCount = confs.VcpuCount
		}
		if confs.CpuSockets > 0 {
			guest.CpuSockets = confs.CpuSockets
		}
		if confs.VmemSize > 0 {
			guest.VmemSize = confs.VmemSize
		}
		if len(confs.InstanceType) > 0 {
			guest.InstanceType = confs.InstanceType
		}
		return nil
	})
	if err != nil {
		task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("Update fail %s", err)))
		return
	}

	if task.Params.Contains("cpu_numa_pin") {
		cpuNumaPinSched := make([]schedapi.SCpuNumaPin, 0)
		task.Params.Unmarshal(&cpuNumaPinSched, "cpu_numa_pin")
		cpuNumaPinTarget := make([]api.SCpuNumaPin, 0)
		if data.Contains("cpu_numa_pin") {
			data.Unmarshal(&cpuNumaPinTarget, "cpu_numa_pin")
		}

		err = guest.UpdateCpuNumaPin(ctx, task.UserCred, cpuNumaPinSched, cpuNumaPinTarget)
		if err != nil {
			task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("Update cpu numa pin fail %s", err)))
			return
		}
	}

	changeConfigSpec := guest.GetShortDesc(ctx)
	if confs.VcpuCount > 0 && confs.AddedCpu() > 0 {
		changeConfigSpec.Set("add_cpu", jsonutils.NewInt(int64(confs.AddedCpu())))
	}
	if confs.VmemSize > 0 && confs.AddedMem() > 0 {
		changeConfigSpec.Set("add_mem", jsonutils.NewInt(int64(confs.AddedMem())))
	}
	if len(confs.InstanceType) > 0 {
		changeConfigSpec.Set("instance_type", jsonutils.NewString(confs.InstanceType))
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
	if addCpu := confs.AddedCpu(); addCpu > 0 {
		cancelUsage.Cpu = addCpu
	}
	if addMem := confs.AddedMem(); addMem > 0 {
		cancelUsage.Memory = addMem
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
	drv, err := guest.GetDriver()
	if err != nil {
		task.markStageFailed(ctx, guest, data)
		return
	}
	if err := drv.OnGuestChangeCpuMemFailed(ctx, guest, data.(*jsonutils.JSONDict), task); err != nil {
		log.Errorln(err)
	}
	task.markStageFailed(ctx, guest, data)
}

func (task *GuestChangeConfigTask) OnGuestChangeCpuMemSpecFinish(ctx context.Context, guest *models.SGuest) {
	models.HostManager.ClearSchedDescCache(guest.HostId)

	confs, err := task.getChangeConfigSetting()
	if err != nil {
		task.markStageFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}

	if len(confs.ResetTrafficLimits) > 0 {
		host, _ := guest.GetHost()
		// resetTraffics := []api.ServerNicTrafficLimit{}
		// task.Params.Unmarshal(&resetTraffics, "reset_traffic_limits")
		task.SetStage("OnGuestResetNicTraffics", nil)
		drv, err := guest.GetDriver()
		if err != nil {
			task.markStageFailed(ctx, guest, jsonutils.NewString(err.Error()))
			return
		}
		err = drv.RequestResetNicTrafficLimit(ctx, task, host, guest, confs.ResetTrafficLimits)
		if err != nil {
			task.markStageFailed(ctx, guest, jsonutils.NewString(err.Error()))
			return
		}
	} else {
		task.OnGuestResetNicTraffics(ctx, guest, nil)
	}
}

func (task *GuestChangeConfigTask) OnGuestResetNicTraffics(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	confs, err := task.getChangeConfigSetting()
	if err != nil {
		task.markStageFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}

	if len(confs.ResetTrafficLimits) > 0 {
		resetTraffics := confs.ResetTrafficLimits
		for i := range resetTraffics {
			input := resetTraffics[i]
			gn, _ := guest.GetGuestnetworkByMac(input.Mac)
			err := gn.UpdateNicTrafficLimit(input.RxTrafficLimit, input.TxTrafficLimit)
			if err != nil {
				task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("failed update guest nic traffic limit %s", err)))
				return
			}
			err = gn.UpdateNicTrafficUsed(0, 0)
			if err != nil {
				task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("failed update guest nic traffic used %s", err)))
				return
			}
		}
	}

	if len(confs.SetTrafficLimits) > 0 {
		host, _ := guest.GetHost()
		setTraffics := confs.SetTrafficLimits
		task.SetStage("OnGuestSetNicTraffics", nil)
		drv, err := guest.GetDriver()
		if err != nil {
			task.markStageFailed(ctx, guest, jsonutils.NewString(err.Error()))
			return
		}
		err = drv.RequestSetNicTrafficLimit(ctx, task, host, guest, setTraffics)
		if err != nil {
			task.markStageFailed(ctx, guest, jsonutils.NewString(err.Error()))
			return
		}
	} else {
		task.OnGuestSetNicTraffics(ctx, guest, nil)
	}
}

func (task *GuestChangeConfigTask) OnGuestSetNicTraffics(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	confs, err := task.getChangeConfigSetting()
	if err != nil {
		task.markStageFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}

	if len(confs.SetTrafficLimits) > 0 {
		setTraffics := confs.SetTrafficLimits
		for i := range setTraffics {
			input := setTraffics[i]
			gn, _ := guest.GetGuestnetworkByMac(input.Mac)
			err := gn.UpdateNicTrafficLimit(input.RxTrafficLimit, input.TxTrafficLimit)
			if err != nil {
				task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("failed update guest nic traffic limit %s", err)))
				return
			}
		}
	}

	task.SetStage("OnSyncConfigComplete", nil)
	err = guest.StartSyncTaskWithoutSyncstatus(ctx, task.UserCred, false, task.GetTaskId())
	if err != nil {
		task.markStageFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("StartSyncstatus fail %s", err)))
		return
	}
}

func (task *GuestChangeConfigTask) OnGuestResetNicTrafficsFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.markStageFailed(ctx, guest, data)
}

func (task *GuestChangeConfigTask) OnGuestSetNicTrafficsFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	task.markStageFailed(ctx, guest, data)
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
		drv, _ := guest.GetDriver()
		if err := drv.PerformStart(ctx, task.GetUserCred(), guest, nil, task.GetTaskId()); err != nil {
			task.OnGuestStartCompleteFailed(ctx, guest, jsonutils.NewString(err.Error()))
			return
		}
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
	guest.SetStatus(ctx, task.UserCred, api.VM_CHANGE_FLAVOR_FAIL, reason.String())
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
