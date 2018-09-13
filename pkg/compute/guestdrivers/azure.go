package guestdrivers

import (
	"context"
	"fmt"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SAzureGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

const (
	DEFAULT_USER = "yunion"
)

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

func (self *SAzureGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, data)
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
			storage := disk.GetStorage()
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
				log.Debugf("VMcreated %s, wait status running ...", iVM.GetGlobalId())

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
					data.Add(jsonutils.NewString(DEFAULT_USER), "account")
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
		iVM, err := ihost.GetIVMById(guest.GetExternalId())
		if err != nil || iVM == nil {
			log.Errorf("cannot find vm %s", err)
			return fmt.Errorf("cannot find vm")
		}
		params := task.GetParams()
		log.Debugf("Deploy VM params %s", params.String())

		name, _ := params.GetString("name")
		description, _ := params.GetString("description")
		publicKey, _ := config.GetString("public_key")
		resetPassword := jsonutils.QueryBoolean(params, "reset_password", false)
		deleteKeypair := jsonutils.QueryBoolean(params, "__delete_keypair__", false)
		password, _ := params.GetString("password")
		if resetPassword && len(password) == 0 {
			password = seclib2.RandomPassword2(12)
		}

		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			encpasswd, err := utils.EncryptAESBase64(guest.Id, password)
			if err != nil {
				log.Errorf("encrypt password failed %s", err)
			}

			data := jsonutils.NewDict()
			data.Add(jsonutils.NewString(DEFAULT_USER), "account") // 用户名
			data.Add(jsonutils.NewString(encpasswd), "key")        // 密码
			e := iVM.DeployVM(name, password, publicKey, resetPassword, deleteKeypair, description)
			return data, e
		})
	} else {
		return fmt.Errorf("Action %s not supported", action)
	}
	return nil
}

func (self *SAzureGuestDriver) OnGuestDeployTaskDataReceived(ctx context.Context, guest *models.SGuest, task taskman.ITask, data jsonutils.JSONObject) error {

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
		for i := 0; i < len(diskInfo); i++ {
			disk := disks[i].GetDisk()
			_, err = disk.GetModelManager().TableSpec().Update(disk, func() error {
				disk.DiskSize = diskInfo[i].Size
				disk.ExternalId = diskInfo[i].Uuid
				disk.Status = models.DISK_READY
				if len(diskInfo[i].Metadata) > 0 {
					for key, value := range diskInfo[i].Metadata {
						if err := disk.SetMetadata(ctx, key, value, task.GetUserCred()); err != nil {
							log.Errorf("set disk %s mata %s => %s error: %v", disk.Name, key, value, err)
						}
					}
				}
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

	if metaData, _ := data.Get("metadata"); metaData != nil {
		meta := make(map[string]string, 0)
		if err := metaData.Unmarshal(meta); err != nil {
			log.Errorf("Get guest %s metadata error: %v", guest.Name, err)
		} else {
			for key, value := range meta {
				if err := guest.SetMetadata(ctx, key, value, task.GetUserCred()); err != nil {
					log.Errorf("set guest %s mata %s => %s error: %v", guest.Name, key, value, err)
				}
			}
		}
	}

	guest.SaveDeployInfo(ctx, task.GetUserCred(), data)
	return nil
}

func (self *SAzureGuestDriver) RequestSyncConfigOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		if ihost, err := host.GetIHost(); err != nil {
			return nil, err
		} else if iVM, err := ihost.GetIVMById(guest.ExternalId); err != nil {
			return nil, err
		} else {
			if fw_only, _ := task.GetParams().Bool("fw_only"); fw_only {
				if err := iVM.SyncSecurityGroup(guest.SecgrpId, guest.GetSecgroupName(), guest.GetSecRules()); err != nil {
					return nil, err
				}
			} else {
				if iDisks, err := iVM.GetIDisks(); err != nil {
					return nil, err
				} else {
					disks := make([]models.SDisk, 0)
					for _, guestdisk := range guest.GetDisks() {
						disk := guestdisk.GetDisk()
						disks = append(disks, *disk)
					}

					added := make([]models.SDisk, 0)
					commondb := make([]models.SDisk, 0)
					commonext := make([]cloudprovider.ICloudDisk, 0)
					removed := make([]cloudprovider.ICloudDisk, 0)

					if err := compare.CompareSets(disks, iDisks, &added, &commondb, &commonext, &removed); err != nil {
						return nil, err
					}
					for _, disk := range removed {
						if err := iVM.DetachDisk(disk.GetId()); err != nil {
							return nil, err
						}
					}
					for _, disk := range added {
						if err := iVM.AttachDisk(disk.ExternalId); err != nil {
							return nil, err
						}
					}
				}
			}
		}
		return nil, nil
	})
	return nil
}

func (self *SAzureGuestDriver) RequestRebuildRootDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	ihost, e := guest.GetHost().GetIHost()
	if e != nil {
		return e
	}

	externalId := guest.GetExternalId()
	if len(externalId) <= 0 {
		return fmt.Errorf("external id not found")
	}

	disks := guest.GetDisks()
	if len(disks) <= 0 {
		return fmt.Errorf("guest has no disk")
	}

	imageId := guest.CategorizeDisks().Root.TemplateId
	cacheId := disks[0].GetDisk().GetStorage().GetStoragecache().Id
	externalImageId := models.StoragecachedimageManager.GetStoragecachedimage(cacheId, imageId).ExternalId
	if len(externalImageId) <= 0 {
		return fmt.Errorf("external image (%s) id is not found", imageId)
	}

	iVM, err := ihost.GetIVMById(externalId)
	if err != nil {
		return err
	}

	err = iVM.RebuildRoot(externalImageId)
	if err != nil {
		return err
	}

	log.Debugf("VMrebuildRoot %s, and status is ready", iVM.GetGlobalId())

	task.ScheduleRun(nil)
	return nil
}
