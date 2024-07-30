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
	"yunion.io/x/onecloud/pkg/cloudcommon/cmdline"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/conditionparser"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestBatchCreateTask struct {
	SSchedTask
}

func init() {
	taskman.RegisterTask(GuestBatchCreateTask{})
}

func (task *GuestBatchCreateTask) GetSchedParams() (*schedapi.ScheduleInput, error) {
	params := getBatchParamsAtIndex(task, 0)
	input, err := cmdline.FetchScheduleInputByJSON(params)
	if err != nil {
		return nil, fmt.Errorf("Unmarsh to schedule input: %v", err)
	}
	return input, err
}

func (task *GuestBatchCreateTask) GetDisks() ([]*api.DiskConfig, error) {
	input, err := task.GetSchedParams()
	if err != nil {
		return nil, err
	}
	return input.Disks, nil
}

func (task *GuestBatchCreateTask) GetFirstDisk() (*api.DiskConfig, error) {
	disks, err := task.GetDisks()
	if err != nil {
		return nil, err
	}
	if len(disks) == 0 {
		return nil, fmt.Errorf("Empty disks to schedule")
	}
	return disks[0], nil
}

func (task *GuestBatchCreateTask) GetCreateInput(data jsonutils.JSONObject) (*api.ServerCreateInput, error) {
	input := new(api.ServerCreateInput)
	err := data.Unmarshal(input)
	return input, err
}

func (task *GuestBatchCreateTask) clearPendingUsage(ctx context.Context, guest *models.SGuest) {
	ClearTaskPendingUsage(ctx, task)
	ClearTaskPendingRegionUsage(ctx, task)
}

func (task *GuestBatchCreateTask) OnInit(ctx context.Context, objs []db.IStandaloneModel, body jsonutils.JSONObject) {
	StartScheduleObjects(ctx, task, objs)
}

func (task *GuestBatchCreateTask) OnScheduleFailCallback(ctx context.Context, obj IScheduleModel, reason jsonutils.JSONObject, index int) {
	guest := obj.(*models.SGuest)
	if guest.DisableDelete.IsTrue() {
		guest.SetDisableDelete(task.UserCred, false)
	}
	task.clearPendingUsage(ctx, guest)
	task.SSchedTask.OnScheduleFailCallback(ctx, obj, reason, index)
}

func (task *GuestBatchCreateTask) SaveScheduleResultWithBackup(ctx context.Context, obj IScheduleModel, master, slave *schedapi.CandidateResource, index int) {
	guest := obj.(*models.SGuest)
	guest.SetHostIdWithBackup(task.UserCred, master.HostId, slave.HostId)
	task.SaveScheduleResult(ctx, obj, master, index)
}

func (task *GuestBatchCreateTask) allocateGuestOnHost(ctx context.Context, guest *models.SGuest, candidate *schedapi.CandidateResource, data *jsonutils.JSONDict) error {
	pendingUsage := models.SQuota{}
	err := task.GetPendingUsage(&pendingUsage, 0)
	if err != nil {
		log.Errorf("GetPendingUsage fail %s", err)
	}

	if conditionparser.IsTemplate(guest.Name) {
		guestInfo := guest.GetShortDesc(ctx)
		generateName := guest.GetMetadata(ctx, "generate_name", task.UserCred)
		if len(generateName) == 0 {
			generateName = guest.Name
		}
		newGenName, err := conditionparser.EvalTemplate(generateName, guestInfo)
		if err == nil {
			func() {
				lockman.LockRawObject(ctx, models.GuestManager.Keyword(), "name")
				defer lockman.ReleaseRawObject(ctx, models.GuestManager.Keyword(), "name")

				newName, err := db.GenerateName2(ctx, models.GuestManager,
					guest.GetOwnerId(), newGenName, guest, 1)
				if err == nil {
					_, err = db.Update(guest, func() error {
						guest.Name = newName
						return nil
					})
					if err != nil {
						log.Errorf("guest update name fail %s", err)
					}
				} else {
					log.Errorf("db.GenerateName2 fail %s", err)
				}
			}()
		} else {
			log.Errorf("conditionparser.EvalTemplate fail %s", err)
		}
	}

	if len(candidate.CpuNumaPin) > 0 {
		if err := guest.SetCpuNumaPin(ctx, task.UserCred, candidate.CpuNumaPin, nil); err != nil {
			log.Errorf("SetCpuNumaPin fail %s", err)
			guest.SetStatus(ctx, task.UserCred, api.VM_CREATE_FAILED, err.Error())
			return err
		}
	}

	host, _ := guest.GetHost()

	quotaCpuMem := models.SQuota{Count: 1, Cpu: int(guest.VcpuCount), Memory: guest.VmemSize}
	keys, err := guest.GetQuotaKeys()
	if err != nil {
		log.Errorf("guest.GetQuotaKeys fail %s", err)
	}
	quotaCpuMem.SetKeys(keys)
	err = quotas.CancelPendingUsage(ctx, task.UserCred, &pendingUsage, &quotaCpuMem, true) // success
	task.SetPendingUsage(&pendingUsage, 0)

	input, err := task.GetCreateInput(data)
	if err != nil {
		log.Errorf("GetCreateInput fail %s", err)
		guest.SetStatus(ctx, task.UserCred, api.VM_CREATE_FAILED, err.Error())
		return err
	}

	if host.IsPrepaidRecycle() {
		input, err = host.SetGuestCreateNetworkAndDiskParams(ctx, task.UserCred, input)
		if err != nil {
			log.Errorf("host.SetGuestCreateNetworkAndDiskParams fail %s", err)
			guest.SetStatus(ctx, task.UserCred, api.VM_CREATE_FAILED, err.Error())
			return err
		}
		// params := input.JSON(input)
		// task.SaveParams(params)
	}

	/*input, err = task.GetCreateInput(data)
	if err != nil {
		guest.SetStatus(ctx,task.UserCred, api.VM_CREATE_FAILED, err.Error())
		return err
	}*/

	pendingRegionUsage := models.SRegionQuota{}
	task.GetPendingUsage(&pendingRegionUsage, 1)
	// allocate networks
	err = guest.CreateNetworksOnHost(ctx, task.UserCred, host, input.Networks, &pendingRegionUsage, &pendingUsage, candidate.Nets)
	task.SetPendingUsage(&pendingUsage, 0)
	task.SetPendingUsage(&pendingRegionUsage, 1)
	if err != nil {
		log.Errorf("Network failed: %s", err)
		guest.SetStatus(ctx, task.UserCred, api.VM_NETWORK_FAILED, err.Error())
		return err
	}

	if input.PublicIpBw > 0 {
		input.Eip, input.EipBw = "", 0
	}

	// allocate eips
	if input.EipBw > 0 {
		eip, err := models.ElasticipManager.NewEipForVMOnHost(ctx, task.UserCred, &models.NewEipForVMOnHostArgs{
			Bandwidth:     input.EipBw,
			BgpType:       input.EipBgpType,
			ChargeType:    input.EipChargeType,
			AutoDellocate: input.EipAutoDellocate,

			Guest:        guest,
			Host:         host,
			PendingUsage: &pendingRegionUsage,
		})
		task.SetPendingUsage(&pendingRegionUsage, 1)
		if err != nil {
			log.Errorf("guest.CreateElasticipOnHost failed: %s", err)
			guest.SetStatus(ctx, task.UserCred, api.VM_NETWORK_FAILED, err.Error())
			return err
		}
		input.Eip = eip.Id
	}

	drv, err := guest.GetDriver()
	if err != nil {
		guest.SetStatus(ctx, task.UserCred, api.VM_DISK_FAILED, err.Error())
		return err
	}

	// allocate disks
	extraDisks, err := drv.PrepareDiskRaidConfig(task.UserCred, host, input.BaremetalDiskConfigs, input.Disks)
	if err != nil {
		log.Errorf("PrepareDiskRaidConfig fail: %s", err)
		guest.SetStatus(ctx, task.UserCred, api.VM_DISK_FAILED, err.Error())
		return err
	}
	if len(extraDisks) > 0 {
		input.Disks = append(input.Disks, extraDisks...)
	}

	var backupCandidateDisks []*schedapi.CandidateDisk
	if candidate.BackupCandidate != nil {
		backupCandidateDisks = candidate.BackupCandidate.Disks
	}
	// 纳管的云需要有关联关系后,在做deploy时才有磁盘的信息
	err = guest.CreateDisksOnHost(ctx, task.UserCred, host, input.Disks, &pendingUsage, true, true, candidate.Disks, backupCandidateDisks, true)
	task.SetPendingUsage(&pendingUsage, 0)

	if err != nil {
		log.Errorf("Disk create failed: %s", err)
		guest.SetStatus(ctx, task.UserCred, api.VM_DISK_FAILED, err.Error())
		return err
	}

	// allocate GPUs
	err = guest.CreateIsolatedDeviceOnHost(ctx, task.UserCred, host, input.IsolatedDevices, &pendingUsage)
	task.SetPendingUsage(&pendingUsage, 0)
	if err != nil {
		log.Errorf("IsolatedDevices create failed: %s", err)
		guest.SetStatus(ctx, task.UserCred, api.VM_DEVICE_FAILED, err.Error())
		return err
	}

	// join groups
	if input.InstanceGroupIds != nil && len(input.InstanceGroupIds) != 0 {
		err := guest.JoinGroups(ctx, task.UserCred, input.InstanceGroupIds)
		if err != nil {
			log.Errorf("Join Groups failed: %v", err)
			guest.SetStatus(ctx, task.UserCred, api.VM_CREATE_FAILED, err.Error())
			return err
		}
	}

	if guest.IsPrepaidRecycle() {
		err := host.RebuildRecycledGuest(ctx, task.UserCred, guest)
		if err != nil {
			log.Errorf("start guest create task fail %s", err)
			guest.SetStatus(ctx, task.UserCred, api.VM_CREATE_FAILED, err.Error())
			return err
		}

		autoStart := input.AutoStart
		resetPassword := true
		if input.ResetPassword != nil {
			resetPassword = *input.ResetPassword
		}
		deployInput := &api.ServerDeployInputBase{}
		deployInput.Password = input.Password
		deployInput.ResetPassword = resetPassword

		err = guest.StartRebuildRootTask(ctx, task.UserCred, "", false, autoStart, true, deployInput)
		if err != nil {
			log.Errorf("start guest create task fail %s", err)
			guest.SetStatus(ctx, task.UserCred, api.VM_CREATE_FAILED, err.Error())
			return err
		}
		return nil
	}

	err = guest.StartGuestCreateTask(ctx, task.UserCred, input, nil, task.GetId())
	if err != nil {
		log.Errorf("start guest create task fail %s", err)
		guest.SetStatus(ctx, task.UserCred, api.VM_CREATE_FAILED, err.Error())
		return err
	}
	return nil
}

func (task *GuestBatchCreateTask) SaveScheduleResult(ctx context.Context, obj IScheduleModel, candidate *schedapi.CandidateResource, index int) {
	var err error
	hostId := candidate.HostId
	guest := obj.(*models.SGuest)
	if len(guest.HostId) == 0 {
		guest.OnScheduleToHost(ctx, task.UserCred, hostId)
	}

	data := getBatchParamsAtIndex(task, index)

	err = task.allocateGuestOnHost(ctx, guest, candidate, data)
	if err != nil {
		task.clearPendingUsage(ctx, guest)
		db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE_FAIL, err, task.UserCred)
		logclient.AddActionLogWithStartable(task, obj, logclient.ACT_ALLOCATE, err, task.GetUserCred(), false)
		notifyclient.EventNotify(ctx, task.GetUserCred(), notifyclient.SEventNotifyParam{
			Obj:    guest,
			Action: notifyclient.ActionCreateBackupServer,
			IsFail: true,
		})
		task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
	}
}

func (task *GuestBatchCreateTask) OnScheduleComplete(ctx context.Context, items []db.IStandaloneModel, data *jsonutils.JSONDict) {
	task.SetStageComplete(ctx, nil)
}
