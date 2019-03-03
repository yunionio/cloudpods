package guestdrivers

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
)

type SHuaweiGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SHuaweiGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SHuaweiGuestDriver) GetHypervisor() string {
	return models.HYPERVISOR_HUAWEI
}

func (self *SHuaweiGuestDriver) GetDefaultSysDiskBackend() string {
	return models.STORAGE_HUAWEI_SATA
}

func (self *SHuaweiGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 10
}

func (self *SHuaweiGuestDriver) ChooseHostStorage(host *models.SHost, backend string) *models.SStorage {
	storages := host.GetAttachedStorages("")
	for i := 0; i < len(storages); i += 1 {
		if storages[i].StorageType == backend {
			return &storages[i]
		}
	}

	for _, stype := range []string{models.STORAGE_HUAWEI_SATA, models.STORAGE_HUAWEI_SAS, models.STORAGE_HUAWEI_SSD} {
		for i := 0; i < len(storages); i += 1 {
			if storages[i].StorageType == stype {
				return &storages[i]
			}
		}
	}
	return nil
}

func (self *SHuaweiGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SHuaweiGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SHuaweiGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SHuaweiGuestDriver) GetChangeConfigStatus() ([]string, error) {
	return []string{models.VM_READY}, nil
}

func (self *SHuaweiGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SHuaweiGuestDriver) RequestDetachDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

func (self *SHuaweiGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, data)
}

func (self *SHuaweiGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{models.VM_RUNNING, models.VM_READY}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	if !utils.IsInStringArray(storage.StorageType, []string{models.STORAGE_HUAWEI_SATA, models.STORAGE_HUAWEI_SAS, models.STORAGE_HUAWEI_SSD}) {
		return fmt.Errorf("Cannot resize disk with unsupported volumes type %s", storage.StorageType)
	}

	return nil
}

func (self *SHuaweiGuestDriver) GetGuestInitialStateAfterCreate() string {
	return models.VM_RUNNING
}

func (self *SHuaweiGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return models.VM_RUNNING
}

/*
func (self *SHuaweiGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
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
			guest.SetExternalId(task.GetUserCred(), iVM.GetGlobalId())

			log.Debugf("VMcreated %s, wait status ready ...", iVM.GetGlobalId())
			err = cloudprovider.WaitStatus(iVM, models.VM_RUNNING, time.Second*5, time.Second*1800)
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

			err = cloudprovider.WaitStatus(iVM, models.VM_RUNNING, time.Second*5, time.Second*1800)
			if err != nil {
				return nil, err
			}
			log.Debugf("VMrebuildRoot %s, and status is %s", iVM.GetGlobalId(), iVM.GetStatus())

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
}*/

func (self *SHuaweiGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	months := bc.GetMonths()
	if (months >= 1 && months <= 9) || (months == 12) || (months == 24) || (months == 36) {
		return true
	}

	return false
}
