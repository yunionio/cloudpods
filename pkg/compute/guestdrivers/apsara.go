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
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SApsaraGuestDriver struct {
	SAliyunGuestDriver
}

func init() {
	driver := SApsaraGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SApsaraGuestDriver) DoScheduleSKUFilter() bool { return false }

func (self *SApsaraGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_APSARA
}

func (self *SApsaraGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_APSARA
}

func (self *SApsaraGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PRIVATE_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_APSARA
	keys.Brand = api.CLOUD_PROVIDER_APSARA
	keys.Hypervisor = api.HYPERVISOR_APSARA
	return keys
}

func (self *SApsaraGuestDriver) GetGuestInitialStateAfterCreate() string {
	return api.VM_READY
}

func (self *SApsaraGuestDriver) GetGuestInitialStateAfterRebuild() string {
	return api.VM_READY
}

func (self *SApsaraGuestDriver) AllowReconfigGuest() bool {
	return true
}

func (self *SApsaraGuestDriver) IsSupportedBillingCycle(bc billing.SBillingCycle) bool {
	return false
}

func (self *SApsaraGuestDriver) IsSupportPublicipToEip() bool {
	return false
}

func (self *SApsaraGuestDriver) IsSupportSetAutoRenew() bool {
	return false
}
