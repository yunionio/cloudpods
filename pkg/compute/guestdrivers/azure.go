package guestdrivers

import (
	"context"
	"fmt"
	"strings"
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
	"yunion.io/x/onecloud/pkg/util/ansible"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/seclib2"
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
	storages := host.GetAttachedStorages("")
	for i := 0; i < len(storages); i += 1 {
		if storages[i].StorageType == backend {
			return &storages[i]
		}
	}
	for _, stype := range []string{"standard_lrs", "standardssd_lrs", "premium_lrs"} {
		for i := 0; i < len(storages); i += 1 {
			if storages[i].StorageType == stype {
				return &storages[i]
			}
		}
	}
	return nil
}

func (self *SAzureGuestDriver) GetMaxSecurityGroupCount() int {
	return 1
}

func (self *SAzureGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SAzureGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SAzureGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SAzureGuestDriver) GetChangeConfigStatus() ([]string, error) {
	return []string{models.VM_READY, models.VM_RUNNING}, nil
}

func (self *SAzureGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{models.VM_RUNNING}, nil
}

func (self *SAzureGuestDriver) IsNeedRestartForResetLoginInfo() bool {
	return false
}

func (self *SAzureGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	//https://docs.microsoft.com/en-us/rest/api/compute/disks/update
	//Resizes are only allowed if the disk is not attached to a running VM, and can only increase the disk's size
	if !utils.IsInStringArray(guest.Status, []string{models.VM_READY}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	return nil
}

func (self *SAzureGuestDriver) RequestDetachDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return guest.StartSyncTask(ctx, task.GetUserCred(), false, task.GetTaskId())
}

func (self *SAzureGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, data)
	if err != nil {
		return nil, err
	}
	if data.Contains("net.0") && data.Contains("net.1") {
		return nil, httperrors.NewInputParameterError("cannot support more than 1 nic")
	}
	return data, nil
}

func (self *SAzureGuestDriver) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, data)
	if err != nil {
		return nil, err
	}
	if data.Contains("name") {
		return nil, httperrors.NewInputParameterError("cannot support change azure instance name")
	}
	return data, nil
}

func (self *SAzureGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	config, err := guest.GetDeployConfigOnHost(ctx, task.GetUserCred(), host, task.GetParams())
	if err != nil {
		log.Errorf("GetDeployConfigOnHost error: %v", err)
		return err
	}
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
			if len(desc.Password) == 0 {
				//Azure创建必须要设置密码
				desc.Password = seclib2.RandomPassword2(12)
			}

			iVM, createErr := ihost.CreateVM(&desc)

			if createErr != nil {
				return nil, createErr
			}

			log.Debugf("VMcreated %s, wait status running ...", iVM.GetGlobalId())
			if err = cloudprovider.WaitStatus(iVM, models.VM_RUNNING, time.Second*5, time.Second*1800); err != nil {
				return nil, err
			}
			if iVM, err = ihost.GetIVMById(iVM.GetGlobalId()); err != nil {
				log.Errorf("cannot find vm %s", err)
				return nil, err
			}

			data := fetchIVMinfo(desc, iVM, guest.Id, ansible.PUBLIC_CLOUD_ANSIBLE_USER, desc.Password, action)
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
			err := iVM.DeployVM(ctx, desc.Name, desc.Password, desc.PublicKey, deleteKeypair, desc.Description)
			if err != nil {
				return nil, err
			}
			data := fetchIVMinfo(desc, iVM, guest.Id, ansible.PUBLIC_CLOUD_ANSIBLE_USER, desc.Password, action)
			return data, nil
		})
	} else if action == "rebuild" {
		iVM, err := ihost.GetIVMById(guest.GetExternalId())
		if err != nil || iVM == nil {
			log.Errorf("cannot find vm %s", err)
			return fmt.Errorf("cannot find vm")
		}

		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			_, err := iVM.RebuildRoot(ctx, desc.ExternalImageId, desc.Password, desc.PublicKey, desc.SysDisk.SizeGB)
			if err != nil {
				return nil, err
			}

			log.Debugf("VMrebuildRoot %s, and status is ready", iVM.GetGlobalId())
			data := fetchIVMinfo(desc, iVM, guest.Id, ansible.PUBLIC_CLOUD_ANSIBLE_USER, desc.Password, action)

			return data, nil
		})

	} else {
		return fmt.Errorf("Action %s not supported", action)
	}
	return nil
}

func (self *SAzureGuestDriver) RequestSyncConfigOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
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
			vpcID := "normal"
			if strings.HasSuffix(host.Name, "-classic") {
				vpcID = "classic"
			}
			iregion, err := host.GetIRegion()
			if err != nil {
				return nil, err
			}

			lockman.LockRawObject(ctx, "secgroupcache", fmt.Sprintf("%s-%s", guest.SecgrpId, vpcID))
			defer lockman.ReleaseRawObject(ctx, "secgroupcache", fmt.Sprintf("%s-%s", guest.SecgrpId, vpcID))

			secgroupCache := models.SecurityGroupCacheManager.Register(ctx, task.GetUserCred(), guest.SecgrpId, vpcID, host.GetRegion().Id, host.ManagerId)
			if secgroupCache == nil {
				return nil, fmt.Errorf("failed to registor secgroupCache for secgroup: %s vpc: %s", guest.SecgrpId, vpcID)
			}
			extID, err := iregion.SyncSecurityGroup(secgroupCache.ExternalId, vpcID, guest.GetSecgroupName(), "", guest.GetSecRules())
			if err != nil {
				return nil, err
			}
			if err = secgroupCache.SetExternalId(extID); err != nil {
				return nil, err
			}
			return nil, iVM.AssignSecurityGroup(extID)
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

func (self *SAzureGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	return false
}

func (self *SAzureGuestDriver) NeedStopForChangeSpec() bool {
	return false
}
