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

type GuestBatchCreateTask struct {
	SSchedTask
}

func init() {
	taskman.RegisterTask(GuestBatchCreateTask{})
}

func (self *GuestBatchCreateTask) GetCreateInput() (*api.ServerCreateInput, error) {
	input := new(api.ServerCreateInput)
	err := self.GetParams().Unmarshal(input)
	return input, err
}

func (self *GuestBatchCreateTask) clearPendingUsage(ctx context.Context, guest *models.SGuest) {
	ClearTaskPendingUsage(ctx, self)
	ClearTaskPendingRegionUsage(ctx, self)
}

func (self *GuestBatchCreateTask) OnInit(ctx context.Context, objs []db.IStandaloneModel, body jsonutils.JSONObject) {
	StartScheduleObjects(ctx, self, objs)
}

func (self *GuestBatchCreateTask) OnScheduleFailCallback(ctx context.Context, obj IScheduleModel, reason string) {
	self.SSchedTask.OnScheduleFailCallback(ctx, obj, reason)
	guest := obj.(*models.SGuest)
	if guest.DisableDelete.IsTrue() {
		guest.SetDisableDelete(self.UserCred, false)
	}
	self.clearPendingUsage(ctx, guest)
}

func (self *GuestBatchCreateTask) SaveScheduleResultWithBackup(ctx context.Context, obj IScheduleModel, master, slave *schedapi.CandidateResource) {
	guest := obj.(*models.SGuest)
	guest.SetHostIdWithBackup(self.UserCred, master.HostId, slave.HostId)
	self.SaveScheduleResult(ctx, obj, master)
}

func (self *GuestBatchCreateTask) allocateGuestOnHost(ctx context.Context, guest *models.SGuest, candidate *schedapi.CandidateResource) error {
	pendingUsage := models.SQuota{}
	err := self.GetPendingUsage(&pendingUsage, 0)
	if err != nil {
		log.Errorf("GetPendingUsage fail %s", err)
	}

	host := guest.GetHost()

	quotaCpuMem := models.SQuota{Count: 1, Cpu: int(guest.VcpuCount), Memory: guest.VmemSize}
	keys, err := guest.GetQuotaKeys()
	if err != nil {
		log.Errorf("guest.GetQuotaKeys fail %s", err)
	}
	quotaCpuMem.SetKeys(keys)
	err = quotas.CancelPendingUsage(ctx, self.UserCred, &pendingUsage, &quotaCpuMem)
	self.SetPendingUsage(&pendingUsage, 0)

	input, err := self.GetCreateInput()
	if err != nil {
		log.Errorf("GetCreateInput fail %s", err)
	}

	if host.IsPrepaidRecycle() {
		input, err = host.SetGuestCreateNetworkAndDiskParams(ctx, self.UserCred, input)
		if err != nil {
			log.Errorf("host.SetGuestCreateNetworkAndDiskParams fail %s", err)
			guest.SetStatus(self.UserCred, api.VM_CREATE_FAILED, err.Error())
			return err
		}
		params := input.JSON(input)
		self.SaveParams(params)
	}

	input, err = self.GetCreateInput()
	if err != nil {
		guest.SetStatus(self.UserCred, api.VM_CREATE_FAILED, err.Error())
		return err
	}

	pendingRegionUsage := models.SRegionQuota{}
	self.GetPendingUsage(&pendingRegionUsage, 1)
	// allocate networks
	err = guest.CreateNetworksOnHost(ctx, self.UserCred, host, input.Networks, &pendingRegionUsage, candidate.Nets)
	self.SetPendingUsage(&pendingRegionUsage, 1)
	if err != nil {
		log.Errorf("Network failed: %s", err)
		guest.SetStatus(self.UserCred, api.VM_NETWORK_FAILED, err.Error())
		return err
	}

	// allocate eips
	if input.EipBw > 0 {
		eip, err := models.ElasticipManager.NewEipForVMOnHost(ctx, self.UserCred, guest, host, input.EipBw, input.EipChargeType, &pendingRegionUsage)
		self.SetPendingUsage(&pendingRegionUsage, 1)
		if err != nil {
			log.Errorf("guest.CreateElasticipOnHost failed: %s", err)
			guest.SetStatus(self.UserCred, api.VM_NETWORK_FAILED, err.Error())
			return err
		}
		input.Eip = eip.Id
	}

	// allocate disks
	extraDisks, err := guest.GetDriver().PrepareDiskRaidConfig(self.UserCred, host, input.BaremetalDiskConfigs, input.Disks)
	if err != nil {
		log.Errorf("PrepareDiskRaidConfig fail: %s", err)
		guest.SetStatus(self.UserCred, api.VM_DISK_FAILED, err.Error())
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
	err = guest.CreateDisksOnHost(ctx, self.UserCred, host, input.Disks, &pendingUsage, true, true, candidate.Disks, backupCandidateDisks, true)
	self.SetPendingUsage(&pendingUsage, 0)

	if err != nil {
		log.Errorf("Disk create failed: %s", err)
		guest.SetStatus(self.UserCred, api.VM_DISK_FAILED, err.Error())
		return err
	}

	// allocate GPUs
	err = guest.CreateIsolatedDeviceOnHost(ctx, self.UserCred, host, input.IsolatedDevices, &pendingUsage)
	self.SetPendingUsage(&pendingUsage, 0)
	if err != nil {
		log.Errorf("IsolatedDevices create failed: %s", err)
		guest.SetStatus(self.UserCred, api.VM_DEVICE_FAILED, err.Error())
		return err
	}

	// join groups
	if input.InstanceGroupIds != nil && len(input.InstanceGroupIds) != 0 {
		err := guest.JoinGroups(ctx, self.UserCred, input.InstanceGroupIds)
		if err != nil {
			log.Errorf("Join Groups failed: %v", err)
			guest.SetStatus(self.UserCred, api.VM_CREATE_FAILED, err.Error())
			return err
		}
	}

	if guest.IsPrepaidRecycle() {
		err := host.RebuildRecycledGuest(ctx, self.UserCred, guest)
		if err != nil {
			log.Errorf("start guest create task fail %s", err)
			guest.SetStatus(self.UserCred, api.VM_CREATE_FAILED, err.Error())
			return err
		}

		autoStart := input.AutoStart
		resetPassword := true
		if input.ResetPassword != nil {
			resetPassword = *input.ResetPassword
		}
		passwd := input.Password
		err = guest.StartRebuildRootTask(ctx, self.UserCred, "", false, autoStart, passwd, resetPassword, true)
		if err != nil {
			log.Errorf("start guest create task fail %s", err)
			guest.SetStatus(self.UserCred, api.VM_CREATE_FAILED, err.Error())
			return err
		}
		return nil
	}

	err = guest.StartGuestCreateTask(ctx, self.UserCred, input, nil, self.GetId())
	if err != nil {
		log.Errorf("start guest create task fail %s", err)
		guest.SetStatus(self.UserCred, api.VM_CREATE_FAILED, err.Error())
		return err
	}
	return nil
}

func (self *GuestBatchCreateTask) SaveScheduleResult(ctx context.Context, obj IScheduleModel, candidate *schedapi.CandidateResource) {
	var err error
	hostId := candidate.HostId
	guest := obj.(*models.SGuest)
	if len(guest.HostId) == 0 {
		guest.OnScheduleToHost(ctx, self.UserCred, hostId)
	}

	err = self.allocateGuestOnHost(ctx, guest, candidate)
	if err != nil {
		self.clearPendingUsage(ctx, guest)
		db.OpsLog.LogEvent(guest, db.ACT_ALLOCATE_FAIL, err, self.UserCred)
		logclient.AddActionLogWithStartable(self, obj, logclient.ACT_ALLOCATE, err.Error(), self.GetUserCred(), false)
		notifyclient.NotifySystemError(guest.Id, guest.Name, api.VM_CREATE_FAILED, err.Error())
		self.SetStageFailed(ctx, err.Error())
	}
}

func (self *GuestBatchCreateTask) OnScheduleComplete(ctx context.Context, items []db.IStandaloneModel, data *jsonutils.JSONDict) {
	self.SetStageComplete(ctx, nil)
}
