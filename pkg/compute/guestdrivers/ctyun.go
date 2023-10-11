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
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCtyunGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func (self *SCtyunGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_CTYUN
}

func (self *SCtyunGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_CTYUN
}

func (self *SCtyunGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
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

func (self *SCtyunGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PUBLIC_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_CTYUN
	keys.Brand = api.CLOUD_PROVIDER_CTYUN
	keys.Hypervisor = api.HYPERVISOR_CTYUN
	return keys
}

func (self *SCtyunGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_CTYUN_SAS
}

func (self *SCtyunGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_RUNNING}, nil
}

func (self *SCtyunGuestDriver) IsNeedRestartForResetLoginInfo() bool {
	return false
}

func (self *SCtyunGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 10
}

func (self *SCtyunGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_RUNNING
}

func (self *SCtyunGuestDriver) GetDetachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SCtyunGuestDriver) GetAttachDiskStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_RUNNING}, nil
}

func (self *SCtyunGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SCtyunGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY}, nil
}

func (self *SCtyunGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_READY
}

func (self *SCtyunGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		return fmt.Errorf("Cannot resize disk when guest in status %s", guest.Status)
	}
	if !utils.IsInStringArray(storage.StorageType, []string{api.STORAGE_CTYUN_SATA, api.STORAGE_CTYUN_SAS, api.STORAGE_CTYUN_SSD}) {
		return fmt.Errorf("Cannot resize disk with unsupported volumes type %s", storage.StorageType)
	}

	return nil
}

func init() {
	driver := SCtyunGuestDriver{}
	models.RegisterGuestDriver(&driver)
}
