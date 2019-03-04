package guestdrivers

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
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

func (self *SAliyunGuestDriver) GetDefaultSysDiskBackend() string {
	return models.STORAGE_CLOUD_EFFICIENCY
}

func (self *SAliyunGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 20
}

func (self *SAliyunGuestDriver) ChooseHostStorage(host *models.SHost, backend string) *models.SStorage {
	storages := host.GetAttachedStorages("")
	for i := 0; i < len(storages); i += 1 {
		if storages[i].StorageType == backend {
			return &storages[i]
		}
	}
	for _, stype := range []string{
		models.STORAGE_CLOUD_EFFICIENCY,
		models.STORAGE_CLOUD_SSD,
		models.STORAGE_CLOUD_ESSD,
		models.STORAGE_PUBLIC_CLOUD,
		models.STORAGE_EPHEMERAL_SSD,
	} {
		for i := 0; i < len(storages); i += 1 {
			if storages[i].StorageType == stype {
				return &storages[i]
			}
		}
	}
	return nil
}

func (self *SAliyunGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SAliyunGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SAliyunGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SAliyunGuestDriver) GetChangeConfigStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SAliyunGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SAliyunGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{models.VM_READY, models.VM_RUNNING}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	if disk.DiskType == models.DISK_TYPE_SYS {
		return fmt.Errorf("Cannot resize system disk")
	}
	if !utils.IsInStringArray(storage.StorageType, []string{models.STORAGE_PUBLIC_CLOUD, models.STORAGE_CLOUD_SSD, models.STORAGE_CLOUD_EFFICIENCY}) {
		return fmt.Errorf("Cannot resize %s disk", storage.StorageType)
	}
	return nil
}

func (self *SAliyunGuestDriver) RequestDetachDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

func (self *SAliyunGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, data)
	if err != nil {
		return nil, err
	}
	if data.Contains("net.0") && data.Contains("net.1") {
		return nil, httperrors.NewInputParameterError("cannot support more than 1 nic")
	}
	for i := 0; data.Contains(fmt.Sprintf("disk.%d", i)); i++ {
		disk := models.SDiskConfig{}
		if err := data.Unmarshal(&disk, fmt.Sprintf("disk.%d", i)); err != nil {
			return nil, httperrors.NewInputParameterError("invalid diskinfo of index %d", i)
		}
		if i == 0 && (disk.SizeMb < 20*1024 || disk.SizeMb > 500*1024) {
			return nil, httperrors.NewInputParameterError("The system disk size must be in the range of 20GB ~ 500Gb")
		}
		switch disk.Backend {
		case models.STORAGE_CLOUD_EFFICIENCY, models.STORAGE_CLOUD_SSD, models.STORAGE_CLOUD_ESSD:
			if disk.SizeMb < 20*1024 || disk.SizeMb > 32768*1024 {
				return nil, httperrors.NewInputParameterError("The %s disk size must be in the range of 20GB ~ 32768GB", disk.Backend)
			}
		case models.STORAGE_PUBLIC_CLOUD:
			if disk.SizeMb < 5*1024 || disk.SizeMb > 2000*1024 {
				return nil, httperrors.NewInputParameterError("The %s disk size must be in the range of 5GB ~ 2000GB", disk.Backend)
			}
		case models.STORAGE_EPHEMERAL_SSD:
			if disk.SizeMb < 5*1024 || disk.SizeMb > 800*1024 {
				return nil, httperrors.NewInputParameterError("The %s disk size must be in the range of 5GB ~ 800GB", disk.Backend)
			}
		}
	}
	return data, nil
}

func (self *SAliyunGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	config, err := guest.GetDeployConfigOnHost(ctx, task.GetUserCred(), host, task.GetParams())
	if err != nil {
		log.Errorf("GetDeployConfigOnHost error: %v", err)
		return err
	}
	log.Debugf("RequestDeployGuestOnHost: %s", config)

	desc := cloudprovider.SManagedVMCreateConfig{}
	if err := desc.GetConfig(config); err != nil {
		return err
	}

	userName := "root"
	if desc.ImageType == "system" && desc.OsType == "Windows" {
		userName = "Administrator"
	}

	action, err := config.GetString("action")
	if err != nil {
		return err
	}

	ihost, err := host.GetIHost()
	if err != nil {
		return err
	}

	if action == "create" {
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {

			iVM, createErr := ihost.CreateVM(&desc)
			if createErr != nil {
				return nil, createErr
			}
			guest.SetExternalId(iVM.GetGlobalId())

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

			data := fetchIVMinfo(desc, iVM, guest.Id, "root", desc.Password, action)

			return data, nil
		})
	} else if action == "deploy" {
		iVM, err := ihost.GetIVMById(guest.GetExternalId())
		if err != nil || iVM == nil {
			log.Errorf("cannot find vm %s", err)
			return fmt.Errorf("cannot find vm")
		}

		params := task.GetParams()
		log.Debugf("Deploy VM params %s", params.String())

		deleteKeypair := jsonutils.QueryBoolean(params, "__delete_keypair__", false)

		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {

			if len(desc.UserData) > 0 {
				err := iVM.UpdateUserData(desc.UserData)
				if err != nil {
					log.Errorf("update userdata fail %s", err)
				}
			}

			err := iVM.DeployVM(ctx, desc.Name, desc.Password, desc.PublicKey, deleteKeypair, desc.Description)
			if err != nil {
				return nil, err
			}

			data := fetchIVMinfo(desc, iVM, guest.Id, "root", desc.Password, action)

			return data, nil
		})
	} else if action == "rebuild" {

		iVM, err := ihost.GetIVMById(guest.GetExternalId())
		if err != nil || iVM == nil {
			log.Errorf("cannot find vm %s", err)
			return fmt.Errorf("cannot find vm")
		}

		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			if len(desc.UserData) > 0 {
				err := iVM.UpdateUserData(desc.UserData)
				if err != nil {
					log.Errorf("update userdata fail %s", err)
				}
			}

			diskId, err := iVM.RebuildRoot(ctx, desc.ExternalImageId, desc.Password, desc.PublicKey, desc.SysDisk.SizeGB)
			if err != nil {
				return nil, err
			}

			log.Debugf("VMrebuildRoot %s new diskID %s, wait status ready ...", iVM.GetGlobalId(), diskId)

			err = cloudprovider.WaitStatus(iVM, models.VM_READY, time.Second*5, time.Second*1800)
			if err != nil {
				return nil, err
			}
			log.Debugf("VMrebuildRoot %s, and status is ready", iVM.GetGlobalId())

			maxWaitSecs := 300
			waited := 0

			for {
				// hack, wait disk number consistent
				idisks, err := iVM.GetIDisks()
				if err != nil {
					log.Errorf("fail to find VM idisks %s", err)
					return nil, err
				}
				if len(idisks) < len(desc.DataDisks)+1 || idisks[0].GetGlobalId() != diskId {
					if waited > maxWaitSecs {
						log.Errorf("inconsistent disk number, wait timeout, must be something wrong on remote")
						return nil, cloudprovider.ErrTimeout
					}
					if len(idisks) < len(desc.DataDisks)+1 {
						log.Debugf("inconsistent disk number???? %d != %d", len(idisks), len(desc.DataDisks)+1)
					}
					if len(idisks) > 0 && idisks[0].GetGlobalId() != diskId {
						log.Errorf("system disk id inconsistent %s != %s", idisks[0].GetGlobalId(), diskId)
					}
					time.Sleep(time.Second * 5)
					waited += 5
				} else {
					break
				}
			}

			data := fetchIVMinfo(desc, iVM, guest.Id, "root", desc.Password, action)

			return data, nil
		})

	} else {
		log.Errorf("RequestDeployGuestOnHost: Action %s not supported", action)
		return fmt.Errorf("Action %s not supported", action)
	}

	return nil
}

func (self *SAliyunGuestDriver) AllowReconfigGuest() bool {
	return true
}

func (self *SAliyunGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	weeks := bc.GetWeeks()
	if weeks >= 1 && weeks <= 4 {
		return true
	}
	months := bc.GetMonths()
	if (months >= 1 && months <= 10) || (months == 12) || (months == 24) || (months == 36) || (months == 48) || (months == 60) {
		return true
	}
	return false
}
