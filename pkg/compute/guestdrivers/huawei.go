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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SHuaweiGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SHuaweiGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SHuaweiGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_HUAWEI
}

func (self *SHuaweiGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_HUAWEI
}

func (self *SHuaweiGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PUBLIC_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_HUAWEI
	keys.Brand = api.CLOUD_PROVIDER_HUAWEI
	keys.Hypervisor = api.HYPERVISOR_HUAWEI
	return keys
}

func (self *SHuaweiGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_HUAWEI_SAS
}

func (self *SHuaweiGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 40
}

func (self *SHuaweiGuestDriver) GetStorageTypes() []string {
	return []string{
		api.STORAGE_HUAWEI_SATA,
		api.STORAGE_HUAWEI_SAS,
		api.STORAGE_HUAWEI_SSD,
		api.STORAGE_HUAWEI_GPSSD,
		api.STORAGE_HUAWEI_ESSD,
	}
}

func (self *SHuaweiGuestDriver) ChooseHostStorage(host *models.SHost, guest *models.SGuest, diskConfig *api.DiskConfig, storageIds []string) (*models.SStorage, error) {
	return chooseHostStorage(self, host, diskConfig.Backend, storageIds), nil
}

func (self *SHuaweiGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SHuaweiGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SHuaweiGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SHuaweiGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SHuaweiGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SHuaweiGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	if !utils.IsInStringArray(storage.StorageType, []string{api.STORAGE_HUAWEI_SATA, api.STORAGE_HUAWEI_SAS, api.STORAGE_HUAWEI_SSD}) {
		return fmt.Errorf("Cannot resize disk with unsupported volumes type %s", storage.StorageType)
	}

	return nil
}

func (self *SHuaweiGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SHuaweiGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_RUNNING
}

func (self *SHuaweiGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
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

func (self *SHuaweiGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	months := bc.GetMonths()
	if (months >= 1 && months <= 9) || (months == 12) || (months == 24) || (months == 36) {
		return true
	}

	return false
}

func (self *SHuaweiGuestDriver) IsNeedInjectPasswordByCloudInit() bool {
	return true
}

func (self *SHuaweiGuestDriver) IsSupportSetAutoRenew() bool {
	return false
}
