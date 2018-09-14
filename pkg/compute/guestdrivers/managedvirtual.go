package guestdrivers

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SManagedVirtualizedGuestDriver struct {
	SVirtualizedGuestDriver
}

func (self *SManagedVirtualizedGuestDriver) RequestGuestCreateAllDisks(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	diskCat := guest.CategorizeDisks()
	var imageId string
	if diskCat.Root != nil {
		imageId = diskCat.Root.GetTemplateId()
	}
	if len(imageId) == 0 {
		task.ScheduleRun(nil)
		return nil
	}
	storage := diskCat.Root.GetStorage()
	if storage == nil {
		return fmt.Errorf("no valid storage")
	}
	storageCache := storage.GetStoragecache()
	if storageCache == nil {
		return fmt.Errorf("no valid storage cache")
	}
	return storageCache.StartImageCacheTask(ctx, task.GetUserCred(), imageId, false, task.GetTaskId())
}

func (self *SManagedVirtualizedGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	return nil
}

func (self *SManagedVirtualizedGuestDriver) OnGuestDeployTaskDataReceived(ctx context.Context, guest *models.SGuest, task taskman.ITask, data jsonutils.JSONObject) error {
	return nil
}

func (self *SManagedVirtualizedGuestDriver) RequestStartOnHost(_ context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential, task taskman.ITask) (jsonutils.JSONObject, error) {
	ihost, e := host.GetIHost()
	if e != nil {
		return nil, e
	}

	ivm, e := ihost.GetIVMById(guest.GetExternalId())
	if e != nil {
		return nil, e
	}

	result := jsonutils.NewDict()
	if ivm.GetStatus() != models.VM_RUNNING {
		if err := ivm.StartVM(); err != nil {
			return nil, e
		} else {
			task.ScheduleRun(result)
		}
	} else {
		result.Add(jsonutils.NewBool(true), "is_running")
	}

	return result, e
}

func (self *SManagedVirtualizedGuestDriver) RequestUndeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		ihost, err := host.GetIHost()
		if err != nil {
			return nil, err
		}
		ivm, err := ihost.GetIVMById(guest.ExternalId)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				return nil, nil
			} else {
				return nil, err
			}
		}
		err = ivm.DeleteVM()
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
	return nil
}

func (self *SManagedVirtualizedGuestDriver) RequestStopOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		ihost, err := host.GetIHost()
		if err != nil {
			return nil, err
		}
		ivm, err := ihost.GetIVMById(guest.ExternalId)
		if err != nil {
			return nil, err
		}
		err = ivm.StopVM(true)
		return nil, err
	})
	return nil
}

func (self *SManagedVirtualizedGuestDriver) RequestSyncstatusOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential) (jsonutils.JSONObject, error) {
	ihost, err := host.GetIHost()
	if err != nil {
		return nil, err
	}
	ivm, err := ihost.GetIVMById(guest.ExternalId)
	if err != nil {
		log.Errorf("fail to find ivm by id %s", err)
		return nil, err
	}

	status := ivm.GetStatus()
	switch status {
	case models.VM_RUNNING:
		status = cloudprovider.CloudVMStatusRunning
	case models.VM_READY:
		status = cloudprovider.CloudVMStatusStopped
	case models.VM_STARTING:
		status = cloudprovider.CloudVMStatusStopped
	case models.VM_STOPPING:
		status = cloudprovider.CloudVMStatusRunning
	default:
		status = cloudprovider.CloudVMStatusOther
	}

	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString(status), "status")
	return body, nil
}

func (self *SManagedVirtualizedGuestDriver) GetGuestVncInfo(userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost) (*jsonutils.JSONDict, error) {
	ihost, err := host.GetIHost()
	if err != nil {
		return nil, err
	}

	iVM, err := ihost.GetIVMById(guest.ExternalId)
	if err != nil {
		log.Errorf("cannot find vm %s %s", iVM, err)
		return nil, err
	}

	data, err := iVM.GetVNCInfo()
	if err != nil {
		return nil, err
	}

	dataDict := data.(*jsonutils.JSONDict)

	return dataDict, nil
}

func (self *SManagedVirtualizedGuestDriver) RequestRebuildRootDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	subtask, err := taskman.TaskManager.NewTask(ctx, "ManagedGuestRebuildRootTask", guest, task.GetUserCred(), nil, task.GetTaskId(), "", nil)
	if err != nil {
		return err
	}
	subtask.ScheduleRun(nil)
	return nil
}

func (self *SManagedVirtualizedGuestDriver) DoGuestCreateDisksTask(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	subtask, err := taskman.TaskManager.NewTask(ctx, "ManagedGuestCreateDiskTask", guest, task.GetUserCred(), task.GetParams(), task.GetTaskId(), "", nil)
	if err != nil {
		return err
	}
	subtask.ScheduleRun(nil)
	return nil
}

func (self *SManagedVirtualizedGuestDriver) RequestDeleteDetachedDisk(ctx context.Context, disk *models.SDisk, task taskman.ITask, isPurge bool) error {
	return disk.StartDiskDeleteTask(ctx, task.GetUserCred(), task.GetTaskId(), isPurge)
}

type SManagedVMChangeConfig struct {
	InstanceId string
	Cpu        int
	Memory     int
}

func (self *SManagedVirtualizedGuestDriver) RequestChangeVmConfig(ctx context.Context, guest *models.SGuest, task taskman.ITask, vcpuCount, vmemSize int64) error {
	config := SAliyunVMChangeConfig{}
	config.InstanceId = guest.GetExternalId()
	config.Cpu = int(vcpuCount)
	config.Memory = int(vmemSize)
	ihost, err := guest.GetHost().GetIHost()
	if err != nil {
		return err
	}

	iVM, err := ihost.GetIVMById(config.InstanceId)
	if err != nil {
		return err
	}

	if int(guest.VcpuCount) != config.Cpu || guest.VmemSize != config.Memory {
		err = iVM.ChangeConfig(config.InstanceId, config.Cpu, config.Memory)
		if err != nil {
			return err
		}
	}

	log.Debugf("VMchangeConfig %s, wait status ready ...", iVM.GetGlobalId())
	err = cloudprovider.WaitStatus(iVM, models.VM_READY, time.Second*5, time.Second*300)
	if err != nil {
		return err
	}
	log.Debugf("VMchangeConfig %s, and status is ready", iVM.GetGlobalId())
	return nil
}
