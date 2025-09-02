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

type SKsyunGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SKsyunGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SKsyunGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_KSYUN
}

func (self *SKsyunGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_KSYUN
}

func (self *SKsyunGuestDriver) GetDefaultSysDiskBackend() string {
	return ""
}

func (self *SKsyunGuestDriver) GetMinimalSysDiskSizeGb() int {
	return 20
}

func (self *SKsyunGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PUBLIC_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_KSYUN
	keys.Brand = api.CLOUD_PROVIDER_KSYUN
	keys.Hypervisor = api.HYPERVISOR_KSYUN
	return keys
}

func (self *SKsyunGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
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

func (self *SKsyunGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_READY
}

func (self *SKsyunGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_READY
}

func (self *SKsyunGuestDriver) AllowReconfigGuest() bool {
	return true
}

func (self *SKsyunGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	return false
}

func (self *SKsyunGuestDriver) IsSupportPublicipToEip() bool {
	return false
}

func (self *SKsyunGuestDriver) IsSupportSetAutoRenew() bool {
	return false
}
