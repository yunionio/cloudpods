package tasks

import (
	"context"

	"github.com/yunionio/jsonutils"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/lockman"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/compute/models"
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
	iResizeDisks, err := self.Params.Get("resize")
	if iResizeDisks == nil || err != nil {
		self.SetStageFailed(ctx, err.Error())
		return
	}
	resizeDisks := iResizeDisks.(*jsonutils.JSONArray)
	for i := 0; i < resizeDisks.Length(); i++ {
		iResizeSet, err := resizeDisks.GetAt(i)
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
			return
		}
		resizeSet := iResizeSet.(*jsonutils.JSONArray)
		diskId, err := resizeSet.GetAt(0)
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
			return
		}
		idStr, err := diskId.GetString()
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
			return
		}
		jSize, err := resizeSet.GetAt(1)
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
			return
		}
		size, err := jSize.Int()
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
			return
		}
		iDisk, err := models.DiskManager.FetchById(idStr)
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
			return
		}
		disk := iDisk.(*models.SDisk)
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
			return
		}
		if disk.DiskSize < int(size) {
			var pendingUsage models.SQuota
			err = self.GetPendingUsage(&pendingUsage)
			if err != nil {
				self.SetStageFailed(ctx, err.Error())
				return
			}
			disk.StartDiskResizeTask(ctx, self.UserCred, size, self.GetTaskId(), &pendingUsage)
			return
		}
	}
	guest := obj.(*models.SGuest)
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

func (self *GuestChangeConfigTask) OnCreateDisksComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	iVcpuCount, errCpu := self.Params.Get("vcpu_count")
	iVmemSize, errMem := self.Params.Get("vmem_size")
	var vcpuCount, vmemSize int64
	var err error
	guest := obj.(*models.SGuest)
	if errCpu == nil || errMem == nil {
		if iVcpuCount != nil {
			vcpuCount, err = iVcpuCount.Int()
			if err != nil {
				self.SetStageFailed(ctx, err.Error())
				return
			}
		}
		if iVmemSize != nil {
			vmemSize, err = iVmemSize.Int()
			if err != nil {
				self.SetStageFailed(ctx, err.Error())
				return
			}
		}
		err = guest.GetDriver().RequestChangeVmConfig(ctx, guest, self, vcpuCount, vmemSize)
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
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
			self.SetStageFailed(ctx, err.Error())
			return
		}
		var pendingUsage models.SQuota
		err = self.GetPendingUsage(&pendingUsage)
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
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
			self.SetStageFailed(ctx, err.Error())
			return
		}
		err = self.SetPendingUsage(&pendingUsage)
		if err != nil {
			self.SetStageFailed(ctx, err.Error())
			return
		}
	}
	self.SetStage("on_sync_status_complete", nil)
	err = guest.StartSyncstatus(ctx, self.UserCred, self.GetTaskId())
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
		return
	}
}

func (self *GuestChangeConfigTask) OnSyncStatusComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if guest.Status == models.VM_READY && jsonutils.QueryBoolean(self.Params, "auto_start", false) {
		self.SetStage("on_guest_start_complete", nil)
		guest.StartGueststartTask(ctx, self.UserCred, nil, self.GetTaskId())
	} else {
		dt := jsonutils.NewDict()
		dt.Add(jsonutils.NewString(guest.Id), "id")
		self.SetStageComplete(ctx, dt)
	}
}

func (self *GuestChangeConfigTask) OnGuestStartComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	dt := jsonutils.NewDict()
	dt.Add(jsonutils.NewString(guest.Id), "id")
	self.SetStageComplete(ctx, dt)
}
