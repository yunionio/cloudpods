package guestdrivers

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
)

type SQcloudGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SQcloudGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SQcloudGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_QCLOUD
}

func (self *SQcloudGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_QCLOUD
}

func (self *SQcloudGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_CLOUD_PREMIUM
}

func (self *SQcloudGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 50
}

func (self *SQcloudGuestDriver) GetStorageTypes() []string {
	return []string{
		api.STORAGE_CLOUD_BASIC,
		api.STORAGE_CLOUD_PREMIUM,
		api.STORAGE_CLOUD_SSD,
		api.STORAGE_LOCAL_BASIC,
		api.STORAGE_LOCAL_SSD,
	}
}

func (self *SQcloudGuestDriver) ChooseHostStorage(host *models.SHost, backend string, storageIds []string) *models.SStorage {
	return self.chooseHostStorage(self, host, backend, storageIds)
}

func (self *SQcloudGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SQcloudGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SQcloudGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SQcloudGuestDriver) GetChangeConfigStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SQcloudGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SQcloudGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	//https://cloud.tencent.com/document/product/362/5747
	if !utils.IsInStringArray(guest.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	if disk.DiskType == api.DISK_TYPE_SYS {
		return fmt.Errorf("Cannot resize system disk")
	}
	if utils.IsInStringArray(storage.StorageType, []string{api.STORAGE_LOCAL_BASIC, api.STORAGE_LOCAL_SSD}) {
		return fmt.Errorf("Cannot resize %s disk", storage.StorageType)
	}
	return nil
}

func (self *SQcloudGuestDriver) RequestDetachDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

func (self *SQcloudGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	input, err := self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, input)
	if err != nil {
		return nil, err
	}
	if len(input.Networks) > 2 {
		return nil, httperrors.NewInputParameterError("cannot support more than 1 nic")
	}

	sysDisk := input.Disks[0]
	switch sysDisk.Backend {
	case api.STORAGE_CLOUD_BASIC, api.STORAGE_CLOUD_SSD:
		if sysDisk.SizeMb > 500*1024 {
			return nil, fmt.Errorf("The %s system disk size must be less than 500GB", sysDisk.Backend)
		}
	case api.STORAGE_CLOUD_PREMIUM:
		if sysDisk.SizeMb > 1024*1024 {
			return nil, fmt.Errorf("The %s system disk size must be less than 1024GB", sysDisk.Backend)
		}
	}

	for i := 1; i < len(input.Disks); i++ {
		disk := input.Disks[i]
		switch disk.Backend {
		case api.STORAGE_CLOUD_BASIC:
			if disk.SizeMb < 10*1024 || disk.SizeMb > 16000*1024 {
				return nil, httperrors.NewInputParameterError("The %s disk size must be in the range of 10GB ~ 16000GB", disk.Backend)
			}
		case api.STORAGE_CLOUD_PREMIUM:
			if disk.SizeMb < 50*1024 || disk.SizeMb > 16000*1024 {
				return nil, httperrors.NewInputParameterError("The %s disk size must be in the range of 50GB ~ 16000GB", disk.Backend)
			}
		case api.STORAGE_CLOUD_SSD:
			if disk.SizeMb < 100*1024 || disk.SizeMb > 16000*1024 {
				return nil, httperrors.NewInputParameterError("The %s disk size must be in the range of 100GB ~ 16000GB", disk.Backend)
			}
		}
	}
	return input, nil
}

func (self *SQcloudGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SQcloudGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_RUNNING
}

func (self *SQcloudGuestDriver) GetUserDataType() string {
	return cloudprovider.CLOUD_SHELL
}

func (self *SQcloudGuestDriver) GetLinuxDefaultAccount(desc cloudprovider.SManagedVMCreateConfig) string {
	userName := "root"
	if desc.ImageType == "system" {
		if desc.OsDistribution == "Ubuntu" {
			userName = "ubuntu"
		}
		if desc.OsType == "Windows" {
			userName = "Administrator"
		}
	}
	return userName
}

/* func (self *SQcloudGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
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
	if desc.ImageType == "system" {
		if desc.OsDistribution == "Ubuntu" {
			userName = "ubuntu"
		}
		if desc.OsType == "Windows" {
			userName = "Administrator"
		}
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

			log.Debugf("VMcreated %s, wait status running ...", iVM.GetGlobalId())
			err = cloudprovider.WaitStatus(iVM, models.VM_RUNNING, time.Second*5, time.Second*1800)
			if err != nil {
				return nil, err
			}
			log.Debugf("VMcreated %s, and status is running", iVM.GetGlobalId())

			iVM, err = ihost.GetIVMById(iVM.GetGlobalId())
			if err != nil {
				log.Errorf("cannot find vm %s", err)
				return nil, err
			}

			err = cloudprovider.RetryUntil(func() (bool, error) {
				idisks, err := iVM.GetIDisks()
				if err != nil {
					log.Errorf("cannot find vm disks %s", err)
					return false, err
				}
				if len(idisks) == len(desc.DataDisks)+1 {
					return true, nil
				} else {
					return false, nil
				}
			}, 10)
			if err != nil {
				return nil, err
			}

			data := fetchIVMinfo(desc, iVM, guest.Id, userName, desc.Password, action)
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

			//腾讯云暂不支持更新自定义用户数据
			// if len(userData) > 0 {
			// 	err := iVM.UpdateUserData(userData)
			// 	if err != nil {
			// 		log.Errorf("update userdata fail %s", err)
			// 	}
			// }

			err := iVM.DeployVM(ctx, desc.Name, desc.Password, desc.PublicKey, deleteKeypair, desc.Description)
			if err != nil {
				return nil, err
			}

			data := fetchIVMinfo(desc, iVM, guest.Id, userName, desc.Password, action)
			return data, nil
		})
	} else if action == "rebuild" {

		iVM, err := ihost.GetIVMById(guest.GetExternalId())
		if err != nil || iVM == nil {
			log.Errorf("cannot find vm %s", err)
			return fmt.Errorf("cannot find vm")
		}

		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			//腾讯云暂不支持更新自定义用户数据
			// if len(userData) > 0 {
			// 	err := iVM.UpdateUserData(userData)
			// 	if err != nil {
			// 		log.Errorf("update userdata fail %s", err)
			// 	}
			// }

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
				if len(idisks) < len(desc.DataDisks)+1 {
					if waited > maxWaitSecs {
						log.Errorf("inconsistent disk number, wait timeout, must be something wrong on remote")
						return nil, cloudprovider.ErrTimeout
					}
					log.Debugf("inconsistent disk number???? %d != %d", len(idisks), len(desc.DataDisks)+1)
					time.Sleep(time.Second * 5)
					waited += 5
				} else {
					if idisks[0].GetGlobalId() != diskId {
						log.Errorf("system disk id inconsistent %s != %s", idisks[0].GetGlobalId(), diskId)
						return nil, fmt.Errorf("inconsistent sys disk id after rebuild root")
					}

					break
				}
			}

			data := fetchIVMinfo(desc, iVM, guest.Id, userName, desc.Password, action)

			return data, nil
		})

	} else {
		log.Errorf("RequestDeployGuestOnHost: Action %s not supported", action)
		return fmt.Errorf("Action %s not supported", action)
	}

	return nil
} */

func (self *SQcloudGuestDriver) RequestSyncConfigOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		ihost, err := host.GetIHost()
		if err != nil {
			return nil, err
		}
		iVM, err := ihost.GetIVMById(guest.ExternalId)
		if err != nil {
			return nil, err
		}

		if fwOnly, _ := task.GetParams().Bool("fw_only"); fwOnly {
			iregion, err := host.GetIRegion()
			if err != nil {
				return nil, err
			}
			secgroups := guest.GetSecgroups()
			externalIds := []string{}
			for _, secgroup := range secgroups {

				lockman.LockRawObject(ctx, "secgroupcache", fmt.Sprintf("%s-normal", guest.SecgrpId))
				defer lockman.ReleaseRawObject(ctx, "secgroupcache", fmt.Sprintf("%s-normal", guest.SecgrpId))

				secgroupCache := models.SecurityGroupCacheManager.Register(ctx, task.GetUserCred(), secgroup.Id, "normal", host.GetRegion().Id, host.ManagerId)
				if secgroupCache == nil {
					return nil, fmt.Errorf("failed to registor secgroupCache for secgroup: %s", secgroup.Id)
				}
				extID, err := iregion.SyncSecurityGroup(secgroupCache.ExternalId, "", secgroup.Name, secgroup.Description, secgroup.GetSecRules(""))
				if err != nil {
					return nil, err
				}
				if err = secgroupCache.SetExternalId(task.GetUserCred(), extID); err != nil {
					return nil, err
				}
				externalIds = append(externalIds, extID)
			}
			return nil, iVM.SetSecurityGroups(externalIds)
		}

		iDisks, err := iVM.GetIDisks()
		if err != nil {
			return nil, err
		}
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
			if err := iVM.DetachDisk(ctx, disk.GetId()); err != nil {
				return nil, err
			}
		}
		for _, disk := range added {
			if err := iVM.AttachDisk(ctx, disk.ExternalId); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	return nil
}

func (self *SQcloudGuestDriver) AllowReconfigGuest() bool {
	return true
}

func (self *SQcloudGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	months := bc.GetMonths()
	if (months >= 1 && months <= 12) || (months == 24) || (months == 36) {
		return true
	}
	return false
}
