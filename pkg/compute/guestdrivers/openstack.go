// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package guestdrivers

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	scheduler "yunion.io/x/onecloud/pkg/scheduler/options"
)

type SOpenStackGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SOpenStackGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SOpenStackGuestDriver) DoScheduleCPUFilter() bool {
	return scheduler.Options.OpenstackSchedulerCPUFilter
}

func (self *SOpenStackGuestDriver) DoScheduleMemoryFilter() bool {
	return scheduler.Options.OpenstackSchedulerMemoryFilter
}

func (self *SOpenStackGuestDriver) DoScheduleSKUFilter() bool {
	return scheduler.Options.OpenstackSchedulerSKUFilter
}

func (self *SOpenStackGuestDriver) DoScheduleStorageFilter() bool {
	return scheduler.Options.OpenstackSchedulerStorageFilter
}

func (self *SOpenStackGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_OPENSTACK
}

func (self *SOpenStackGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_OPENSTACK
}

func (self *SOpenStackGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
	return cloudprovider.SInstanceCapability{
		Hypervisor: self.GetHypervisor(),
		Provider:   self.GetProvider(),
		DefaultAccount: cloudprovider.SDefaultAccount{
			Linux: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_DEFAULT_LINUX_LOGIN_USER,
				Changeable:     false,
			},
			Windows: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_DEFAULT_WINDOWS_LOGIN_USER,
				Changeable:     false,
			},
		},
	}
}

func (self *SOpenStackGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PRIVATE_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_OPENSTACK
	keys.Brand = brand
	keys.Hypervisor = api.HYPERVISOR_OPENSTACK
	return keys
}

func (self *SOpenStackGuestDriver) IsSupportEip() bool {
	return false
}

func (self *SOpenStackGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_OPENSTACK_NOVA
}

func (self *SOpenStackGuestDriver) GetMinimalSysDiskSizeGb() int {
	return options.Options.DefaultDiskSizeMB / 1024
}

func (self *SOpenStackGuestDriver) GetStorageTypes() []string {
	storages, _ := models.StorageManager.GetStorageTypesByProvider(self.GetProvider())
	return storages
}

func (self *SOpenStackGuestDriver) ChooseHostStorage(host *models.SHost, guest *models.SGuest, diskConfig *api.DiskConfig, storageIds []string) (*models.SStorage, error) {
	return chooseHostStorage(self, host, diskConfig.Backend, storageIds), nil
}

func (self *SOpenStackGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SOpenStackGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SOpenStackGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING, api.VM_REBUILD_ROOT_FAIL}, nil
}

func (self *SOpenStackGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SOpenStackGuestDriver) IsNeedInjectPasswordByCloudInit() bool {
	return true
}

func (self *SOpenStackGuestDriver) IsWindowsUserDataTypeNeedEncode() bool {
	return true
}

func (self *SOpenStackGuestDriver) IsNeedRestartForResetLoginInfo() bool {
	return false
}

func (self *SOpenStackGuestDriver) IsRebuildRootSupportChangeImage() bool {
	return true
}

func (self *SOpenStackGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_RUNNING}, nil
}

func (self *SOpenStackGuestDriver) ValidateCreateEip(ctx context.Context, userCred mcclient.TokenCredential, input api.ServerCreateEipInput) error {
	return httperrors.NewInputParameterError("%s not support create eip, it only support bind eip", self.GetHypervisor())
}

func (self *SOpenStackGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_READY}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	return nil
}

func (self *SOpenStackGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	var err error
	input, err = self.SManagedVirtualizedGuestDriver.ValidateCreateData(ctx, userCred, input)
	if err != nil {
		return nil, err
	}
	if len(input.Networks) >= 2 {
		return nil, httperrors.NewInputParameterError("cannot support more than 1 nic")
	}
	if len(input.Eip) > 0 || input.EipBw > 0 {
		return nil, httperrors.NewUnsupportOperationError("%s not support create virtual machine with eip", self.GetHypervisor())
	}
	for i := 1; i < len(input.Disks); i++ {
		disk := input.Disks[i]
		if disk.Backend == api.STORAGE_OPENSTACK_NOVA {
			return nil, httperrors.NewUnsupportOperationError("data disk not support storage type %s", disk.Backend)
		}
	}

	return input, nil
}

func (self *SOpenStackGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SOpenStackGuestDriver) attachDisks(ctx context.Context, ihost cloudprovider.ICloudHost, instanceId string, diskIds []string) {
	if len(diskIds) == 0 {
		return
	}
	iVM, err := ihost.GetIVMById(instanceId)
	if err != nil || iVM == nil {
		log.Errorf("cannot find vm %s", instanceId)
		return
	}
	for _, diskId := range diskIds {
		err = iVM.AttachDisk(ctx, diskId)
		if err != nil {
			log.Errorf("failed to attach disk %s", diskId)
		}
	}
	return
}

func (self *SOpenStackGuestDriver) RemoteDeployGuestForRebuildRoot(ctx context.Context, guest *models.SGuest, ihost cloudprovider.ICloudHost, task taskman.ITask, desc cloudprovider.SManagedVMCreateConfig) (jsonutils.JSONObject, error) {
	iVM, err := ihost.GetIVMById(guest.GetExternalId())
	if err != nil || iVM == nil {
		return nil, fmt.Errorf("cannot find vm %s(%s)", guest.Id, guest.Name)
	}

	driver, err := guest.GetDriver()
	if err != nil {
		return nil, err
	}

	instanceId := iVM.GetGlobalId()

	diskId, err := func() (string, error) {
		lockman.LockObject(ctx, guest)
		defer lockman.ReleaseObject(ctx, guest)

		sysDisk, err := guest.GetSystemDisk()
		if err != nil {
			return "", errors.Wrap(err, "guest.GetSystemDisk(")
		}
		storage, err := sysDisk.GetStorage()
		if err != nil {
			return "", errors.Wrap(err, "sysDisk.GetStorage")
		}
		if storage.StorageType == api.STORAGE_OPENSTACK_NOVA { //不通过镜像创建磁盘的机器
			conf := cloudprovider.SManagedVMRebuildRootConfig{
				Account:   desc.Account,
				ImageId:   desc.ExternalImageId,
				Password:  desc.Password,
				PublicKey: desc.PublicKey,
				SysSizeGB: desc.SysDisk.SizeGB,
				OsType:    desc.OsType,
			}
			return iVM.RebuildRoot(ctx, &conf)
		}

		iDisks, err := iVM.GetIDisks()
		if err != nil {
			return "", errors.Wrap(err, "iVM.GetIDisks")
		}

		detachDisks := []string{}
		for i, iDisk := range iDisks {
			if i != 0 {
				err = iVM.DetachDisk(ctx, iDisk.GetGlobalId())
				if err != nil {
					return "", errors.Wrap(err, "iVM.DetachDisk")
				}
				detachDisks = append(detachDisks, iDisk.GetGlobalId())
			}
		}

		var ieip cloudprovider.ICloudEIP
		if eip, err := guest.GetElasticIp(); eip != nil {
			ieip, err = eip.GetIEip(ctx)
			if err != nil {
				return "", errors.Wrap(err, "eip.GetIEip")
			}
			err = ieip.Dissociate()
			if err != nil {
				return "", errors.Wrap(err, "ieip.Dissociate")
			}
		} else if err != nil {
			return "", err
		}

		err = iVM.DeleteVM(ctx)
		if err != nil {
			return "", errors.Wrap(err, "iVM.DeleteVM")
		}
		err = cloudprovider.WaitDeleted(iVM, time.Second*5, time.Minute*10)
		if err != nil {
			return "", errors.Wrap(err, "WaitDeleted")
		}
		desc.DataDisks = []cloudprovider.SDiskInfo{}
		iVM, err = ihost.CreateVM(&desc)
		if err != nil {
			return "", errors.Wrap(err, "ihost.CreateVM")
		}

		instanceId = iVM.GetGlobalId()
		db.SetExternalId(guest, task.GetUserCred(), instanceId)
		initialState := driver.GetGuestInitialStateAfterCreate()
		log.Debugf("VMrebuildRoot %s new instance, wait status %s ...", iVM.GetGlobalId(), initialState)
		cloudprovider.WaitStatus(iVM, initialState, time.Second*5, time.Second*1800)

		opts := &cloudprovider.ServerStopOptions{
			IsForce: true,
		}
		iVM.StopVM(ctx, opts)

		if ieip != nil {
			conf := &cloudprovider.AssociateConfig{
				InstanceId:    instanceId,
				AssociateType: api.EIP_ASSOCIATE_TYPE_SERVER,
			}
			if err = ieip.Associate(conf); err != nil {
				return "", errors.Wrap(err, "eip.Associate")
			}
		}
		self.attachDisks(ctx, ihost, instanceId, detachDisks)

		iDisks, err = iVM.GetIDisks()
		if err != nil {
			return "", errors.Wrapf(err, "iVM.GetIDisks.AfterCreated")
		}

		for _, iDisk := range iDisks {
			return iDisk.GetGlobalId(), nil
		}
		return "", fmt.Errorf("failed to found new instance system disk")
	}()
	if err != nil {
		return nil, err
	}

	initialState := driver.GetGuestInitialStateAfterRebuild()
	log.Debugf("VMrebuildRoot %s new diskID %s, wait status %s ...", iVM.GetGlobalId(), diskId, initialState)
	err = cloudprovider.WaitStatus(iVM, initialState, time.Second*5, time.Second*1800)
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
			if idisks[0].GetGlobalId() == diskId {
				break
			}
			if waited > maxWaitSecs {
				return nil, fmt.Errorf("inconsistent sys disk id after rebuild root")
			}
			log.Debugf("current system disk id inconsistent %s != %s, try after 5 seconds", idisks[0].GetGlobalId(), diskId)
			time.Sleep(time.Second * 5)
			waited += 5
		}
	}

	data := fetchIVMinfo(desc, iVM, guest.Id, desc.Account, desc.Password, desc.PublicKey, "rebuild")

	return data, nil
}

func (self *SOpenStackGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_READY
}

func (self *SOpenStackGuestDriver) AllowReconfigGuest() bool {
	return true
}

func (self *SOpenStackGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	return false
}

func (self *SOpenStackGuestDriver) IsSupportMigrate() bool {
	return true
}

func (self *SOpenStackGuestDriver) IsSupportLiveMigrate() bool {
	return true
}
