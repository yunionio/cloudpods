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
	"fmt"

	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SHuaweiCloudStackGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SHuaweiCloudStackGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SHuaweiCloudStackGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_HUAWEI_CLOUD_STACK
}

func (self *SHuaweiCloudStackGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_HUAWEI_CLOUD_STACK
}

func (self *SHuaweiCloudStackGuestDriver) GetComputeQuotaKeys(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PRIVATE_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_HUAWEI_CLOUD_STACK
	keys.Brand = api.CLOUD_PROVIDER_HUAWEI_CLOUD_STACK
	keys.Hypervisor = api.HYPERVISOR_HUAWEI_CLOUD_STACK
	return keys
}

func (self *SHuaweiCloudStackGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_HUAWEI_SAS
}

func (self *SHuaweiCloudStackGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 40
}

func (self *SHuaweiCloudStackGuestDriver) GetStorageTypes() []string {
	return []string{api.STORAGE_HUAWEI_SATA, api.STORAGE_HUAWEI_SAS, api.STORAGE_HUAWEI_SSD}
}

func (self *SHuaweiCloudStackGuestDriver) ChooseHostStorage(host *models.SHost, guest *models.SGuest, diskConfig *api.DiskConfig, storageIds []string) (*models.SStorage, error) {
	return self.chooseHostStorage(self, host, diskConfig.Backend, storageIds), nil
}

func (self *SHuaweiCloudStackGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SHuaweiCloudStackGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SHuaweiCloudStackGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SHuaweiCloudStackGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SHuaweiCloudStackGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SHuaweiCloudStackGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	if !utils.IsInStringArray(storage.StorageType, []string{api.STORAGE_HUAWEI_SATA, api.STORAGE_HUAWEI_SAS, api.STORAGE_HUAWEI_SSD}) {
		return fmt.Errorf("Cannot resize disk with unsupported volumes type %s", storage.StorageType)
	}

	return nil
}

func (self *SHuaweiCloudStackGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SHuaweiCloudStackGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_RUNNING
}

func (self *SHuaweiCloudStackGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
	return cloudprovider.SInstanceCapability{
		Hypervisor: self.GetHypervisor(),
		Provider:   self.GetProvider(),
		DefaultAccount: cloudprovider.SDefaultAccount{
			Linux: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_DEFAULT_LINUX_LOGIN_USER,
			},
			Windows: cloudprovider.SOsDefaultAccount{
				DefaultAccount: api.VM_DEFAULT_WINDOWS_LOGIN_USER,
			},
		},
		Storages: cloudprovider.Storage{
			DataDisk: []cloudprovider.StorageInfo{
				cloudprovider.StorageInfo{StorageType: api.STORAGE_HUAWEI_SSD, MaxSizeGb: 32768, MinSizeGb: 10, StepSizeGb: 1, Resizable: true},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_HUAWEI_SATA, MaxSizeGb: 32768, MinSizeGb: 10, StepSizeGb: 1, Resizable: true},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_HUAWEI_SAS, MaxSizeGb: 32768, MinSizeGb: 10, StepSizeGb: 1, Resizable: true},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_HUAWEI_GPSSD, MaxSizeGb: 32768, MinSizeGb: 10, StepSizeGb: 1, Resizable: true},
			},
			SysDisk: []cloudprovider.StorageInfo{
				cloudprovider.StorageInfo{StorageType: api.STORAGE_HUAWEI_SSD, MaxSizeGb: 1024, MinSizeGb: 40, StepSizeGb: 1, Resizable: false},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_HUAWEI_SATA, MaxSizeGb: 1024, MinSizeGb: 40, StepSizeGb: 1, Resizable: false},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_HUAWEI_SAS, MaxSizeGb: 1024, MinSizeGb: 40, StepSizeGb: 1, Resizable: false},
				cloudprovider.StorageInfo{StorageType: api.STORAGE_HUAWEI_GPSSD, MaxSizeGb: 1024, MinSizeGb: 40, StepSizeGb: 1, Resizable: false},
			},
		},
	}
}

func (self *SHuaweiCloudStackGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	months := bc.GetMonths()
	if (months >= 1 && months <= 9) || (months == 12) || (months == 24) || (months == 36) {
		return true
	}

	return false
}

func (self *SHuaweiCloudStackGuestDriver) IsNeedInjectPasswordByCloudInit(desc *cloudprovider.SManagedVMCreateConfig) bool {
	return true
}

func (self *SHuaweiCloudStackGuestDriver) IsSupportSetAutoRenew() bool {
	return true
}
