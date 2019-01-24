package guestdrivers

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/pkg/utils"
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

func (self *SHuaweiGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	config := guest.GetDeployConfigOnHost(ctx, host, task.GetParams())
	log.Debugf("RequestDeployGuestOnHost: %s", config)

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

			secgroupCache := models.SecurityGroupCacheManager.Register(ctx, task.GetUserCred(), desc.SecGroupId, vpc.Id, vpc.CloudregionId, vpc.ManagerId)
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

			if len(userData) > 0 {
				err := iVM.UpdateUserData(userData)
				if err != nil {
					log.Errorf("update userdata fail %s", err)
				}
			}

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
			if len(userData) > 0 {
				err := iVM.UpdateUserData(userData)
				if err != nil {
					log.Errorf("update userdata fail %s", err)
				}
			}

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

			data := fetchIVMinfo(desc, iVM, guest.Id, "root", passwd, action)

			return data, nil
		})

	} else {
		log.Errorf("RequestDeployGuestOnHost: Action %s not supported", action)
		return fmt.Errorf("Action %s not supported", action)
	}

	return nil
}

func (self *SHuaweiGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	months := bc.GetMonths()
	if (months >= 1 && months <= 9) || (months == 12) || (months == 24) || (months == 36) {
		return true
	}

	return false
}
