package guestdrivers

import (
	"context"
	"fmt"
	"time"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/onecloud/pkg/httperrors"
	"github.com/yunionio/pkg/util/seclib"
	"github.com/yunionio/pkg/utils"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db/taskman"
	"github.com/yunionio/onecloud/pkg/cloudprovider"
	"github.com/yunionio/onecloud/pkg/compute/models"
)

type SAliyunGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SAliyunGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SAliyunGuestDriver) GetHypervisor() string {
	return models.HYPERVISOR_ALIYUN
}

func (self *SAliyunGuestDriver) ChooseHostStorage(host *models.SHost, backend string) *models.SStorage {
	storages := host.GetAttachedStorages()
	for i := 0; i < len(storages); i += 1 {
		if storages[i].StorageType == backend {
			return &storages[i]
		}
	}
	for _, stype := range []string{"cloud_efficiency", "cloud_ssd", "cloud", "ephemeral_ssd"} {
		for i := 0; i < len(storages); i += 1 {
			if storages[i].StorageType == stype {
				return &storages[i]
			}
		}
	}
	return nil
}

func (self *SAliyunGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, data)
	if err != nil {
		return nil, err
	}
	if data.Contains("net.0") && data.Contains("net.1") {
		return nil, httperrors.NewInputParameterError("cannot support more than 1 nic")
	}
	return data, nil
}

type SAliyunVMCreateConfig struct {
	Name              string
	ExternalImageId   string
	Cpu               int
	Memory            int
	ExternalNetworkId string
	IpAddr            string
	Description       string
	StorageType       string
	SysDiskSize       int
	DataDisks         []int
	PublicKey         string
}

func (self *SAliyunGuestDriver) GetJsonDescAtHost(ctx context.Context, guest *models.SGuest, host *models.SHost) jsonutils.JSONObject {
	config := SAliyunVMCreateConfig{}
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

	disks := guest.GetDisks()
	config.DataDisks = make([]int, len(disks)-1)

	for i := 0; i < len(disks); i += 1 {
		disk := disks[i].GetDisk()
		if i == 0 {
			storage := disk.GetStorage()
			config.StorageType = storage.StorageType
			cache := storage.GetStoragecache()
			imageId := disk.GetTemplateId()
			scimg := models.StoragecachedimageManager.GetStoragecachedimage(cache.Id, imageId)
			config.ExternalImageId = scimg.ExternalId
			config.SysDiskSize = disk.DiskSize / 1024 // MB => GB
		} else {
			config.DataDisks[i-1] = disk.DiskSize / 1024 // MB => GB
		}
	}

	return jsonutils.Marshal(&config)
}

type SDiskInfo struct {
	Size int
	Uuid string
}

func (self *SAliyunGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	config := guest.GetDeployConfigOnHost(ctx, host, task.GetParams())

	onfinish, err := config.GetString("on_finish")
	if err != nil {
		return err
	}
	action, err := config.GetString("action")
	if err != nil {
		return err
	}

	if action != "create" {
		return fmt.Errorf("Action %s not supported", action)
	}

	ihost, err := host.GetIHost()
	if err != nil {
		return err
	}

	desc := SAliyunVMCreateConfig{}
	err = config.Unmarshal(&desc, "desc")
	if err != nil {
		return err
	}

	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		passwd := seclib.RandomPassword(12)

		iVM, err := ihost.CreateVM(desc.Name, desc.ExternalImageId, desc.SysDiskSize, desc.Cpu, desc.Memory, desc.ExternalNetworkId,
			desc.IpAddr, desc.Description, passwd, desc.StorageType, desc.DataDisks, desc.PublicKey)
		if err != nil {
			return nil, err
		}
		log.Debugf("VMcreated %s, wait status ready ...", iVM.GetGlobalId())
		err = cloudprovider.WaitStatus(iVM, models.VM_READY, time.Second*5, time.Second*1800)
		if err != nil {
			return nil, err
		}
		log.Debugf("VMcreated %s, and status is ready", iVM.GetGlobalId())

		iVM, err = ihost.GetIVMById(iVM.GetGlobalId())
		if err != nil {
			log.Errorf("cannot find vm %s", err)
			return nil, err
		}

		if onfinish == "none" {
			err = iVM.StartVM()
			if err != nil {
				return nil, err
			}
		}

		encpasswd, err := utils.EncryptAESBase64(guest.Id, passwd)
		if err != nil {
			log.Errorf("encrypt password failed %s", err)
		}

		data := jsonutils.NewDict()
		data.Add(jsonutils.NewString(iVM.GetOSType()), "os")
		data.Add(jsonutils.NewString("root"), "account")
		data.Add(jsonutils.NewString(encpasswd), "key")

		idisks, err := iVM.GetIDisks()

		if err != nil {
			log.Errorf("GetiDisks error %s", err)
		} else {
			diskInfo := make([]SDiskInfo, len(idisks))
			for i := 0; i < len(idisks); i += 1 {
				dinfo := SDiskInfo{}
				dinfo.Uuid = idisks[i].GetGlobalId()
				dinfo.Size = idisks[i].GetDiskSizeMB()
				diskInfo[i] = dinfo
			}
			data.Add(jsonutils.Marshal(&diskInfo), "disks")
		}

		data.Add(jsonutils.NewString(iVM.GetGlobalId()), "uuid")

		return data, nil
	})
	return nil
}

func (self *SAliyunGuestDriver) OnGuestDeployTaskDataReceived(ctx context.Context, guest *models.SGuest, task taskman.ITask, data jsonutils.JSONObject) error {

	if data.Contains("disks") {
		diskInfo := make([]SDiskInfo, 0)
		err := data.Unmarshal(&diskInfo, "disks")
		if err != nil {
			return err
		}
		disks := guest.GetDisks()
		if len(disks) != len(diskInfo) {
			msg := fmt.Sprintf("inconsistent disk number: have %d want %d", len(disks), len(diskInfo))
			log.Errorf(msg)
			return fmt.Errorf(msg)
		}
		for i := 0; i < len(diskInfo); i += 1 {
			disk := disks[i].GetDisk()
			_, err = disk.GetModelManager().TableSpec().Update(disk, func() error {
				disk.DiskSize = diskInfo[i].Size
				disk.ExternalId = diskInfo[i].Uuid
				disk.Status = models.DISK_READY
				return nil
			})
			if err != nil {
				msg := fmt.Sprintf("save disk info failed %s", err)
				log.Errorf(msg)
				break
			} else {
				db.OpsLog.LogEvent(disk, db.ACT_ALLOCATE, disk.GetShortDesc(), task.GetUserCred())
			}
		}
	}
	uuid, _ := data.GetString("uuid")
	if len(uuid) > 0 {
		guest.SetExternalId(uuid)
	}
	guest.SaveDeployInfo(ctx, task.GetUserCred(), data)
	return nil
}

func (self *SAliyunGuestDriver) GetGuestVncInfo(userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost) (*jsonutils.JSONDict, error) {
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
