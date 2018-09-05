package guestdrivers

import (
	"context"
	"fmt"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SAzureGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SAzureGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SAzureGuestDriver) GetHypervisor() string {
	return models.HYPERVISOR_AZURE
}

func (self *SAzureGuestDriver) ChooseHostStorage(host *models.SHost, backend string) *models.SStorage {
	storages := host.GetAttachedStorages()
	for i := 0; i < len(storages); i += 1 {
		if storages[i].StorageType == backend {
			return &storages[i]
		}
	}
	for _, stype := range []string{"standard_lrs", "premium_lrs"} {
		for i := 0; i < len(storages); i += 1 {
			if storages[i].StorageType == stype {
				return &storages[i]
			}
		}
	}
	return nil
}

func (self *SAzureGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SAzureGuestDriver) RequestDetachDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

func (self *SAzureGuestDriver) RequestSyncConfigOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	return fmt.Errorf("Not Implement")
}

type SAzureVMCreateConfig struct {
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
}

func (self *SAzureGuestDriver) GetJsonDescAtHost(ctx context.Context, guest *models.SGuest, host *models.SHost) jsonutils.JSONObject {
	config := SAzureVMCreateConfig{}
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
			log.Debugf("disk: %v", disk)
			storage := disk.GetStorage()
			log.Debugf("disk storage: %v", storage)
			config.StorageType = storage.StorageType
			cache := storage.GetStoragecache()
			imageId := disk.GetTemplateId()
			scimg := models.StoragecachedimageManager.GetStoragecachedimage(cache.Id, imageId)
			config.ExternalImageId = scimg.ExternalId
			img := scimg.GetCachedimage()
			config.OsDistribution, _ = img.Info.GetString("properties", "os_distribution")
			config.OsVersion, _ = img.Info.GetString("properties", "os_version")

			config.SysDiskSize = disk.DiskSize / 1024 // MB => GB
		} else {
			config.DataDisks[i-1] = disk.DiskSize / 1024 // MB => GB
		}
	}
	return jsonutils.Marshal(&config)
}

func (self *SAzureGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	config := guest.GetDeployConfigOnHost(ctx, host, task.GetParams())

	if action, err := config.GetString("action"); err != nil {
		return err
	} else if ihost, err := host.GetIHost(); err != nil {
		return err
	} else if action == "create" {
		desc := SAzureVMCreateConfig{}
		if err := config.Unmarshal(&desc, "desc"); err != nil {
			return err
		}

		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			passwd := seclib2.RandomPassword2(12)

			if iVM, err := ihost.CreateVM(desc.Name, desc.ExternalImageId, desc.SysDiskSize, desc.Cpu, desc.Memory, desc.ExternalNetworkId,
				desc.IpAddr, desc.Description, passwd, desc.StorageType, desc.DataDisks, desc.PublicKey); err != nil {
				return nil, err
			} else {
				log.Debugf("VMcreated %s, wait status ready ...", iVM.GetGlobalId())

				if iVM, err = ihost.GetIVMById(iVM.GetGlobalId()); err != nil {
					log.Errorf("cannot find vm %s", err)
					return nil, err
				}

				if len(guest.SecgrpId) > 0 {
					if err := iVM.SyncSecurityGroup(guest.SecgrpId, guest.GetSecgroupName(), guest.GetSecRules()); err != nil {
						log.Errorf("SyncSecurityGroup error: %v", err)
						return nil, err
					}
				}

				data := jsonutils.NewDict()

				if encpasswd, err := utils.EncryptAESBase64(guest.Id, passwd); err != nil {
					log.Errorf("encrypt password failed %s", err)
				} else {
					data.Add(jsonutils.NewString(iVM.GetOSType()), "os")
					data.Add(jsonutils.NewString("root"), "account")
					data.Add(jsonutils.NewString(encpasswd), "key")

					if len(desc.OsDistribution) > 0 {
						data.Add(jsonutils.NewString(desc.OsDistribution), "distro")
					}
					if len(desc.OsVersion) > 0 {
						data.Add(jsonutils.NewString(desc.OsVersion), "version")
					}
				}

				if idisks, err := iVM.GetIDisks(); err != nil {
					log.Errorf("GetiDisks error %s", err)
				} else {
					diskInfo := make([]SDiskInfo, len(idisks))
					for i := 0; i < len(idisks); i++ {
						dinfo := SDiskInfo{}
						dinfo.Uuid = idisks[i].GetGlobalId()
						dinfo.Size = idisks[i].GetDiskSizeMB()
						if metaData := idisks[i].GetMetadata(); metaData != nil {
							dinfo.Metadata = make(map[string]string, 0)
							if err := metaData.Unmarshal(dinfo.Metadata); err != nil {
								log.Errorf("Get disk %s metadata info error: %v", idisks[i].GetName(), err)
							}
						}
						diskInfo[i] = dinfo
					}
					data.Add(jsonutils.Marshal(&diskInfo), "disks")
				}

				data.Add(jsonutils.NewString(iVM.GetGlobalId()), "uuid")
				data.Add(iVM.GetMetadata(), "metadata")
				return data, nil
			}
		})
	} else if action == "deploy" {

	} else {
		return fmt.Errorf("Action %s not supported", action)
	}
	return nil
}
