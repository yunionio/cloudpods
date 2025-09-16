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
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCucloudGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SCucloudGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SCucloudGuestDriver) DoScheduleSKUFilter() bool { return false }

func (self *SCucloudGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_CUCLOUD
}

func (self *SCucloudGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_CUCLOUD
}

func (self *SCucloudGuestDriver) GetDefaultSysDiskBackend() string {
	return ""
}

func (self *SCucloudGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 20
}

func (self *SCucloudGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PUBLIC_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_CUCLOUD
	keys.Brand = api.CLOUD_PROVIDER_CUCLOUD
	keys.Hypervisor = api.HYPERVISOR_CUCLOUD
	return keys
}

func (self *SCucloudGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
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

func (self *SCucloudGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_READY
}

func (self *SCucloudGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_READY
}

func (self *SCucloudGuestDriver) AllowReconfigGuest() bool {
	return true
}

func (self *SCucloudGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	return false
}

func (self *SCucloudGuestDriver) IsSupportPublicipToEip() bool {
	return false
}

func (self *SCucloudGuestDriver) IsSupportSetAutoRenew() bool {
	return false
}
