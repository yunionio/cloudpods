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
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type GuestChangeConfigTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestChangeConfigTask{})
}

func (self *GuestChangeConfigTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	_, err := self.Params.Get("resize")
	if err == nil {
		self.SetStage("OnDisksResizeComplete", nil)
		self.OnDisksResizeComplete(ctx, obj, data)
	} else {
		guest := obj.(*models.SGuest)
		self.DoCreateDisksTask(ctx, guest)
	}
}

func (self *GuestChangeConfigTask) OnDisksResizeComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	iResizeDisks, err := self.Params.Get("resize")
	if iResizeDisks == nil || err != nil {
		self.markStageFailed(ctx, guest, err.Error())
		return
	}
	resizeDisks := iResizeDisks.(*jsonutils.JSONArray)
	for i := 0; i < resizeDisks.Length(); i++ {
		iResizeSet, err := resizeDisks.GetAt(i)
		if err != nil {
			self.markStageFailed(ctx, guest, fmt.Sprintf("resizeDisks.GetAt fail %s", err))
			return
		}
		resizeSet := iResizeSet.(*jsonutils.JSONArray)
		diskId, err := resizeSet.GetAt(0)
		if err != nil {
			self.markStageFailed(ctx, guest, fmt.Sprintf("resizeSet.GetAt(0) fail %s", err))
			return
		}
		idStr, err := diskId.GetString()
		if err != nil {
			self.markStageFailed(ctx, guest, fmt.Sprintf("diskId.GetString fail %s", err))
			return
		}
		jSize, err := resizeSet.GetAt(1)
		if err != nil {
			self.markStageFailed(ctx, guest, fmt.Sprintf("resizeSet.GetAt(1) fail %s", err))
			return
		}
		size, err := jSize.Int()
		if err != nil {
			self.markStageFailed(ctx, guest, fmt.Sprintf("jSize.Int fail %s", err))
			return
		}
		iDisk, err := models.DiskManager.FetchById(idStr)
		if err != nil {
			self.markStageFailed(ctx, guest, fmt.Sprintf("models.DiskManager.FetchById(idStr) fail %s", err))
			return
		}
		disk := iDisk.(*models.SDisk)
		if disk.DiskSize < int(size) {
			var pendingUsage models.SQuota
			err = self.GetPendingUsage(&pendingUsage)
			if err != nil {
				self.markStageFailed(ctx, guest, fmt.Sprintf("self.GetPendingUsage(&pendingUsage) fail %s", err))
				return
			}
			err = guest.StartGuestDiskResizeTask(ctx, self.UserCred, disk.Id, size, self.GetTaskId(), &pendingUsage)
			if err != nil {
				self.markStageFailed(ctx, guest, fmt.Sprintf("guest.StartGuestDiskResizeTask fail %s", err))
				return
			}
			return
		}
	}

	self.DoCreateDisksTask(ctx, guest)
}

func (self *GuestChangeConfigTask) DoCreateDisksTask(ctx context.Context, guest *models.SGuest) {
	disks := make([]*api.DiskConfig, 0)
	err := self.Params.Unmarshal(&disks, "create")
	if err != nil || len(disks) == 0 {
		self.OnCreateDisksComplete(ctx, guest, nil)
		return
	}
	self.SetStage("OnCreateDisksComplete", nil)
	guest.StartGuestCreateDiskTask(ctx, self.UserCred, disks, self.GetTaskId())
}

func (self *GuestChangeConfigTask) OnCreateDisksCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.markStageFailed(ctx, guest, fmt.Sprintf("OnCreateDisksCompleteFailed %s", err))
}

func (self *GuestChangeConfigTask) OnCreateDisksComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	if self.Params.Contains("instance_type") || self.Params.Contains("vcpu_count") || self.Params.Contains("vmem_size") {
		self.SetStage("OnGuestChangeCpuMemSpecComplete", nil)
		instanceType, _ := self.Params.GetString("instance_type")
		vcpuCount, _ := self.Params.Int("vcpu_count")
		vmemSize, _ := self.Params.Int("vmem_size")
		if len(instanceType) > 0 {
			provider := guest.GetDriver().GetProvider()
			sku, err := models.ServerSkuManager.FetchSkuByNameAndProvider(instanceType, provider, false)
			if err != nil {
				self.markStageFailed(ctx, guest, fmt.Sprintf("models.ServerSkuManager.FetchSkuByNameAndProvider error %s", err))
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
		self.startGuestChangeCpuMemSpec(ctx, guest, instanceType, vcpuCount, vmemSize)
	} else {
		self.OnGuestChangeCpuMemSpecComplete(ctx, obj, data)
	}
}

func (self *GuestChangeConfigTask) startGuestChangeCpuMemSpec(ctx context.Context, guest *models.SGuest, instanceType string, vcpuCount int64, vmemSize int64) {
	err := guest.GetDriver().RequestChangeVmConfig(ctx, guest, self, instanceType, vcpuCount, vmemSize)
	if err != nil {
		self.markStageFailed(ctx, guest, fmt.Sprintf("guest.GetDriver().RequestChangeVmConfig fail %s", err))
		return
	}
}

func (self *GuestChangeConfigTask) OnGuestChangeCpuMemSpecComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	instanceType, _ := self.Params.GetString("instance_type")
	vcpuCount, _ := self.Params.Int("vcpu_count")
	vmemSize, _ := self.Params.Int("vmem_size")

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
		self.markStageFailed(ctx, guest, fmt.Sprintf("Update fail %s", err))
		return
	}

	var pendingUsage models.SQuota
	err = self.GetPendingUsage(&pendingUsage)
	if err != nil {
		self.markStageFailed(ctx, guest, fmt.Sprintf("GetPendingUsage %s", err))
		return
	}
	var cancelUsage models.SQuota
	if addCpu > 0 {
		cancelUsage.Cpu = addCpu
	}
	if addMem > 0 {
		cancelUsage.Memory = addMem
	}

	lockman.LockClass(ctx, guest.GetModelManager(), guest.ProjectId)
	defer lockman.ReleaseClass(ctx, guest.GetModelManager(), guest.ProjectId)

	err = models.QuotaManager.CancelPendingUsage(ctx, self.UserCred, rbacutils.ScopeProject, guest.GetOwnerId(), guest.GetQuotaPlatformID(), &pendingUsage, &cancelUsage)
	if err != nil {
		self.markStageFailed(ctx, guest, fmt.Sprintf("CancelPendingUsage fail %s", err))
		return
	}
	err = self.SetPendingUsage(&pendingUsage)
	if err != nil {
		self.markStageFailed(ctx, guest, fmt.Sprintf("SetPendingUsage fail %s", err))
		return
	}

	self.OnGuestChangeCpuMemSpecFinish(ctx, guest)
}

func (self *GuestChangeConfigTask) OnGuestChangeCpuMemSpecCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if err := guest.GetDriver().OnGuestChangeCpuMemFailed(ctx, guest, data.(*jsonutils.JSONDict), self); err != nil {
		log.Errorln(err)
	}
	self.markStageFailed(ctx, guest, fmt.Sprintf("guest.GetDriver().RequestChangeVmConfig fail %s", data))
}

func (self *GuestChangeConfigTask) OnGuestChangeCpuMemSpecFinish(ctx context.Context, guest *models.SGuest) {
	models.HostManager.ClearSchedDescCache(guest.HostId)
	self.SetStage("OnSyncConfigComplete", nil)
	err := guest.StartSyncTaskWithoutSyncstatus(ctx, self.UserCred, false, self.GetTaskId())
	if err != nil {
		self.markStageFailed(ctx, guest, fmt.Sprintf("StartSyncstatus fail %s", err))
		return
	}
}

func (self *GuestChangeConfigTask) OnSyncConfigComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)

	self.SetStage("on_sync_status_complete", nil)
	err := guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
	if err != nil {
		self.markStageFailed(ctx, guest, fmt.Sprintf("StartSyncstatus fail %s", err))
		return
	}
}

func (self *GuestChangeConfigTask) OnSyncStatusComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if guest.Status == api.VM_READY && jsonutils.QueryBoolean(self.Params, "auto_start", false) {
		self.SetStage("OnGuestStartComplete", nil)
		guest.StartGueststartTask(ctx, self.UserCred, nil, self.GetTaskId())
	} else {
		dt := jsonutils.NewDict()
		dt.Add(jsonutils.NewString(guest.Id), "id")
		self.SetStageComplete(ctx, dt)
	}
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_CHANGE_FLAVOR, "", self.UserCred, true)
}

func (self *GuestChangeConfigTask) OnGuestStartComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	dt := jsonutils.NewDict()
	dt.Add(jsonutils.NewString(guest.Id), "id")
	self.SetStageComplete(ctx, dt)
}

func (self *GuestChangeConfigTask) OnGuestStartCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data.String())
}

func (self *GuestChangeConfigTask) markStageFailed(ctx context.Context, guest *models.SGuest, reason string) {
	guest.SetStatus(self.UserCred, api.VM_CHANGE_FLAVOR_FAIL, reason)
	db.OpsLog.LogEvent(guest, db.ACT_CHANGE_FLAVOR_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_CHANGE_FLAVOR, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}
