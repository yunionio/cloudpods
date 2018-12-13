package guestdrivers

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type SQcloudGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SQcloudGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SQcloudGuestDriver) GetHypervisor() string {
	return models.HYPERVISOR_QCLOUD
}

func (self *SQcloudGuestDriver) ChooseHostStorage(host *models.SHost, backend string) *models.SStorage {
	storages := host.GetAttachedStorages("")
	for i := 0; i < len(storages); i++ {
		if storages[i].StorageType == backend {
			return &storages[i]
		}
	}
	for _, stype := range []string{"cloud_basic", "cloud_premium", "cloud_ssd", "local_basic", "local_ssd"} {
		for i := 0; i < len(storages); i++ {
			if storages[i].StorageType == stype {
				return &storages[i]
			}
		}
	}
	return nil
}

func (self *SQcloudGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SQcloudGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SQcloudGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SQcloudGuestDriver) GetChangeConfigStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SQcloudGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SQcloudGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	//https://cloud.tencent.com/document/product/362/5747
	if !utils.IsInStringArray(guest.Status, []string{models.VM_READY, models.VM_RUNNING}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	if disk.DiskType == models.DISK_TYPE_SYS {
		return fmt.Errorf("Cannot resize system disk")
	}
	if utils.IsInStringArray(storage.StorageType, []string{models.STORAGE_LOCAL_BASIC, models.STORAGE_LOCAL_SSD}) {
		return fmt.Errorf("Cannot resize %s disk", storage.StorageType)
	}
	return nil
}

func (self *SQcloudGuestDriver) RequestDetachDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

func (self *SQcloudGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, data)
	if err != nil {
		return nil, err
	}
	if data.Contains("net.0") && data.Contains("net.1") {
		return nil, httperrors.NewInputParameterError("cannot support more than 1 nic")
	}
	sysDisk := models.SDiskConfig{}
	if err := data.Unmarshal(&sysDisk, "disk.0"); err != nil {
		return nil, fmt.Errorf("Missing system disk information")
	}
	switch sysDisk.Backend {
	case models.STORAGE_CLOUD_BASIC, models.STORAGE_CLOUD_SSD:
		if sysDisk.SizeMb < 50*10024 || sysDisk.SizeMb > 500*1024 {
			return nil, fmt.Errorf("The %s system disk size must be in the range of 50 ~ 500GB", sysDisk.Backend)
		}
	case models.STORAGE_CLOUD_PREMIUM:
		if sysDisk.SizeMb < 50*1024 || sysDisk.SizeMb > 1024*1024 {
			return nil, fmt.Errorf("The %s system disk size must be in the range of 50 ~ 1024GB", sysDisk.Backend)
		}
	default:
		if sysDisk.SizeMb < 50*1024 {
			return nil, fmt.Errorf("The system disk must be greater than or equal to 50GB")
		}
	}
	return data, nil
}

func (self *SQcloudGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	config := guest.GetDeployConfigOnHost(ctx, host, task.GetParams())
	log.Debugf("RequestDeployGuestOnHost: %s", config)
	/* onfinish, err := config.GetString("on_finish")
	if err != nil {
		return err
	} */

	action, err := config.GetString("action")
	if err != nil {
		return err
	}

	publicKey, _ := config.GetString("public_key")

	adminPublicKey, _ := config.GetString("admin_public_key")
	projectPublicKey, _ := config.GetString("project_public_key")
	oUserData, _ := config.GetString("user_data")

	userData := generateUserData(adminPublicKey, projectPublicKey, oUserData)

	resetPassword := jsonutils.QueryBoolean(config, "reset_password", false)
	passwd, _ := config.GetString("password")
	if resetPassword && len(passwd) == 0 {
		passwd = seclib2.RandomPassword2(12)
	}

	ihost, err := host.GetIHost()
	if err != nil {
		return err
	}

	desc := SManagedVMCreateConfig{}
	err = config.Unmarshal(&desc, "desc")
	if err != nil {
		return err
	}

	if action == "create" {
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {

			nets := guest.GetNetworks()
			net := nets[0].GetNetwork()
			vpc := net.GetVpc()

			iregion, err := host.GetIRegion()
			if err != nil {
				return nil, err
			}

			secgroupCache := models.SecurityGroupCacheManager.Register(ctx, task.GetUserCred(), desc.SecGroupId, "normal", vpc.CloudregionId, vpc.ManagerId)
			if secgroupCache == nil {
				return nil, fmt.Errorf("failed to registor secgroupCache for secgroup: %s, vpc: %s", desc.SecGroupId, vpc.Name)
			}

			secgroupExtId, err := iregion.SyncSecurityGroup(secgroupCache.ExternalId, vpc.ExternalId, desc.SecGroupName, "", desc.SecRules)
			if err != nil {
				log.Errorf("SyncSecurityGroup fail %s", err)
				return nil, err
			}
			if err := secgroupCache.SetExternalId(secgroupExtId); err != nil {
				return nil, fmt.Errorf("failed to set externalId for secgroup %s externalId %s: error: %v", desc.SecGroupId, secgroupExtId, err)
			}

			var createErr error
			var iVM cloudprovider.ICloudVM
			var bc *billing.SBillingCycle
			if desc.BillingCycle.IsValid() {
				bc = &desc.BillingCycle
			}
			if len(desc.InstanceType) > 0 {
				iVM, createErr = ihost.CreateVM2(desc.Name, desc.ExternalImageId, desc.SysDiskSize, desc.InstanceType, desc.ExternalNetworkId,
					desc.IpAddr, desc.Description, passwd, desc.StorageType, desc.DataDisks, publicKey, secgroupExtId, userData, bc)
			} else {
				iVM, createErr = ihost.CreateVM(desc.Name, desc.ExternalImageId, desc.SysDiskSize, desc.Cpu, desc.Memory, desc.ExternalNetworkId,
					desc.IpAddr, desc.Description, passwd, desc.StorageType, desc.DataDisks, publicKey, secgroupExtId, userData, bc)
			}

			if createErr != nil {
				return nil, createErr
			}

			log.Debugf("VMcreated %s, wait status running ...", iVM.GetGlobalId())
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
			data := fetchIVMinfo(desc, iVM, guest.Id, "root", passwd, action)
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

		name, _ := params.GetString("name")
		description, _ := params.GetString("description")
		publicKey, _ := config.GetString("public_key")
		deleteKeypair := jsonutils.QueryBoolean(params, "__delete_keypair__", false)
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {

			//腾讯云暂不支持更新自定义用户数据
			// if len(userData) > 0 {
			// 	err := iVM.UpdateUserData(userData)
			// 	if err != nil {
			// 		log.Errorf("update userdata fail %s", err)
			// 	}
			// }

			err := iVM.DeployVM(ctx, name, passwd, publicKey, deleteKeypair, description)
			if err != nil {
				return nil, err
			}

			data := fetchIVMinfo(desc, iVM, guest.Id, "root", passwd, action)
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

			diskId, err := iVM.RebuildRoot(ctx, desc.ExternalImageId, passwd, publicKey, desc.SysDiskSize)
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

			data := fetchIVMinfo(desc, iVM, guest.Id, "root", passwd, action)

			return data, nil
		})

	} else {
		log.Errorf("RequestDeployGuestOnHost: Action %s not supported", action)
		return fmt.Errorf("Action %s not supported", action)
	}

	return nil
}

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
				if err = secgroupCache.SetExternalId(extID); err != nil {
					return nil, err
				}
				externalIds = append(externalIds, extID)
			}
			return nil, iVM.AssignSecurityGroups(externalIds)
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
