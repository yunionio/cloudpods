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

type SVolcengineGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SVolcengineGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SVolcengineGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_VOLCENGINE
}

func (self *SVolcengineGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_VOLCENGINE
}

func (self *SVolcengineGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PUBLIC_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_VOLCENGINE
	keys.Brand = api.CLOUD_PROVIDER_VOLCENGINE
	keys.Hypervisor = api.HYPERVISOR_VOLCENGINE
	return keys
}

func (self *SVolcengineGuestDriver) GetDefaultSysDiskBackend() string {
	return ""
}

func (self *SVolcengineGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 40
}

func (self *SVolcengineGuestDriver) GetStorageTypes() []string {
	return []string{}
}

func (self *SVolcengineGuestDriver) ChooseHostStorage(host *models.SHost, guest *models.SGuest, diskConfig *api.DiskConfig, storageIds []string) (*models.SStorage, error) {
	return chooseHostStorage(self, host, diskConfig.Backend, storageIds), nil
}

func (self *SVolcengineGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SVolcengineGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SVolcengineGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SVolcengineGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SVolcengineGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SVolcengineGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SVolcengineGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_RUNNING
}

func (self *SVolcengineGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
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
		Storages: cloudprovider.Storage{
			DataDisk: []cloudprovider.StorageInfo{
				{StorageType: api.STORAGE_VOLC_CLOUD_PTSSD, MaxSizeGb: 8192, MinSizeGb: 20, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_VOLC_CLOUD_PL0, MaxSizeGb: 32768, MinSizeGb: 20, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_VOLC_CLOUD_FLEXPL, MaxSizeGb: 32768, MinSizeGb: 20, StepSizeGb: 1, Resizable: true},
			},
			SysDisk: []cloudprovider.StorageInfo{
				{StorageType: api.STORAGE_VOLC_CLOUD_PTSSD, MaxSizeGb: 500, MinSizeGb: 40, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_VOLC_CLOUD_PL0, MaxSizeGb: 2048, MinSizeGb: 40, StepSizeGb: 1, Resizable: true},
				{StorageType: api.STORAGE_VOLC_CLOUD_FLEXPL, MaxSizeGb: 2048, MinSizeGb: 40, StepSizeGb: 1, Resizable: true},
			},
		},
	}
}

func (self *SVolcengineGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	months := bc.GetMonths()
	if (months >= 1 && months <= 9) || (months == 12) || (months == 24) || (months == 36) {
		return true
	}

	return false
}

func (self *SVolcengineGuestDriver) IsNeedInjectPasswordByCloudInit() bool {
	return true
}

func (self *SVolcengineGuestDriver) IsSupportSetAutoRenew() bool {
	return false
}

func (self *SVolcengineGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		return fmt.Errorf("cannot resize disk when guest in status %s", guest.Status)
	}
	return nil
}
