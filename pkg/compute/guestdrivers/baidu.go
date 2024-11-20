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

type SBaiduGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SBaiduGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SBaiduGuestDriver) DoScheduleSKUFilter() bool { return false }

func (self *SBaiduGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_BAIDU
}

func (self *SBaiduGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_BAIDU
}

func (self *SBaiduGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_BAIDU_SSD
}

func (self *SBaiduGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 20
}

func (self *SBaiduGuestDriver) ChooseHostStorage(host *models.SHost, guest *models.SGuest, diskConfig *api.DiskConfig, storageIds []string) (*models.SStorage, error) {
	return chooseHostStorage(self, host, diskConfig.Backend, storageIds), nil
}

func (self *SBaiduGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PUBLIC_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_BAIDU
	keys.Brand = api.CLOUD_PROVIDER_BAIDU
	keys.Hypervisor = api.HYPERVISOR_BAIDU
	return keys
}

func (self *SBaiduGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
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

func (self *SBaiduGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SBaiduGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SBaiduGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SBaiduGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SBaiduGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SBaiduGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_RUNNING
}

func (self *SBaiduGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SBaiduGuestDriver) AllowReconfigGuest() bool {
	return true
}

func (self *SBaiduGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	return true
}

func (self *SBaiduGuestDriver) IsSupportPublicipToEip() bool {
	return false
}

func (self *SBaiduGuestDriver) IsSupportSetAutoRenew() bool {
	return false
}

func (self *SBaiduGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		return fmt.Errorf("cannot resize disk when guest in status %s", guest.Status)
	}
	return nil
}
