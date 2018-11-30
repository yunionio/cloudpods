package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
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
		self.SetStage("on_disks_resize_complete", nil)
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
			err = disk.StartDiskResizeTask(ctx, self.UserCred, size, self.GetTaskId(), &pendingUsage)
			if err != nil {
				self.markStageFailed(ctx, guest, fmt.Sprintf("disk.StartDiskResizeTask fail %s", err))
				return
			}
			return
		}
	}

	self.DoCreateDisksTask(ctx, guest)
}

func (self *GuestChangeConfigTask) DoCreateDisksTask(ctx context.Context, guest *models.SGuest) {
	iCreateData, err := self.Params.Get("create")
	if err != nil || iCreateData == nil {
		self.OnCreateDisksComplete(ctx, guest, nil)
		return
	}
	data := (iCreateData).(*jsonutils.JSONDict)
	self.SetStage("on_create_disks_complete", nil)
	guest.StartGuestCreateDiskTask(ctx, self.UserCred, data, self.GetTaskId())
}

func (self *GuestChangeConfigTask) OnCreateDisksCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.markStageFailed(ctx, guest, fmt.Sprintf("OnCreateDisksCompleteFailed %s", err))
}

func (self *GuestChangeConfigTask) OnCreateDisksComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	var vcpuCount, vmemSize int64
	var paramsError error
	var err error
	iSku, paramsError := self.Params.GetString("sku_id")
	if paramsError == nil {
		isku, err := models.ServerSkuManager.FetchById(iSku)
		if err != nil {
			self.markStageFailed(ctx, guest, fmt.Sprintf("Sku_id fail %s", err))
			logclient.AddActionLog(guest, logclient.ACT_VM_CHANGE_FLAVOR, err, self.UserCred, false)
			return
		}

		sku := isku.(*models.SServerSku)
		self.Params.Set("instance_type", jsonutils.NewString(sku.GetName()))
		vcpuCount = int64(sku.CpuCoreCount)
		vmemSize = int64(sku.MemorySizeMB)
	} else {
		iVcpuCount, cpuError := self.Params.Get("vcpu_count")
		if iVcpuCount != nil {
			vcpuCount, err = iVcpuCount.Int()
			if err != nil {
				self.markStageFailed(ctx, guest, fmt.Sprintf("iVcpuCount.Int() fail %s", err))
				return
			}
		}

		iVmemSize, memError := self.Params.Get("vmem_size")
		if iVmemSize != nil {
			vmemSize, err = iVmemSize.Int()
			if err != nil {
				self.markStageFailed(ctx, guest, fmt.Sprintf("iVmemSize.Int fail %s", err))
				return
			}
		}

		if cpuError == nil || memError == nil {
			paramsError = nil
		}
	}

	if paramsError == nil {
		err = guest.GetDriver().RequestChangeVmConfig(ctx, guest, self, vcpuCount, vmemSize)
		if err != nil {
			self.markStageFailed(ctx, guest, fmt.Sprintf("guest.GetDriver().RequestChangeVmConfig fail %s", err))
			return
		}
		var addCpu, addMem = 0, 0
		if vcpuCount > 0 {
			addCpu = int(vcpuCount - int64(guest.VcpuCount))
			if addCpu < 0 {
				addCpu = 0
			}
		}
		if vmemSize > 0 {
			addMem = int(vmemSize - int64(guest.VmemSize))
			if addMem < 0 {
				addMem = 0
			}
		}
		_, err = guest.GetModelManager().TableSpec().Update(guest, func() error {
			if vcpuCount > 0 {
				guest.VcpuCount = int8(vcpuCount)
			}
			if vmemSize > 0 {
				guest.VmemSize = int(vmemSize)
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
		// ownerCred := guest.GetOwnerUserCred()
		var cancelUsage models.SQuota
		if addCpu > 0 {
			cancelUsage.Cpu = addCpu
		}
		if addMem > 0 {
			cancelUsage.Memory = addMem
		}

		lockman.LockClass(ctx, guest.GetModelManager(), guest.ProjectId)
		defer lockman.ReleaseClass(ctx, guest.GetModelManager(), guest.ProjectId)

		err = models.QuotaManager.CancelPendingUsage(ctx, self.UserCred, guest.ProjectId, &pendingUsage, &cancelUsage)
		if err != nil {
			self.markStageFailed(ctx, guest, fmt.Sprintf("CancelPendingUsage fail %s", err))
			return
		}
		err = self.SetPendingUsage(&pendingUsage)
		if err != nil {
			self.markStageFailed(ctx, guest, fmt.Sprintf("SetPendingUsage fail %s", err))
			return
		}
	}
	self.SetStage("on_sync_status_complete", nil)
	err = guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
	if err != nil {
		self.markStageFailed(ctx, guest, fmt.Sprintf("StartSyncstatus fail %s", err))
		return
	}
}

func (self *GuestChangeConfigTask) OnSyncStatusComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if guest.Status == models.VM_READY && jsonutils.QueryBoolean(self.Params, "auto_start", false) {
		self.SetStage("on_guest_start_complete", nil)
		guest.StartGueststartTask(ctx, self.UserCred, nil, self.GetTaskId())
		logclient.AddActionLog(guest, logclient.ACT_VM_CHANGE_FLAVOR, "", self.UserCred, true)
	} else {
		dt := jsonutils.NewDict()
		dt.Add(jsonutils.NewString(guest.Id), "id")
		logclient.AddActionLog(guest, logclient.ACT_VM_CHANGE_FLAVOR, "", self.UserCred, true)
		self.SetStageComplete(ctx, dt)
	}
}

func (self *GuestChangeConfigTask) OnGuestStartComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	dt := jsonutils.NewDict()
	dt.Add(jsonutils.NewString(guest.Id), "id")
	self.SetStageComplete(ctx, dt)
}

func (self *GuestChangeConfigTask) markStageFailed(ctx context.Context, guest *models.SGuest, reason string) {
	guest.SetStatus(self.UserCred, models.VM_CHANGE_FLAVOR_FAIL, reason)
	db.OpsLog.LogEvent(guest, db.ACT_CHANGE_FLAVOR_FAIL, reason, self.UserCred)
	logclient.AddActionLog(guest, logclient.ACT_VM_CHANGE_FLAVOR, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}
