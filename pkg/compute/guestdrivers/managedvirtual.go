package guestdrivers

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/util/secrules"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SManagedVirtualizedGuestDriver struct {
	SVirtualizedGuestDriver
}

type SManagedVMCreateConfig struct {
	Name              string
	ExternalImageId   string
	OsDistribution    string
	OsVersion         string
	Cpu               int
	Memory            int
	ExternalNetworkId string
	IpAddr            string
	Description       string
	StorageType       string
	SysDiskSize       int
	DataDisks         []int
	PublicKey         string
	SecGroupId        string
	SecGroupName      string
	SecRules          []secrules.SecurityRule
}

func (self *SManagedVirtualizedGuestDriver) GetJsonDescAtHost(ctx context.Context, guest *models.SGuest, host *models.SHost) jsonutils.JSONObject {
	config := SManagedVMCreateConfig{}
	config.Name = guest.Name
	config.Cpu = int(guest.VcpuCount)
	config.Memory = guest.VmemSize
	config.Description = guest.Description

	if len(guest.KeypairId) > 0 {
		config.PublicKey = guest.GetKeypairPublicKey()
	}

	nics := guest.GetNetworks()
	net := nics[0].GetNetwork()
	config.ExternalNetworkId = net.ExternalId
	config.IpAddr = nics[0].IpAddr

	config.SecGroupId = guest.SecgrpId
	config.SecGroupName = guest.GetSecgroupName()
	config.SecRules = guest.GetSecRules()

	disks := guest.GetDisks()
	config.DataDisks = make([]int, len(disks)-1)

	for i := 0; i < len(disks); i += 1 {
		disk := disks[i].GetDisk()
		if i == 0 {
			storage := disk.GetStorage()
			config.StorageType = storage.StorageType
			cache := storage.GetStoragecache()
			imageId := disk.GetTemplateId()
			//避免因同步过来的instance没有对应的imagecache信息，重置密码时引发空指针访问
			if scimg := models.StoragecachedimageManager.GetStoragecachedimage(cache.Id, imageId); scimg != nil {
				config.ExternalImageId = scimg.ExternalId
				img := scimg.GetCachedimage()
				config.OsDistribution, _ = img.Info.GetString("properties", "os_distribution")
				config.OsVersion, _ = img.Info.GetString("properties", "os_version")
			}
			config.SysDiskSize = disk.DiskSize / 1024 // MB => GB
		} else {
			config.DataDisks[i-1] = disk.DiskSize / 1024 // MB => GB
		}
	}
	return jsonutils.Marshal(&config)
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
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		ihost, err := host.GetIHost()
		if err != nil {
			return nil, err
		}
		ivm, err := ihost.GetIVMById(guest.ExternalId)
		if err != nil {
			return nil, err
		}
		err = ivm.StartVM()
		return nil, err
	})
	return nil, nil
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
	subtask, err := taskman.TaskManager.NewTask(ctx, "ManagedGuestRebuildRootTask", guest, task.GetUserCred(), task.GetParams(), task.GetTaskId(), "", nil)
	if err != nil {
		return err
	}
	subtask.ScheduleRun(nil)
	return nil
}
